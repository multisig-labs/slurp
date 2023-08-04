# Slurp

Use the AvalancheGo Index API to slurp blocks into a sqlite database.

```
brew install sqlc
just create-db
just build
bin/slurp pchain --node-url https://indexer-demo.avax.network 0 100000
```

## Schema

```
CREATE TABLE blocks_p (
  height integer NOT NULL, -- integer height of the block
  id text NOT NULL,     -- base58 id of the block
  ts integer NOT NULL,  -- timestamp the node accepted the block
  bytes blob            -- raw bytes of the block
);
```

The idea would be to first grab all raw P-chain blocks, then have a seperate command to go through those and deserialize them from the bytes and make new tables with the data as necessary.
