-- SPDX-FileCopyrightText: 2025 Danila Gorelko <hello@danilax86.space>
--
-- SPDX-License-Identifier: MIT

BEGIN;

ALTER TABLE merged_mrs
    ADD COLUMN IF NOT EXISTS id         SERIAL,
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMP NOT NULL DEFAULT NOW();

SELECT setval('merged_mrs_id_seq', (SELECT MAX(id) FROM merged_mrs));

ALTER TABLE merged_mrs
    DROP CONSTRAINT merged_mrs_pkey,
    ADD PRIMARY KEY (id);

CREATE INDEX IF NOT EXISTS idx_merged_mrs_user_project ON merged_mrs (username, project_id);
CREATE INDEX IF NOT EXISTS idx_merged_mrs_created_at ON merged_mrs (created_at);

COMMIT;
