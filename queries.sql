-- name: DeleteAllForBuild :exec
DELETE from results
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
SELECT r.result_id, r.pkg_name, r.build_status, r.breaks, b.build_id, b.platform, b.build_ts, b.branch, b.compiler, b.build_user
FROM results r, builds b
WHERE r.build_id == b.build_id AND r.pkg_id == ?
ORDER BY b.build_ts DESC;

-- name: GetPkgsInCategory :many
SELECT DISTINCT dir
FROM pkgs
WHERE category = ?
ORDER BY dir;

-- name: GetPkgID :one
SELECT pkg_id FROM pkgs
WHERE category == ? and dir == ?;

-- name: GetResultsInCategory :many
SELECT r.*, p.*
FROM results r
JOIN pkgs p ON (r.pkg_id == p.pkg_id)
WHERE p.category == ? AND r.build_id == ?;

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

-- name: PutResult :exec
INSERT INTO results
(build_id, pkg_id, pkg_name, build_status, breaks, failed_deps)
VALUES (?, ?, ?, ?, ?, '');
