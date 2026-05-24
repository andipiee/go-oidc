-- +goose Up

-- Role-based access for relying parties. Stored as a plain TEXT (not an enum)
-- so adding new roles later is a no-op — no ALTER TYPE dance. RPs (e.g.
-- honest-kos) read this from the id_token's `role` claim and gate their own
-- admin surfaces.
ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'user';

-- +goose Down
ALTER TABLE users DROP COLUMN role;
