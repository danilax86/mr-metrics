BEGIN;

ALTER TABLE merged_mrs
    ADD COLUMN id         SERIAL,
    ADD COLUMN created_at TIMESTAMP NOT NULL DEFAULT NOW();

SELECT setval('merged_mrs_id_seq', (SELECT MAX(id) FROM merged_mrs));

ALTER TABLE merged_mrs
    DROP CONSTRAINT merged_mrs_pkey,
    ADD PRIMARY KEY (id);

CREATE INDEX idx_merged_mrs_user_project ON merged_mrs (username, project_id);
CREATE INDEX idx_merged_mrs_created_at ON merged_mrs (created_at);

COMMIT;
