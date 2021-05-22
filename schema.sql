/*-
 * Copyright (c) 2021
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
    build_id int UNIQUE PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    platform text,
    build_ts timestamp,
    branch text,
    compiler text,
    build_user text,
    report_url text,

    num_ok int,
    num_prefailed int,
    num_failed int,
    num_indirect_failed int,
    num_indirect_prefailed int    
);

CREATE TABLE IF NOT EXISTS pkgs (
    pkg_id int UNIQUE PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    category text,
    dir text
);

CREATE TABLE IF NOT EXISTS results (
    result_id int UNIQUE PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    build_id int REFERENCES builds,
    pkg_id int REFERENCES pkgs,
    pkg_name text,
    build_status int8,
    failed_deps text[],
    breaks int
);