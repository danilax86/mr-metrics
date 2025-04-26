-- SPDX-FileCopyrightText: 2025 Danila Gorelko <hello@danilax86.space>
--
-- SPDX-License-Identifier: MIT

ALTER TABLE merged_mrs
    ADD COLUMN IF NOT EXISTS merged_at TIMESTAMP NOT NULL DEFAULT NOW();

ALTER TABLE merged_mrs
    DROP COLUMN IF EXISTS created_at;