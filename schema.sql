CREATE TABLE blocks_p (
  idx integer PRIMARY KEY, -- index of the block in a particular node's index
  id text NOT NULL,
  bytes blob NOT NULL, -- block (may be a container block)
  decoded integer NOT NULL, -- If true we have decoded the below columns from bytes and saved txs_p records
  type_id integer,
  height integer,
  ts integer,
  parent_id text
) STRICT, WITHOUT ROWID;

CREATE UNIQUE INDEX blocks_p_id ON blocks_p(id);

CREATE TABLE txs_p (
  id text PRIMARY KEY,
  block_id text NOT NULL,
  type_id integer NOT NULL,
  unsigned_tx text NOT NULL, -- JSON of the unsignedTx key
  memo text GENERATED ALWAYS AS (cast(unhex(substr(json_extract(unsigned_tx, '$.memo'),3)) AS text)) STORED,
  node_id text GENERATED ALWAYS AS (json_extract(unsigned_tx, '$.validator.nodeID')) STORED,
  validator_start_ts integer GENERATED ALWAYS AS (json_extract(unsigned_tx, '$.validator.start')) STORED,
  validator_end_ts integer GENERATED ALWAYS AS (json_extract(unsigned_tx, '$.validator.end')) STORED,
  validator_weight integer GENERATED ALWAYS AS (json_extract(unsigned_tx, '$.validator.weight')) STORED,
  source_chain text GENERATED ALWAYS AS (json_extract(unsigned_tx, '$.sourceChain')) STORED,
  destination_chain text GENERATED ALWAYS AS (json_extract(unsigned_tx, '$.destinationChain')) STORED,
  output_addrs text GENERATED ALWAYS AS (json_extract(unsigned_tx, '$.outputs[0].output.addresses')) STORED,
  stake_addrs text GENERATED ALWAYS AS (json_extract(unsigned_tx, '$.stake[0].output.addresses')) STORED,
  rewards_addrs text GENERATED ALWAYS AS (json_extract(unsigned_tx, '$.rewardsOwner.addresses')) STORED,
  FOREIGN KEY(type_id) REFERENCES type_ids(id),
  FOREIGN KEY(block_id) REFERENCES blocks_p(id)
) STRICT, WITHOUT ROWID;

CREATE INDEX txs_p_type_id ON txs_p(type_id);

CREATE TABLE type_ids (
  id integer PRIMARY KEY,
  name text NOT NULL
) STRICT, WITHOUT ROWID;

INSERT INTO type_ids (id, name) VALUES (0,  "ApricotProposalBlock");
INSERT INTO type_ids (id, name) VALUES (1,  "ApricotAbortBlock");
INSERT INTO type_ids (id, name) VALUES (2,  "ApricotCommitBlock");
INSERT INTO type_ids (id, name) VALUES (3,  "ApricotStandardBlock");
INSERT INTO type_ids (id, name) VALUES (4,  "ApricotAtomicBlock");
INSERT INTO type_ids (id, name) VALUES (5,  "secp256k1fx.TransferInput");
INSERT INTO type_ids (id, name) VALUES (6,  "secp256k1fx.MintOutput");
INSERT INTO type_ids (id, name) VALUES (7,  "secp256k1fx.TransferOutput");
INSERT INTO type_ids (id, name) VALUES (8,  "secp256k1fx.MintOperation");
INSERT INTO type_ids (id, name) VALUES (9,  "secp256k1fx.Credential");
INSERT INTO type_ids (id, name) VALUES (10, "secp256k1fx.Input");
INSERT INTO type_ids (id, name) VALUES (11, "secp256k1fx.OutputOwners");
INSERT INTO type_ids (id, name) VALUES (12, "AddValidatorTx");
INSERT INTO type_ids (id, name) VALUES (13, "AddSubnetValidatorTx");
INSERT INTO type_ids (id, name) VALUES (14, "AddDelegatorTx");
INSERT INTO type_ids (id, name) VALUES (15, "CreateChainTx");
INSERT INTO type_ids (id, name) VALUES (16, "CreateSubnetTx");
INSERT INTO type_ids (id, name) VALUES (17, "ImportTx");
INSERT INTO type_ids (id, name) VALUES (18, "ExportTx");
INSERT INTO type_ids (id, name) VALUES (19, "AdvanceTimeTx");
INSERT INTO type_ids (id, name) VALUES (20, "RewardValidatorTx");
INSERT INTO type_ids (id, name) VALUES (21, "stakeable.LockIn");
INSERT INTO type_ids (id, name) VALUES (22, "stakeable.LockOut");
INSERT INTO type_ids (id, name) VALUES (23, "RemoveSubnetValidatorTx");
INSERT INTO type_ids (id, name) VALUES (24, "TransformSubnetTx");
INSERT INTO type_ids (id, name) VALUES (25, "AddPermissionlessValidatorTx");
INSERT INTO type_ids (id, name) VALUES (26, "AddPermissionlessDelegatorTx");
INSERT INTO type_ids (id, name) VALUES (27, "EmptyProofOfPossession");
INSERT INTO type_ids (id, name) VALUES (28, "BLSProofOfPossession  ");
INSERT INTO type_ids (id, name) VALUES (29, "BanffProposalBlock");
INSERT INTO type_ids (id, name) VALUES (30, "BanffAbortBlock");
INSERT INTO type_ids (id, name) VALUES (31, "BanffCommitBlock");
INSERT INTO type_ids (id, name) VALUES (32, "BanffStandardBlock");
