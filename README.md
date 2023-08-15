# Slurp

Use the AvalancheGo Index API to slurp blocks into a sqlite database.

A SQLite DB file containing the entire P-Chain (as of 8-2023) can be found [here](https://gogopool.s3.amazonaws.com/slurp-mainnet.db.7z) (~7.5G zipped)

## Usage

```
# We require SQLite 3.42+ (built-in Mac one is too old)
brew install sqlc just sqlite3

# Need to activate the brew sqlite3 version with something like this
# echo 'export PATH="/opt/homebrew/opt/sqlite/bin:$PATH"' >> ~/.zshrc

just build
just create-db

# Slurp 7.5M raw blocks (roughly until 08-2023)
bin/slurp pchain --node-url https://indexer-demo.avax.network 0 7500000

# Now process the DB to parse the blocks and extract txs
bin/slurp process-p 0 7500000
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
