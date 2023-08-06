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
	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/ava-labs/avalanchego/utils/hashing"
	"github.com/ava-labs/avalanchego/vms/platformvm/blocks"
	"github.com/ava-labs/avalanchego/vms/proposervm/block"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	"github.com/ava-labs/coreth/plugin/evm"
	_ "github.com/mattn/go-sqlite3"
	"github.com/multisig-labs/slurp/pkg/db"
	"github.com/multisig-labs/slurp/pkg/uploads3"
	"github.com/tidwall/gjson"

	"github.com/jxskiss/mcli"
	"github.com/multisig-labs/slurp/pkg/version"
)

func main() {
	defer handlePanic()
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
var factory secp256k1.Factory

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
	handleError(err)
}

func pchainCmd() {
	args := struct {
		StartIdx   uint64 `cli:"startIdx, starting index number to fetch"`
		NumToFetch int64  `cli:"#R, numToFetch, number of blocks to fetch"`
	}{}
	mcli.Parse(&args, mcli.WithErrorHandling(flag.ExitOnError))

	err := fetchBlocksP(gargs.DbFile, gargs.NodeURL, args.StartIdx, args.NumToFetch, gargs.BatchSize)
	handleError(err)
}

func processPCmd() {
	args := struct {
		StartIdx     uint64 `cli:"startIdx, starting index number to process"`
		NumToProcess int64  `cli:"#R, numToProcess, number of blocks to process"`
	}{}
	mcli.Parse(&args, mcli.WithErrorHandling(flag.ExitOnError))

	err := processBlocksP(gargs.DbFile, args.StartIdx, args.NumToProcess)
	handleError(err)
}

func processBlocksP(dbFileName string, startIdx uint64, numToProcess int64) error {
	ctx := context.Background()
	mainnetCtx := genMainnetCtx()

	_, queries := openDB(dbFileName)

	for i := int64(0); i < numToProcess; i++ {
		idx := int64(startIdx) + i
		if idx%1000 == 0 {
			fmt.Println("Processing idx:", idx)
		}

		// If we already processed this block, skip it
		if _, err := queries.GetTxP(ctx, idx); err != sql.ErrNoRows {
			continue
		}

		dbBlk, err := queries.GetRawBlockP(ctx, idx)
		if err != nil {
			return fmt.Errorf("error GetRawBlockP idx %d: %v", idx, err)
		}

		// Get both the block and the marshalled json
		blk, js, err := decodeBlock(mainnetCtx, dbBlk.Bytes)
		if err != nil {
			return fmt.Errorf("error decodeBlock idx %d: %v", idx, err)
		}

		height := blk.Height()
		// FYI Not all blocks have txs, and some blocks have more than one tx
		// We are inserting the marshaled JSON for each tx and letting SQLite destructure via generated columns
		txs := gjson.Get(js, "tx*")
		if len(txs.Array()) > 0 {
			for j, tx := range txs.Array() {
				txid := tx.Get("id").String()
				txjs := tx.Get("unsignedTx").String()
				ts := gjson.Get(txjs, "time").Int()
				unsignedBytes := blk.Txs()[j].Unsigned.Bytes()
				unsignedBytesHex, err := formatting.Encode(formatting.Hex, unsignedBytes)
				if err != nil {
					fmt.Printf("warning unable to encode unsignedBytesHex txid: %s err: %v\n", txid, err)
					continue
				}

				// Look for signatures, and recover public key
				sigBytes := [65]byte{}
				sigBytesHex := ""
				recoveredAddrP := ""
				recoveredAddrC := ""

				creds := blk.Txs()[j].Creds
				if len(creds) > 0 {
					sigBytes = blk.Txs()[j].Creds[0].(*secp256k1fx.Credential).Sigs[0]
					sigBytesHex, _ = formatting.Encode(formatting.Hex, sigBytes[:])
					recoveredAddrP, recoveredAddrC, err = recoverAddrs(unsignedBytes, sigBytes)
					if err != nil {
						fmt.Printf("warning unable to recoverAddrs idx: %d txid: %s err: %v\n", idx, txid, err)
					}
				}

				p := db.CreateTxPParams{
					Idx:           idx,
					ID:            txid,
					Height:        int64(height),
					BlockID:       blk.ID().String(),
					TypeID:        parseTypeID(blk.Txs()[j].Bytes()),
					Ts:            int64(ts),
					UnsignedTx:    txjs,
					UnsignedBytes: unsignedBytesHex,
					SigBytes:      sigBytesHex,
					SignerAddrP:   recoveredAddrP,
					SignerAddrC:   recoveredAddrC,
				}
				err = queries.CreateTxP(ctx, p)
				if err != nil {
					return fmt.Errorf("error CreateTxP idx: %d txid: %s %v", idx, txid, err)
				}
			}
		}
	}

	return nil
}

// Recover the P and C chain addrs from a tx sig
func recoverAddrs(unsignedBytes []byte, sigBytes [65]byte) (string, string, error) {
	txHash := hashing.ComputeHash256(unsignedBytes)
	pk, err := factory.RecoverHashPublicKey(txHash, sigBytes[:])
	if err != nil {
		return "", "", err
	}
	recoveredAddrP, err := address.FormatBech32("avax", pk.Address().Bytes())
	if err != nil {
		return "", "", err
	}
	recoveredAddrC := evm.PublicKeyToEthAddress(pk).String()
	return recoveredAddrP, recoveredAddrC, nil
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
		handleError(err)

		for i, bk := range batchOfBlocks {
			idx := batchStartIdx + uint64(i)
			err := queries.CreateRawBlockP(ctx, db.CreateRawBlockPParams{
				Idx:   int64(idx),
				Bytes: bk.Bytes,
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
	handleError(err)
	_, err = dbFile.Exec("PRAGMA optimize;PRAGMA foreign_keys=ON;PRAGMA journal_mode=WAL;")
	handleError(err)
	return dbFile, db.New(dbFile)
}

// CodecID is first 4 bytes, typeID is next 2 bytes
func parseTypeID(b []byte) int64 {
	var result int64 = int64(b[4])<<8 | int64(b[5])
	return result
}

func handleError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func handlePanic() {
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
