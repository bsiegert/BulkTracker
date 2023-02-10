// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0
// source: queries.sql

package ddao

import (
	"context"
	"database/sql"
	"time"
)

const deleteAllForBuild = `-- name: DeleteAllForBuild :exec
DELETE from results
WHERE build_id = ?
`

func (q *Queries) DeleteAllForBuild(ctx context.Context, buildID sql.NullInt64) error {
	_, err := q.db.ExecContext(ctx, deleteAllForBuild, buildID)
	return err
}

const getAllPkgResults = `-- name: GetAllPkgResults :many
SELECT r.pkg_name, r.build_status, r.breaks, b.build_id, b.platform, b.build_ts, b.branch, b.compiler, b.build_user
FROM results r, builds b
WHERE r.build_id == b.build_id AND r.pkg_id == ?
ORDER BY b.build_ts DESC
`

type GetAllPkgResultsRow struct {
	PkgName     string
	BuildStatus int64
	Breaks      int64
	BuildID     int64
	Platform    string
	BuildTs     time.Time
	Branch      string
	Compiler    string
	BuildUser   string
}

func (q *Queries) GetAllPkgResults(ctx context.Context, pkgID sql.NullInt64) ([]GetAllPkgResultsRow, error) {
	rows, err := q.db.QueryContext(ctx, getAllPkgResults, pkgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetAllPkgResultsRow
	for rows.Next() {
		var i GetAllPkgResultsRow
		if err := rows.Scan(
			&i.PkgName,
			&i.BuildStatus,
			&i.Breaks,
			&i.BuildID,
			&i.Platform,
			&i.BuildTs,
			&i.Branch,
			&i.Compiler,
			&i.BuildUser,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getBuild = `-- name: GetBuild :one
SELECT build_id, platform, build_ts, branch, compiler, build_user, report_url, num_ok, num_prefailed, num_failed, num_indirect_failed, num_indirect_prefailed FROM builds
WHERE build_id = ?
`

func (q *Queries) GetBuild(ctx context.Context, buildID int64) (Build, error) {
	row := q.db.QueryRowContext(ctx, getBuild, buildID)
	var i Build
	err := row.Scan(
		&i.BuildID,
		&i.Platform,
		&i.BuildTs,
		&i.Branch,
		&i.Compiler,
		&i.BuildUser,
		&i.ReportUrl,
		&i.NumOk,
		&i.NumPrefailed,
		&i.NumFailed,
		&i.NumIndirectFailed,
		&i.NumIndirectPrefailed,
	)
	return i, err
}

const getCategories = `-- name: GetCategories :many
SELECT DISTINCT category
FROM pkgs
ORDER BY category
`

func (q *Queries) GetCategories(ctx context.Context) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, getCategories)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, err
		}
		items = append(items, category)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getLatestBuildsPerPlatform = `-- name: GetLatestBuildsPerPlatform :many

SELECT build_id, platform, build_ts, branch, compiler, build_user, report_url, num_ok, num_prefailed, num_failed, num_indirect_failed, num_indirect_prefailed FROM builds
WHERE build_id IN (
	SELECT DISTINCT
	MAX(build_id) OVER (PARTITION BY platform, branch, compiler, build_user)
	FROM builds
) 
ORDER BY build_ts DESC
LIMIT 1000
`

// This relies on the fact that IDs are monotonically increasing, so a newer
// build will have a higher ID. There is probably a cleaner way.
func (q *Queries) GetLatestBuildsPerPlatform(ctx context.Context) ([]Build, error) {
	rows, err := q.db.QueryContext(ctx, getLatestBuildsPerPlatform)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Build
	for rows.Next() {
		var i Build
		if err := rows.Scan(
			&i.BuildID,
			&i.Platform,
			&i.BuildTs,
			&i.Branch,
			&i.Compiler,
			&i.BuildUser,
			&i.ReportUrl,
			&i.NumOk,
			&i.NumPrefailed,
			&i.NumFailed,
			&i.NumIndirectFailed,
			&i.NumIndirectPrefailed,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getPkgID = `-- name: GetPkgID :one
SELECT pkg_id FROM pkgs
WHERE category == ? and dir == ?
`

type GetPkgIDParams struct {
	Category string
	Dir      string
}

func (q *Queries) GetPkgID(ctx context.Context, arg GetPkgIDParams) (int64, error) {
	row := q.db.QueryRowContext(ctx, getPkgID, arg.Category, arg.Dir)
	var pkg_id int64
	err := row.Scan(&pkg_id)
	return pkg_id, err
}

const getPkgsInCategory = `-- name: GetPkgsInCategory :many
SELECT DISTINCT dir
FROM pkgs
WHERE category = ?
ORDER BY dir
`

func (q *Queries) GetPkgsInCategory(ctx context.Context, category string) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, getPkgsInCategory, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var dir string
		if err := rows.Scan(&dir); err != nil {
			return nil, err
		}
		items = append(items, dir)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getResultsInCategory = `-- name: GetResultsInCategory :many
SELECT r.result_id, r.build_id, r.pkg_id, r.pkg_name, r.build_status, r.failed_deps, r.breaks, p.pkg_id, p.category, p.dir
FROM results r
JOIN pkgs p ON (r.pkg_id == p.pkg_id)
WHERE p.category == ? AND r.build_id == ?
`

type GetResultsInCategoryParams struct {
	Category string
	BuildID  sql.NullInt64
}

type GetResultsInCategoryRow struct {
	ResultID    int64
	BuildID     sql.NullInt64
	PkgID       sql.NullInt64
	PkgName     string
	BuildStatus int64
	FailedDeps  string
	Breaks      int64
	PkgID_2     int64
	Category    string
	Dir         string
}

func (q *Queries) GetResultsInCategory(ctx context.Context, arg GetResultsInCategoryParams) ([]GetResultsInCategoryRow, error) {
	rows, err := q.db.QueryContext(ctx, getResultsInCategory, arg.Category, arg.BuildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetResultsInCategoryRow
	for rows.Next() {
		var i GetResultsInCategoryRow
		if err := rows.Scan(
			&i.ResultID,
			&i.BuildID,
			&i.PkgID,
			&i.PkgName,
			&i.BuildStatus,
			&i.FailedDeps,
			&i.Breaks,
			&i.PkgID_2,
			&i.Category,
			&i.Dir,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const putBuild = `-- name: PutBuild :one

INSERT INTO builds
(platform, build_ts, branch, compiler, build_user, report_url, num_ok,
	num_prefailed, num_failed, num_indirect_failed, num_indirect_prefailed)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING build_id
`

type PutBuildParams struct {
	Platform             string
	BuildTs              time.Time
	Branch               string
	Compiler             string
	BuildUser            string
	ReportUrl            string
	NumOk                int64
	NumPrefailed         int64
	NumFailed            int64
	NumIndirectFailed    int64
	NumIndirectPrefailed int64
}

// PutBuild writes the Build record to the DB and returns the ID.
func (q *Queries) PutBuild(ctx context.Context, arg PutBuildParams) (int64, error) {
	row := q.db.QueryRowContext(ctx, putBuild,
		arg.Platform,
		arg.BuildTs,
		arg.Branch,
		arg.Compiler,
		arg.BuildUser,
		arg.ReportUrl,
		arg.NumOk,
		arg.NumPrefailed,
		arg.NumFailed,
		arg.NumIndirectFailed,
		arg.NumIndirectPrefailed,
	)
	var build_id int64
	err := row.Scan(&build_id)
	return build_id, err
}

const putPkg = `-- name: PutPkg :exec
INSERT OR IGNORE INTO pkgs
(category, dir)
VALUES (?, ?)
`

type PutPkgParams struct {
	Category string
	Dir      string
}

func (q *Queries) PutPkg(ctx context.Context, arg PutPkgParams) error {
	_, err := q.db.ExecContext(ctx, putPkg, arg.Category, arg.Dir)
	return err
}

const putResult = `-- name: PutResult :exec
INSERT INTO results
(build_id, pkg_id, pkg_name, build_status, breaks, failed_deps)
VALUES (?, ?, ?, ?, ?, '')
`

type PutResultParams struct {
	BuildID     sql.NullInt64
	PkgID       sql.NullInt64
	PkgName     string
	BuildStatus int64
	Breaks      int64
}

func (q *Queries) PutResult(ctx context.Context, arg PutResultParams) error {
	_, err := q.db.ExecContext(ctx, putResult,
		arg.BuildID,
		arg.PkgID,
		arg.PkgName,
		arg.BuildStatus,
		arg.Breaks,
	)
	return err
}

const getAllPkgs = `-- name: getAllPkgs :many
SELECT category || dir AS name
FROM pkgs
WHERE (category || dir) LIKE ?
ORDER BY name
`

func (q *Queries) getAllPkgs(ctx context.Context, dollar_1 interface{}) ([]interface{}, error) {
	rows, err := q.db.QueryContext(ctx, getAllPkgs, dollar_1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []interface{}
	for rows.Next() {
		var name interface{}
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		items = append(items, name)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getLatestBuilds = `-- name: getLatestBuilds :many
SELECT build_id, platform, build_ts, branch, compiler, build_user, report_url, num_ok, num_prefailed, num_failed, num_indirect_failed, num_indirect_prefailed FROM builds
ORDER BY build_ts DESC
LIMIT 1000
`

func (q *Queries) getLatestBuilds(ctx context.Context) ([]Build, error) {
	rows, err := q.db.QueryContext(ctx, getLatestBuilds)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Build
	for rows.Next() {
		var i Build
		if err := rows.Scan(
			&i.BuildID,
			&i.Platform,
			&i.BuildTs,
			&i.Branch,
			&i.Compiler,
			&i.BuildUser,
			&i.ReportUrl,
			&i.NumOk,
			&i.NumPrefailed,
			&i.NumFailed,
			&i.NumIndirectFailed,
			&i.NumIndirectPrefailed,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
