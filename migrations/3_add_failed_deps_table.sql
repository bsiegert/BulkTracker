PRAGMA foreign_keys = on;

CREATE TABLE IF NOT EXISTS failed_deps (
    from_result INTEGER REFERENCES results NOT NULL,
    on_result INTEGER REFERENCES results NOT NULL,
    PRIMARY KEY (from_result, on_result),
    FOREIGN KEY (from_result) REFERENCES results (result_id) ON DELETE CASCADE,
    FOREIGN KEY (on_result) REFERENCES results (result_id) ON DELETE CASCADE
) WITHOUT ROWID;

CREATE INDEX failed_deps_i_from ON failed_deps (from_result);
CREATE INDEX failed_deps_i_on ON failed_deps (on_result);

BEGIN TRANSACTION;

INSERT INTO failed_deps (from_result, on_result)
WITH RECURSIVE split_deps (result_id, build_id, word, remainder) AS (
        -- first word
        SELECT
                result_id,
                build_id,
                NULL,
                failed_deps || ' '
        FROM results
        WHERE failed_deps IS NOT NULL and failed_deps != ''

        UNION ALL

        -- recursively cut the string at spaces
        SELECT
                result_id,
                build_id,
                TRIM(SUBSTR(remainder, 1, INSTR(remainder, ' '))),
                SUBSTR(remainder, INSTR(remainder, ' ') + 1)
        FROM split_deps
        WHERE remainder != ''
)
SELECT
        sd.result_id,
        r.result_id
FROM split_deps sd
JOIN results r ON sd.build_id = r.build_id AND sd.word = r.pkg_name
WHERE sd.word IS NOT NULL AND sd.word != '';

COMMIT;

-- Example query:
-- select r.pkg_name, (SELECT count(*) FROM failed_deps f WHERE r.result_id = f.on_result) as breaks FROM results r ORDER BY breaks DESC LIMIT 30;