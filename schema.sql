CREATE TABLE raw_blocks_p 
-- Raw P-Chain block bytes by index, other tables are generated from this one
(
  idx integer PRIMARY KEY, -- index of the block in a particular node's index
  bytes blob NOT NULL -- block (may be a container block)
) STRICT, WITHOUT ROWID;

CREATE TABLE txs_p (
  idx integer PRIMARY KEY, -- index of this txs block in a particular node's index
  id text NOT NULL,
  height integer NOT NULL,
  block_id text NOT NULL,
  type_id integer NOT NULL,
  unsigned_tx text NOT NULL, -- JSON of the unsigned_tx key
  unsigned_bytes text NOT NULL DEFAULT '', -- Hex encoded
	sig_bytes text NOT NULL DEFAULT '', -- Hex encoded
	signer_addr_p text NOT NULL DEFAULT '',
	signer_addr_c text NOT NULL DEFAULT '',
  ts integer NOT NULL DEFAULT 0,
  memo text GENERATED ALWAYS AS (cast(unhex(substr(json_extract(unsigned_tx, '$.memo'),3)) AS text)) STORED,
  node_id text GENERATED ALWAYS AS (json_extract(unsigned_tx, '$.validator.nodeID')) STORED,
  validator_start_ts integer GENERATED ALWAYS AS (json_extract(unsigned_tx, '$.validator.start')) STORED,
  validator_end_ts integer GENERATED ALWAYS AS (json_extract(unsigned_tx, '$.validator.end')) STORED,
  validator_weight integer GENERATED ALWAYS AS (json_extract(unsigned_tx, '$.validator.weight')) STORED,
  source_chain text GENERATED ALWAYS AS (json_extract(unsigned_tx, '$.sourceChain')) STORED,
  destination_chain text GENERATED ALWAYS AS (json_extract(unsigned_tx, '$.destinationChain')) STORED,
  rewards_addr text GENERATED ALWAYS AS (json_extract(unsigned_tx, '$.rewardsOwner.addresses[0]')) STORED,
  FOREIGN KEY(idx) REFERENCES raw_blocks_p(idx),
  FOREIGN KEY(type_id) REFERENCES types(id)
) STRICT, WITHOUT ROWID;

CREATE INDEX txs_p_id ON txs_p(id);
CREATE INDEX txs_p_height ON txs_p(height);
CREATE INDEX txs_p_block_id ON txs_p(block_id);
CREATE INDEX txs_p_type_id ON txs_p(type_id);
CREATE INDEX txs_p_node_id ON txs_p(node_id);
CREATE INDEX txs_p_ts ON txs_p(ts);
CREATE INDEX txs_p_rewards_addr ON txs_p(rewards_addr);
CREATE INDEX txs_p_signer_addr_p ON txs_p(signer_addr_p);
CREATE INDEX txs_p_signer_addr_c ON txs_p(signer_addr_c);


CREATE TABLE types (
  id integer PRIMARY KEY,
  name text NOT NULL
) STRICT, WITHOUT ROWID;

INSERT INTO types (id, name) VALUES (0,  "ApricotProposalBlock");
INSERT INTO types (id, name) VALUES (1,  "ApricotAbortBlock");
INSERT INTO types (id, name) VALUES (2,  "ApricotCommitBlock");
INSERT INTO types (id, name) VALUES (3,  "ApricotStandardBlock");
INSERT INTO types (id, name) VALUES (4,  "ApricotAtomicBlock");
INSERT INTO types (id, name) VALUES (5,  "secp256k1fx.TransferInput");
INSERT INTO types (id, name) VALUES (6,  "secp256k1fx.MintOutput");
INSERT INTO types (id, name) VALUES (7,  "secp256k1fx.TransferOutput");
INSERT INTO types (id, name) VALUES (8,  "secp256k1fx.MintOperation");
INSERT INTO types (id, name) VALUES (9,  "secp256k1fx.Credential");
INSERT INTO types (id, name) VALUES (10, "secp256k1fx.Input");
INSERT INTO types (id, name) VALUES (11, "secp256k1fx.OutputOwners");
INSERT INTO types (id, name) VALUES (12, "AddValidatorTx");
INSERT INTO types (id, name) VALUES (13, "AddSubnetValidatorTx");
INSERT INTO types (id, name) VALUES (14, "AddDelegatorTx");
INSERT INTO types (id, name) VALUES (15, "CreateChainTx");
INSERT INTO types (id, name) VALUES (16, "CreateSubnetTx");
INSERT INTO types (id, name) VALUES (17, "ImportTx");
INSERT INTO types (id, name) VALUES (18, "ExportTx");
INSERT INTO types (id, name) VALUES (19, "AdvanceTimeTx");
INSERT INTO types (id, name) VALUES (20, "RewardValidatorTx");
INSERT INTO types (id, name) VALUES (21, "stakeable.LockIn");
INSERT INTO types (id, name) VALUES (22, "stakeable.LockOut");
INSERT INTO types (id, name) VALUES (23, "RemoveSubnetValidatorTx");
INSERT INTO types (id, name) VALUES (24, "TransformSubnetTx");
INSERT INTO types (id, name) VALUES (25, "AddPermissionlessValidatorTx");
INSERT INTO types (id, name) VALUES (26, "AddPermissionlessDelegatorTx");
INSERT INTO types (id, name) VALUES (27, "EmptyProofOfPossession");
INSERT INTO types (id, name) VALUES (28, "BLSProofOfPossession  ");
INSERT INTO types (id, name) VALUES (29, "BanffProposalBlock");
INSERT INTO types (id, name) VALUES (30, "BanffAbortBlock");
INSERT INTO types (id, name) VALUES (31, "BanffCommitBlock");
INSERT INTO types (id, name) VALUES (32, "BanffStandardBlock");
