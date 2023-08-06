package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sync"

	"runtime/debug"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/indexer"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/vms/platformvm/blocks"
	"github.com/ava-labs/avalanchego/vms/proposervm/block"
	_ "github.com/mattn/go-sqlite3"
	"github.com/multisig-labs/slurp/pkg/db"
	"github.com/multisig-labs/slurp/pkg/uploads3"
	"github.com/tidwall/gjson"

	"github.com/jxskiss/mcli"
	"github.com/multisig-labs/slurp/pkg/version"
)

func main() {
	defer panicHandler()
	gargs = &GlobalArgs{}
	mcli.SetGlobalFlags(gargs)
	mcli.Add("pchain", pchainCmd, "Slurp the P-Chain using Index API")
	mcli.Add("process-p", processPCmd, "Process the raw P-Chain blocks")
	mcli.Add("upload", uploadS3Cmd, "Upload file to S3 bucket")
	mcli.AddHelp()
	mcli.AddCompletion()
	mcli.Run()
}

var gargs *GlobalArgs

type GlobalArgs struct {
	NodeURL   string `cli:"--node-url, Avalanche node URL" default:"http://localhost:9650"`
	DbFile    string `cli:"--db, SQLite database file name" default:"slurp.db"`
	BatchSize int64  `cli:"--batch, Batch size to use when querying API" default:"1000"`
}

func uploadS3Cmd() {
	var awsKey, awsSecret string
	var found bool

	if awsKey, found = os.LookupEnv("AWS_ACCESS_KEY_ID"); !found {
		panic("Missing ENV AWS_ACCESS_KEY_ID")
	}
	if awsSecret, found = os.LookupEnv("AWS_SECRET_ACCESS_KEY"); !found {
		panic("Missing ENV AWS_SECRET_ACCESS_KEY")
	}

	args := struct {
		Region   string `cli:"region, S3 region"`
		Bucket   string `cli:"bucket, S3 bucket"`
		Filename string `cli:"filename, Filename to upload"`
	}{}
	mcli.Parse(&args, mcli.WithErrorHandling(flag.ExitOnError))

	err := uploads3.UploadS3(awsKey, awsSecret, args.Region, args.Bucket, args.Filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func pchainCmd() {
	args := struct {
		StartIdx   uint64 `cli:"startIdx, starting index number to fetch"`
		NumToFetch int64  `cli:"#R, NumToFetch, number of blocks to fetch"`
	}{}
	mcli.Parse(&args, mcli.WithErrorHandling(flag.ExitOnError))

	err := fetchBlocksP(gargs.DbFile, gargs.NodeURL, args.StartIdx, args.NumToFetch, gargs.BatchSize)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func processPCmd() {
	args := struct {
		StartIdx     uint64 `cli:"startIdx, starting index number to process"`
		NumToProcess int64  `cli:"#R, numToProcess, number of blocks to process"`
	}{}
	mcli.Parse(&args, mcli.WithErrorHandling(flag.ExitOnError))

	err := processBlocksP(gargs.DbFile, gargs.NodeURL, args.StartIdx, args.NumToProcess)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func processBlocksP(dbFileName string, nodeURL string, startIdx uint64, numToProcess int64) error {
	ctx := context.Background()
	mainnetCtx := genMainnetCtx()

	_, queries := openDB(dbFileName)

	for i := int64(0); i < numToProcess; i++ {
		idx := int64(startIdx) + i
		if idx%1000 == 0 {
			fmt.Println("Processing", idx)
		}
		dbBlk, err := queries.GetBlockP(ctx, idx)
		if err != nil {
			return err
		}

		// Get both the block and the marshalled json
		blk, js, err := decodeBlock(mainnetCtx, dbBlk.Bytes)
		if err != nil {
			fmt.Printf("error idx %d: %v\n", idx, err)
			return err
		}

		if dbBlk.Decoded == 0 {
			// Save additional info decoded from bytes to block_p table
			height := blk.Height()
			parentId := blk.Parent().String()
			typeId := parseTypeID(dbBlk.Bytes)
			// Only Banff blocks have time, so just use json to grab it if it exists
			ts := gjson.Get(js, "time").Int()
			p := db.UpdateBlockPParams{
				Decoded:  1,
				TypeID:   sql.NullInt64{Valid: true, Int64: typeId},
				Height:   sql.NullInt64{Valid: true, Int64: int64(height)},
				Ts:       sql.NullInt64{Valid: true, Int64: ts},
				ParentID: sql.NullString{Valid: true, String: parentId},
				Idx:      idx,
			}

			err = queries.UpdateBlockP(ctx, p)
			if err != nil {
				fmt.Printf("error updating idx %d: %v\n", idx, err)
				return err
			}
		}

		// We are inserting the marshaled JSON and letting SQLite destructure via generated columns
		txs := gjson.Get(js, "tx*")
		if len(txs.Array()) > 0 {
			for j, tx := range txs.Array() {
				txid := tx.Get("id").String()
				txjs := tx.Get("unsignedTx").String()
				p := db.CreateTxPParams{
					ID:         txid,
					BlockID:    dbBlk.ID,
					TypeID:     parseTypeID(blk.Txs()[j].Bytes()),
					UnsignedTx: txjs,
				}
				err := queries.CreateTxP(ctx, p)
				if err != nil {
					fmt.Printf("error creating txid %s blkid %s %v\n", txid, dbBlk.ID, err)
				}
			}
		}
	}

	return nil
}

func decodeBlock(ctx *snow.Context, b []byte) (blocks.Block, string, error) {
	decoded := decodeProposerBlock(b)

	blk, js, err := decodeInnerBlock(ctx, decoded)
	if err != nil {
		return blk, "", err
	}
	return blk, string(js), nil
}

// Tries to decode as proposal block (post-Banff) if it fails just return the original bytes
func decodeProposerBlock(b []byte) []byte {
	innerBlk, err := block.Parse(b)
	if err != nil {
		return b
	}
	return innerBlk.Block()
}

func decodeInnerBlock(ctx *snow.Context, b []byte) (blocks.Block, string, error) {
	res, err := blocks.Parse(blocks.GenesisCodec, b)
	if err != nil {
		return res, "", fmt.Errorf("blocks.Parse error: %w", err)
	}

	res.InitCtx(ctx)
	j, err := json.Marshal(res)
	if err != nil {
		return res, "", fmt.Errorf("json.Marshal error: %w", err)
	}
	return res, string(j), nil
}

func fetchBlocksP(dbFileName string, nodeURL string, startIdx uint64, numToFetch int64, batchSize int64) error {
	ctx := context.Background()
	_, queries := openDB(dbFileName)
	c := indexer.NewClient(nodeURL + "/ext/index/P/block")

	totalBatches := (numToFetch / batchSize)
	if numToFetch%batchSize != 0 {
		totalBatches = totalBatches + 1
	}
	fmt.Printf("numToFetch: %d batchSize: %d totalBatches: %d startIdx: %d\n", numToFetch, batchSize, totalBatches, startIdx)

	for batch := int64(0); batch < totalBatches; batch++ {
		batchStartIdx := startIdx + (uint64(batchSize) * uint64(batch))
		fmt.Printf("batch: %d  batchStartIdx: %d\n", batch, batchStartIdx)
		batchOfBlocks, err := c.GetContainerRange(context.Background(), batchStartIdx, int(batchSize))
		if err != nil {
			fmt.Printf("error %v\n", err)
			os.Exit(2)
		}

		for i, bk := range batchOfBlocks {
			idx := batchStartIdx + uint64(i)
			err := queries.CreateBlockP(ctx, db.CreateBlockPParams{
				Idx:     int64(idx),
				ID:      bk.ID.String(),
				Bytes:   bk.Bytes,
				Decoded: 0,
			})
			if err != nil {
				fmt.Printf("error inserting block: %v\n", err)
			}
		}
	}

	return nil
}

// Simple context so that Marshal works
func genMainnetCtx() *snow.Context {
	pChainID, _ := ids.FromString("11111111111111111111111111111111LpoYY")
	xChainID, _ := ids.FromString("2oYMBNV4eNHyqk2fjjV5nVQLDbtmNJzq5s3qs3Lo6ftnC6FByM")
	cChainID, _ := ids.FromString("2q9e4r6Mu3U68nU1fYjgbR6JvwrRx36CohpAX5UQxse55x1Q5")
	avaxAssetID, _ := ids.FromString("FvwEAhmxKfeiG8SnEvq42hc6whRyY3EFYAvebMqDNDGCgxN5Z")
	lookup := ids.NewAliaser()
	lookup.Alias(xChainID, "X")
	lookup.Alias(cChainID, "C")
	lookup.Alias(pChainID, "P")
	c := &snow.Context{
		NetworkID:   1,
		SubnetID:    [32]byte{},
		ChainID:     [32]byte{},
		NodeID:      [20]byte{},
		XChainID:    xChainID,
		CChainID:    cChainID,
		AVAXAssetID: avaxAssetID,
		Lock:        sync.RWMutex{},
		BCLookup:    lookup,
	}
	return c
}

func openDB(dbFileName string) (*sql.DB, *db.Queries) {
	dbFile, err := sql.Open("sqlite3", dbFileName)
	if err != nil {
		panic(err)
	}
	_, err = dbFile.Exec("PRAGMA optimize;PRAGMA foreign_keys=ON;PRAGMA journal_mode=WAL;")
	if err != nil {
		panic(err)
	}
	return dbFile, db.New(dbFile)
}

// CodecID is first 4 bytes, typeID is next 2 bytes
func parseTypeID(b []byte) int64 {
	var result int64 = int64(b[4])<<8 | int64(b[5])
	return result
}

func panicHandler() {
	if panicPayload := recover(); panicPayload != nil {
		stack := string(debug.Stack())
		fmt.Fprintln(os.Stderr, "================================================================================")
		fmt.Fprintln(os.Stderr, "            Fatal error. Sorry! You found a bug.")
		fmt.Fprintln(os.Stderr, "    Please copy all of this info into an issue at")
		fmt.Fprintln(os.Stderr, "     https://github.com/multisig-labs/slurp")
		fmt.Fprintln(os.Stderr, "================================================================================")
		fmt.Fprintf(os.Stderr, "Version:           %s\n", version.Version)
		fmt.Fprintf(os.Stderr, "Build Date:        %s\n", version.BuildDate)
		fmt.Fprintf(os.Stderr, "Git Commit:        %s\n", version.GitCommit)
		fmt.Fprintf(os.Stderr, "Go Version:        %s\n", version.GoVersion)
		fmt.Fprintf(os.Stderr, "OS / Arch:         %s\n", version.OsArch)
		fmt.Fprintf(os.Stderr, "Panic:             %s\n\n", panicPayload)
		fmt.Fprintln(os.Stderr, stack)
		os.Exit(1)
	}
}
