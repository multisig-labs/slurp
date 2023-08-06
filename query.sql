-- name: GetBlockP :one
SELECT * FROM blocks_p
WHERE idx = ? LIMIT 1;

-- name: ListBlocksP :many
SELECT * FROM blocks_p;

-- name: CreateBlockP :exec
INSERT OR IGNORE INTO blocks_p (
  idx, id, bytes, decoded, type_id, height, ts, parent_id
) VALUES (
  ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: UpdateBlockP :exec
UPDATE blocks_p
SET 
  decoded = ?,
  type_id = ?,
  height = ?,
  ts = ?,
  parent_id = ?
WHERE idx = ?;

-- name: CreateTxP :exec
INSERT OR IGNORE INTO txs_p (
  id, block_id, type_id, unsigned_tx
) VALUES (
  ?, ?, ?, ?
);

