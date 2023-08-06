# Slurp

Use the AvalancheGo Index API to slurp blocks into a sqlite database.

A SQLite DB file containing the entire P-Chain (as of 8-2023) can be found [here](https://gogopool.s3.amazonaws.com/slurp-mainnet-processed.db.7z) (~5G zipped)

## Data

The `raw_blocks_p` table has a row for every index number, which is just an incrementing integer the indexer creates when indexing the chain. Once this table is populated with every block, we then process each block and save the transactions into the `txs_p` table.

## Usage

```
# We require SQLite 3.42+ (built-in Mac one is too old)
brew install sqlc just sqlite3

# Need to activate the brew sqlite3 version with something like this
# echo 'export PATH="/opt/homebrew/opt/sqlite/bin:$PATH"' >> ~/.zshrc

just build
just create-db

# Find out latest chain height
curl --request POST \
  --url https://api.avax.network/ext/bc/P \
  --header 'content-type: application/json' \
  --data '{"id": 0,"jsonrpc": "2.0","method": "platform.getHeight","params": {}}'

# Slurp the raw blocks
bin/slurp pchain --node-url https://indexer-demo.avax.network 0 7827793

(NOTE: We were using the indexer demo above, since it was faster, but it seems to be gone now, so we should really change back to using an archive node)

# Now process the DB to parse the blocks starting at idx 0 and processing 7827793 blocks into transactions
bin/slurp process-p 0 7827793
```

## SQL FTW

In keeping with the quick-and-dirty theme of this project, we use the `avalanchego` packages to marshal a block into JSON, then insert the JSON into a SQLite `text` column, and leverage generated columns and json functions to break out the pieces we are interested in into their own columns.

## Interesting Queries

```sql
select count(*) count, memo from txs_p
group by memo
order by count desc;

select count(*) count, node_id
from txs_p
where type_id = 14 -- AddValidatorTx
group by node_id
order by count desc;

select count(*) count, rewards_addr
from txs_p
where type_id = 14 -- AddValidatorTx
group by rewards_addr
order by count desc;

select count(*) as count, signer_addr_p
from txs_p
where signer_addr_p != ""
group by signer_addr_p
order by count desc;

select count(*) as count, signer_addr_c
from txs_p
where signer_addr_c != ""
group by signer_addr_c
order by count desc;

```
