curl --silent -H 'accept: application/json' \
'https://glacier-api.avax.network/v1/networks/mainnet/blockchains/c-chain/transactions?txTypes=ImportTx&startTimestamp=1685100249&endTimestamp=1689800249&pageSize=100&sortOrder=asc' \
| jq '.transactions[] | select(.txType=="ImportTx") | {paddy: .consumedUtxos[0].addresses[0], caddy: .evmOutputs[0].toAddress}'

Multiple Txs per block: height 7287085
https://subnets.avax.network/p-chain/block/gvcneuaZEjV2Pxkyis9gViK2bff9X2EWSChZtWb6XutcLNnVH

5336531
https://subnets.avax.network/p-chain/block/btwFRqzEz1ZkPFghksDXoAjkrofpuTefXp41VCav3EUt3uRuQ
