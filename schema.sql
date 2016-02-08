-- CREATE TABLE "certificates" (
--   "id" serial PRIMARY KEY,
--   "hash" bytea UNIQUE NOT NULL,
--   "offset" int NOT NULL,
--   "length" int NOT NULL
-- );

CREATE TABLE "chains" (
  "id" serial PRIMARY KEY,
  "hash" bytea UNIQUE NOT NULL,
  -- "certIDs" jsonb UNIQUE NOT NULL,
  "root_dn" varchar(255) NOT NULL,
  "entry_type" smallint NOT NULL,
  "unparseable_component" boolean NOT NULL,
  "logs" jsonb NOT NULL
);

CREATE TABLE "log_entries" (
  "id" serial PRIMARY KEY,
  "chain_id" int NOT NULL,
  "entry_num" int NOT NULL,
  "log_id" bytea NOT NULL,
  CONSTRAINT "entry_num_log_id" UNIQUE ("entry_num", "log_id")
);
