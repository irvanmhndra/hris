-- Self-service password reset, company sign-up metadata, and uploaded files.
CREATE TABLE password_resets (
 token_hash TEXT PRIMARY KEY,
 user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 expires_at TIMESTAMPTZ NOT NULL, used_at TIMESTAMPTZ,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX password_resets_user ON password_resets(user_id);
ALTER TABLE companies ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now();
CREATE TABLE files (
 id BIGSERIAL PRIMARY KEY, company_id BIGINT NOT NULL REFERENCES companies(id),
 name TEXT NOT NULL CHECK(length(name) BETWEEN 1 AND 200),
 content_type TEXT NOT NULL, size_bytes BIGINT NOT NULL CHECK(size_bytes BETWEEN 1 AND 10485760),
 storage_key TEXT NOT NULL UNIQUE, uploaded_by BIGINT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(company_id,id), FOREIGN KEY(company_id,uploaded_by) REFERENCES users(company_id,id)
);
