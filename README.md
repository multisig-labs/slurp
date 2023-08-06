# Slurp

Use the AvalancheGo Index API to slurp blocks into a sqlite database.

A SQLite DB file containing the entire P-Chain (as of 8-2023) can be found [here](http://gogopool.s3.amazonaws.com/slurp-mainnet.db.7z) (~7.5G zipped)

## Usage

```
brew install sqlc
just build
just create-db
# Slurp 7.5M raw blocks (roughly until 08-2023)
bin/slurp pchain --node-url https://indexer-demo.avax.network 0 7500000
# Now process the DB to parse the blocks and extract txs
bin/slurp process-p 0 7500000
```

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

select count(*) count, rewards_addrs
from txs_p
where type_id = 14 -- AddValidatorTx
group by rewards_addrs
order by count desc

```
