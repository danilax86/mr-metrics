-- SPDX-FileCopyrightText: 2025 Danila Gorelko <hello@danilax86.space>
--
-- SPDX-License-Identifier: MIT

BEGIN;

DROP INDEX IF EXISTS idx_merged_mrs_created_at;
DROP INDEX IF EXISTS idx_merged_mrs_user_project;

ALTER TABLE merged_mrs
    DROP CONSTRAINT merged_mrs_pkey,
    ADD PRIMARY KEY (username, project_id);

DROP SEQUENCE IF EXISTS merged_mrs_id_seq;

ALTER TABLE merged_mrs
    DROP COLUMN IF EXISTS id,
    DROP COLUMN IF EXISTS created_at;

COMMIT;