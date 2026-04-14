CREATE TABLE IF NOT EXISTS accounts (
    id         TEXT        PRIMARY KEY,
    balance    BIGINT      NOT NULL DEFAULT 0
                           CONSTRAINT accounts_balance_non_negative CHECK (balance >= 0),
    status     TEXT        NOT NULL DEFAULT 'active'
                           CONSTRAINT accounts_status_valid CHECK (status IN ('active', 'closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER accounts_set_updated_at
    BEFORE UPDATE ON accounts
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

INSERT INTO accounts (id, balance, status) VALUES
    ('alice', 100000, 'active'),
    ('bob',   50000,  'active')
ON CONFLICT (id) DO NOTHING;
