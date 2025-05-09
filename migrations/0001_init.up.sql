-- SPDX-FileCopyrightText: 2025 Danila Gorelko <hello@danilax86.space>
--
-- SPDX-License-Identifier: MIT

CREATE TABLE IF NOT EXISTS projects
(
    project_id   INT PRIMARY KEY,
    project_name VARCHAR(255) NOT NULL,
    last_updated TIMESTAMP    NOT NULL
);

CREATE TABLE IF NOT EXISTS merged_mrs
(
    username    VARCHAR(255) NOT NULL,
    project_id  INT          NOT NULL,
    merge_count INT          NOT NULL,
    PRIMARY KEY (username, project_id),
    FOREIGN KEY (project_id) REFERENCES projects (project_id)
);