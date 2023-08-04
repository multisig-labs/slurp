package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"

	"runtime/debug"

	"github.com/ava-labs/avalanchego/indexer"
	_ "github.com/mattn/go-sqlite3"
	"github.com/multisig-labs/slurp/pkg/db"

	"github.com/jxskiss/mcli"
	"github.com/multisig-labs/slurp/pkg/version"
)

func main() {
	defer panicHandler()
	gargs = &GlobalArgs{}
	mcli.SetGlobalFlags(gargs)
	mcli.Add("pchain", pchainCmd, "Slurp the P-Chain")
	mcli.AddHelp()
	mcli.AddCompletion()
	mcli.Run()
}

var gargs *GlobalArgs

type GlobalArgs struct {
	NodeURL string `cli:"--node-url, Avalanche node URL" default:"http://localhost:9650"`
	DbFile  string `cli:"--db, SQLite database file name" default:"slurp.db"`
}

func pchainCmd() {
	args := struct {
		StartHeight uint64 `cli:"startHeight, starting block height to fetch (first block is height 0)"`
		NumToFetch  int    `cli:"#R, numToFetch, number of blocks to fetch"`
	}{}
	_, err := mcli.Parse(&args, mcli.WithErrorHandling(flag.ExitOnError))
	if err != nil && err != flag.ErrHelp {
		fmt.Printf("mcli.Parse error: %v", err)
		fmt.Println()
	}

	err = fetchBlocks(gargs.DbFile, gargs.NodeURL, args.StartHeight, int64(args.NumToFetch))
	if err != nil {
		fmt.Printf("%v\n", err)
	}
}

// Height is off by one when compared to explorer. Weird.
// https://subnets.avax.network/p-chain/block/1 => 4AqeFPxtTW4B5D6oR8gRZTvRKnnqkUWiV6mUNZxjUMbQKYWpi
// But curl index.getContainerByIndex 0 => 4AqeFPxtTW4B5D6oR8gRZTvRKnnqkUWiV6mUNZxjUMbQKYWpi
func fetchBlocks(dbFileName string, nodeURL string, startHeight uint64, numToFetch int64) error {
	// GetContainerRange can get a bunch of blocks at once, not sure what the upper limit is.
	const batchSize = int64(1000)

	ctx := context.Background()

	dbFile, err := sql.Open("sqlite3", dbFileName)
	if err != nil {
		return err
	}
	queries := db.New(dbFile)

	c := indexer.NewClient(nodeURL + "/ext/index/P/block")

	totalBatches := (numToFetch / batchSize)
	if numToFetch%batchSize != 0 {
		totalBatches = totalBatches + 1
	}
	fmt.Printf("numToFetch: %d batchSize: %d totalBatches: %d startHeight: %d\n", numToFetch, batchSize, totalBatches, startHeight)

	for batch := int64(0); batch < totalBatches; batch++ {
		batchStartHeight := startHeight + (uint64(batchSize) * uint64(batch))
		fmt.Printf("batch: %d  batchStartHeight: %d\n", batch, batchStartHeight)
		x, err := c.GetContainerRange(context.Background(), batchStartHeight, int(batchSize))
		if err != nil {
			fmt.Printf("%v", err)
			os.Exit(2)
		}

		for i, bk := range x {
			h := batchStartHeight + uint64(i)
			_, err := queries.CreateBlock(ctx, db.CreateBlockParams{
				Height: int64(h),
				ID:     bk.ID.String(),
				Ts:     bk.Timestamp,
				Bytes:  bk.Bytes,
			})
			if err != nil {
				fmt.Printf("error inserting block: %v\n", err)
			}
		}
	}

	return nil
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
