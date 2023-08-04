-- name: GetBlock :one
SELECT * FROM blocks_p
WHERE height = ? LIMIT 1;

-- name: ListBlocks :many
SELECT * FROM blocks_p;

-- name: CreateBlock :one
INSERT INTO blocks_p (
  height, id, ts, bytes
) VALUES (
  ?, ?, ?, ?
)
RETURNING *;

