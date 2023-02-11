/*-
 * Copyright (c) 2021-2022
 *      Benny Siegert <bsiegert@gmail.com>
 *
 * Provided that these terms and disclaimer and all copyright notices
 * are retained or reproduced in an accompanying document, permission
 * is granted to deal in this work without restriction, including un-
 * limited rights to use, publicly perform, distribute, sell, modify,
 * merge, give away, or sublicence.
 *
 * This work is provided "AS IS" and WITHOUT WARRANTY of any kind, to
 * the utmost extent permitted by applicable law, neither express nor
 * implied; without malicious intent or gross negligence. In no event
 * may a licensor, author or contributor be held liable for indirect,
 * direct, other damage, loss, or other issues arising in any way out
 * of dealing in the work, even if advised of the possibility of such
 * damage or existence of a defect, except proven that it results out
 * of said person's immediate fault when using the work as intended.
 */

CREATE TABLE IF NOT EXISTS builds (
    build_id INTEGER PRIMARY KEY ASC,
    platform text NOT NULL,
    build_ts timestamp NOT NULL,
    branch text NOT NULL,
    compiler text NOT NULL,
    build_user text NOT NULL,
    report_url text NOT NULL,

    num_ok INTEGER NOT NULL,
    num_prefailed INTEGER NOT NULL,
    num_failed INTEGER NOT NULL,
    num_indirect_failed INTEGER NOT NULL,
    num_indirect_prefailed INTEGER NOT NULL    
);

CREATE TABLE IF NOT EXISTS pkgs (
    pkg_id INTEGER PRIMARY KEY ASC,
    category text NOT NULL,
    dir text NOT NULL,
    UNIQUE (category, dir)
);

CREATE VIEW IF NOT EXISTS pkgpaths (
    pkgpath
) AS SELECT (category || dir) AS pkgpath FROM pkgs;

CREATE TABLE IF NOT EXISTS results (
    result_id INTEGER PRIMARY KEY ASC,
    build_id INTEGER REFERENCES builds,
    pkg_id INTEGER REFERENCES pkgs,
    pkg_name text NOT NULL,
    build_status INTEGER NOT NULL,
    failed_deps text NOT NULL,
    breaks INTEGER NOT NULL
);