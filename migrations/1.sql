CREATE TABLE projects
(
    project_id   INT PRIMARY KEY,
    project_name VARCHAR(255) NOT NULL,
    last_updated TIMESTAMP    NOT NULL
);

CREATE TABLE merged_mrs
(
    username    VARCHAR(255) NOT NULL,
    project_id  INT          NOT NULL,
    merge_count INT          NOT NULL,
    PRIMARY KEY (username, project_id),
    FOREIGN KEY (project_id) REFERENCES projects (project_id)
);

CREATE INDEX merged_mrs_project_id_idx ON merged_mrs (project_id);
CREATE INDEX projects_project_name_idx ON projects (project_name);