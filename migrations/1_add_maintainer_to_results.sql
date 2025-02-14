CREATE TABLE IF NOT EXISTS maintainers (
    maintainer_id INTEGER PRIMARY KEY ASC,
    pkg_maintainer TEXT NOT NULL UNIQUE
);

ALTER TABLE results
    ADD COLUMN maintainer_id INTEGER REFERENCES maintainers(maintainer_id) ON DELETE SET NULL;

CREATE INDEX maintainer_id ON results (maintainer_id);