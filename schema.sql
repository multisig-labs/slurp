CREATE TABLE blocks_p (
  height integer PRIMARY KEY,
  id text NOT NULL,
  ts integer NOT NULL,
  bytes blob
) STRICT, WITHOUT ROWID;

CREATE UNIQUE INDEX idx_blocks_id ON blocks_p(id);
