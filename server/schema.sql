CREATE TABLE IF NOT EXISTS worlds (
 id text PRIMARY KEY, last_tick bigint NOT NULL DEFAULT 0 CHECK (last_tick >= 0),
 due_tick bigint NOT NULL DEFAULT 0 CHECK (due_tick >= last_tick),
 recovering boolean NOT NULL DEFAULT false
);
CREATE TABLE IF NOT EXISTS companies (
 world text REFERENCES worlds(id), account text NOT NULL, token_hash bytea NOT NULL UNIQUE,
 state jsonb NOT NULL, PRIMARY KEY (world, account)
);
CREATE TABLE IF NOT EXISTS commands (
 world text, account text, id text, body jsonb NOT NULL, target bigint NOT NULL,
 status text NOT NULL DEFAULT 'accepted' CHECK (status IN ('accepted','applied','rejected')),
 reason text NOT NULL DEFAULT '', PRIMARY KEY (world, account, id),
 FOREIGN KEY (world, account) REFERENCES companies(world, account)
);
CREATE INDEX IF NOT EXISTS commands_tick ON commands(world, target, account, id);
CREATE TABLE IF NOT EXISTS ticks (
 world text REFERENCES worlds(id), number bigint, PRIMARY KEY (world, number)
);
CREATE TABLE IF NOT EXISTS ledger (
 world text, account text, tick bigint, ordinal integer, event text NOT NULL,
 movements jsonb NOT NULL, PRIMARY KEY (world, account, tick, ordinal),
 FOREIGN KEY (world, account) REFERENCES companies(world, account),
 FOREIGN KEY (world, tick) REFERENCES ticks(world, number)
);
