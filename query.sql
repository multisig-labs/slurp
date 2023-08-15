-- name: GetRawBlockP :one
SELECT * FROM raw_blocks_p
WHERE idx = ? LIMIT 1;

-- name: CreateRawBlockP :exec
INSERT OR IGNORE INTO raw_blocks_p (
  idx, bytes
) VALUES (
  ?, ?
);

-- name: GetTxP :one
SELECT * FROM txs_p
WHERE idx = ? LIMIT 1;

-- name: CreateTxP :exec
INSERT OR IGNORE INTO txs_p (
  idx, id, height, block_id, type_id, unsigned_tx, unsigned_bytes, sig_bytes, signer_addr_p, signer_addr_c, ts
) VALUES (
  ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

