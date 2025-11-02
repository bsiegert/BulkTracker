-- name: DeleteAllForBuild :exec
DELETE from results
WHERE build_id = ?;

-- name: deleteBuild :exec
DELETE from builds
WHERE build_id = ?;

-- name: GetBuild :one
SELECT * FROM builds
WHERE build_id = ?;

-- name: GetCategories :many
SELECT DISTINCT category
FROM pkgs
ORDER BY category;

-- name: getLatestBuilds :many
SELECT * FROM builds
ORDER BY build_ts DESC
LIMIT 1000;

-- name: GetLatestBuildsPerPlatform :many

-- This relies on the fact that IDs are monotonically increasing, so a newer
-- build will have a higher ID. There is probably a cleaner way.
SELECT * FROM builds
WHERE build_id IN (
	SELECT DISTINCT
	MAX(build_id) OVER (PARTITION BY platform, branch, compiler, build_user)
	FROM builds
) 
ORDER BY build_ts DESC
LIMIT 1000;

-- name: getAllPkgsMatching :many
SELECT pkgpath
FROM pkgpaths
WHERE pkgpath LIKE @name
ORDER BY pkgpath;


-- name: GetAllPkgResults :many
SELECT
	r.result_id,
	r.pkg_name,
	COALESCE(m.pkg_maintainer, '') AS pkg_maintainer,
	r.build_status,
	r.breaks,
	b.build_id,
	b.platform,
	b.build_ts,
	b.branch,
	b.compiler,
	b.build_user
FROM results r
JOIN builds b ON (r.build_id == b.build_id)
LEFT JOIN maintainers m ON (r.maintainer_id == m.maintainer_id)
WHERE r.pkg_id == ?
ORDER BY b.build_ts DESC;


-- name: ResultCount :one

-- ResultCount returns the number of result records for a given build.
SELECT count(*)
FROM results r
WHERE r.build_id = ?;

-- name: GetSingleResult :one
SELECT
	r.result_id,
	r.pkg_name,
	COALESCE(m.pkg_maintainer, '') AS pkg_maintainer,
	r.build_status,
	r.failed_deps,
	r.breaks,
	r.failure_msg,
	p.category,
	p.dir,
	b.build_id,
	b.platform,
	b.build_ts,
	b.branch,
	b.compiler,
	b.build_user,
	b.report_url
FROM results r
JOIN builds b ON (r.build_id == b.build_id)
JOIN pkgs p ON (r.pkg_id == p.pkg_id)
LEFT JOIN maintainers m ON (r.maintainer_id == m.maintainer_id)
WHERE r.result_id == ?;

-- name: GetSingleResultByPkgName :one
SELECT
	r.result_id,
	r.pkg_name,
	COALESCE(m.pkg_maintainer, '') AS pkg_maintainer,
	r.build_status,
	r.failed_deps,
	r.breaks,
	p.category,
	p.dir
FROM results r
JOIN pkgs p ON (r.pkg_id == p.pkg_id)
LEFT JOIN maintainers m ON (r.maintainer_id == m.maintainer_id)
WHERE r.build_id == ? AND r.pkg_name == ?;

-- name: GetSingleResultIDByPkgName :one
SELECT result_id
FROM results
WHERE build_id == ? and pkg_name == ?;

-- name: GetPkgsInCategory :many
SELECT DISTINCT dir
FROM pkgs
WHERE category = ?
ORDER BY dir;

-- name: GetPkgID :one
SELECT pkg_id FROM pkgs
WHERE category == ? and dir == ?;

-- name: GetResultsInCategory :many
SELECT r.*, p.*, COALESCE(m.pkg_maintainer, '') AS pkg_maintainer
FROM results r
JOIN pkgs p ON (r.pkg_id == p.pkg_id)
LEFT JOIN maintainers m ON (r.maintainer_id == m.maintainer_id)
WHERE p.category == ? AND r.build_id == ?;

-- name: GetPkgsBreakingMostOthers :many
SELECT
	r.result_id,
	(p.category || p.dir) AS pkg_path,
	r.pkg_name,
	COALESCE(m.pkg_maintainer, '') AS pkg_maintainer,
	r.build_status,
	r.failed_deps,
	r.breaks,
FROM results r
LEFT JOIN maintainers m ON (r.maintainer_id == m.maintainer_id)
JOIN pkgs p ON (r.pkg_id == p.pkg_id)
WHERE r.build_id == ? AND r.build_status > 0
ORDER BY r.breaks DESC
LIMIT 100;

-- name: GetSentinelStatus :many

-- Get the status and failed dependencies of the sentinel packages
-- (bulk-*).
SELECT *
FROM results
WHERE build_id == ? AND pkg_id IN (
	SELECT pkg_id
	FROM pkgs
	WHERE category == 'meta-pkgs/' AND dir LIKE 'bulk-%'
);

-- name: getPkgsBrokenBy :many
SELECT
	r.result_id,
	(p.category || p.dir) AS pkg_path,
	r.pkg_name,
	COALESCE(m.pkg_maintainer, '') AS pkg_maintainer,
	r.build_status,
	r.failed_deps,
	r.breaks,

FROM results r
LEFT JOIN maintainers m ON (r.maintainer_id == m.maintainer_id)
JOIN pkgs p ON (r.pkg_id == p.pkg_id)
WHERE r.build_id = ? AND
	r.failed_deps LIKE ?;

-- name: PutBuild :one

-- PutBuild writes the Build record to the DB and returns the ID.
INSERT INTO builds
(platform, build_ts, branch, compiler, build_user, report_url, num_ok,
	num_prefailed, num_failed, num_indirect_failed, num_indirect_prefailed)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING build_id;

-- name: PutPkg :exec
INSERT OR IGNORE INTO pkgs
(category, dir)
VALUES (?, ?);

-- name: PutMaintainer :exec
INSERT OR IGNORE INTO maintainers
(pkg_maintainer)
VALUES (?);

-- name: GetMaintainerID :one
SELECT maintainer_id FROM maintainers
WHERE pkg_maintainer == ?;

-- name: PutResult :exec
INSERT INTO results
(build_id, pkg_id, pkg_name, build_status, breaks, failed_deps, maintainer_id, failure_msg)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: SetBuildLastError :exec

-- SetBuildLastError sets the last_error column on a given build.
UPDATE builds
SET last_error = ?
WHERE build_id = ?;
