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