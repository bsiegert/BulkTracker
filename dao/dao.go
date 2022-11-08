/*-
 * Copyright (c) 2022
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

// Package dao defines a database interaction layer for BulkTracker.
package dao

import (
	"context"
	"database/sql"
	"errors"

	"github.com/bsiegert/BulkTracker/bulk"
	"github.com/bsiegert/BulkTracker/log"
)

const (
	deleteAllForBuild = iota
	getBuild
	getCategories
	getLatestBuilds
	getAllPkgs
	getPkgsInCategory
	getPkgID
	putBuild
	putPkg
	putResult
)

var sqlTxt = [...]string{
	deleteAllForBuild: `DELETE from results
				WHERE build_id = ?;`,
	getBuild: `SELECT * FROM builds
				WHERE build_id = ?;`,
	getCategories: `SELECT DISTINCT category
				FROM pkgs
				ORDER BY category;`,
	getLatestBuilds: `SELECT * FROM builds
				ORDER BY build_ts DESC
				LIMIT 1000;`,
	getAllPkgs: `SELECT category || dir AS name
				FROM pkgs
				WHERE name LIKE ?
				ORDER BY name;`,
	getPkgsInCategory: `SELECT DISTINCT dir
				FROM pkgs
				WHERE category = ?
				ORDER BY dir;`,
	getPkgID: `SELECT pkg_id FROM pkgs
				WHERE category == ? and dir == ?;`,
	putBuild: `INSERT INTO builds
				(platform, build_ts, branch, compiler, build_user, report_url,
				num_ok, num_prefailed, num_failed,
				num_indirect_failed, num_indirect_prefailed)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				RETURNING build_id;`,
	putPkg: `INSERT OR IGNORE INTO pkgs
				(category, dir)
				VALUES (?, ?);`,
	putResult: `INSERT INTO results
				(build_id, pkg_id, pkg_name, build_status, breaks, failed_deps)
				VALUES (?, ?, ?, ?, ?, '')`,
}

// New opens a new DB instance with the given SQL driver and connection string.
func New(ctx context.Context, driver, dbPath string) (*DB, error) {
	sqldb, err := sql.Open(driver, dbPath+"?_fk=true")
	if err != nil {
		return nil, err
	}
	db := &DB{
		DB:    sqldb,
		stmts: make([]*sql.Stmt, len(sqlTxt)),
	}
	for i := range sqlTxt {
		if sqlTxt[i] == "" {
			continue
		}
		var err error
		db.stmts[i], err = db.DB.PrepareContext(ctx, sqlTxt[i])
		if err != nil {
			db.DB.Close()
			return nil, err
		}
	}
	return db, nil
}

// DB is a wrapper around a SQL database that provides ready-made functions
// for interacting with the database.
type DB struct {
	DB    *sql.DB
	stmts []*sql.Stmt
}

// PutBuild writes the Build record to the DB and returns the ID.
func (d *DB) PutBuild(ctx context.Context, build *bulk.Build) (int, error) {
	tx, err := d.DB.BeginTx(ctx, nil)
	if err != nil {
		return -1, err
	}
	row := tx.StmtContext(ctx, d.stmts[putBuild]).QueryRowContext(ctx,
		build.Platform,
		build.Timestamp,
		build.Branch,
		build.Compiler,
		build.User,
		build.ReportURL,
		build.NumOK,
		build.NumPrefailed,
		build.NumFailed,
		build.NumIndirectFailed,
		build.NumIndirectPrefailed,
	)

	var id int
	err = row.Scan(&id)
	if err != nil {
		tx.Rollback()
		return -1, err
	}
	err = tx.Commit()
	return id, err
}

// PutResults writes the results for the given build ID to the database.
func (d *DB) PutResults(ctx context.Context, results []bulk.Pkg, buildID int) error {
	tx, err := d.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete all previous results for this build.
	_, err = tx.StmtContext(ctx, d.stmts[deleteAllForBuild]).ExecContext(ctx, buildID)
	if err != nil {
		return err
	}

	l := len(results)
	for i, pkg := range results {
		if i%1000 == 0 || i == l {
			log.Debugf(ctx, "Inserting record %v/%v ...", i, len(results))
		}
		_, err := tx.StmtContext(ctx, d.stmts[putPkg]).ExecContext(ctx, pkg.Category, pkg.Dir)
		if err != nil {
			return err
		}
		row := tx.StmtContext(ctx, d.stmts[getPkgID]).QueryRowContext(ctx, pkg.Category, pkg.Dir)
		var pkgID int
		err = row.Scan(&pkgID)
		if err != nil {
			return err
		}
		// TODO add (array-valued) failed_deps field
		_, err = tx.StmtContext(ctx, d.stmts[putResult]).ExecContext(ctx, buildID, pkgID, pkg.PkgName, pkg.BuildStatus, pkg.Breaks)
		if err != nil {
			return err
		}
	}

	log.Infof(ctx, "Successfully added results for build %v", buildID)
	return tx.Commit()
}

// GetBuild returns the build record with the given ID.
func (d *DB) GetBuild(ctx context.Context, buildID int) (*bulk.Build, error) {
	builds, err := d.builds(ctx, false /* filter */, getBuild, buildID)
	if err != nil {
		return nil, err
	}
	if len(builds) == 0 {
		return nil, errors.New("build not found")
	}
	return &builds[0], nil
}

// GetPkgID returns the ID of the package with the given category and dir.
func (d *DB) GetPkgID(ctx context.Context, category, dir string) (int, error) {
	row := d.stmts[getPkgID].QueryRowContext(ctx, category, dir)
	var pkgID int
	err := row.Scan(&pkgID)
	return pkgID, err
}

// LatestBuilds returns a list of the latest 1000 (max) builds in the DB.
// If filter is true, filter out older builds for the same platform.
func (d *DB) LatestBuilds(ctx context.Context, filter bool) ([]bulk.Build, error) {
	return d.builds(ctx, filter, getLatestBuilds)
}

func (d *DB) builds(ctx context.Context, filter bool, stmtID int, args ...interface{}) ([]bulk.Build, error) {
	rs, err := d.stmts[stmtID].QueryContext(ctx, args...)
	if err != nil {
		return nil, err
	}
	var (
		b      bulk.Build
		builds []bulk.Build

		ok, prefailed, failed, indirectFailed, indirectPrefailed sql.NullInt64
	)
RowLoop:
	for rs.Next() {
		err = rs.Scan(
			&b.BuildID,
			&b.Platform,
			&b.Timestamp,
			&b.Branch,
			&b.Compiler,
			&b.User,
			&b.ReportURL,
			&ok,
			&prefailed,
			&failed,
			&indirectFailed,
			&indirectPrefailed,
		)
		if err != nil {
			return nil, err
		}
		if ok.Valid {
			b.NumOK = ok.Int64
		} else {
			b.NumOK = 0
		}
		if failed.Valid {
			b.NumFailed = failed.Int64
		} else {
			b.NumFailed = 0
		}
		if prefailed.Valid {
			b.NumPrefailed = prefailed.Int64
		} else {
			b.NumPrefailed = 0
		}
		if indirectFailed.Valid {
			b.NumIndirectFailed = indirectFailed.Int64
		} else {
			b.NumIndirectFailed = 0
		}
		if indirectPrefailed.Valid {
			b.NumIndirectPrefailed = indirectPrefailed.Int64
		} else {
			b.NumIndirectPrefailed = 0
		}

		if filter {
			// Is this the first entry of this type?
			// TODO(bsiegert) eliminate O(n2) algo.
			for i := range builds {
				bb := builds[i]
				if b.Platform == bb.Platform && b.Branch == bb.Branch && b.Compiler == bb.Compiler && b.User == bb.User {
					continue RowLoop
				}
			}
		}
		builds = append(builds, b)
	}
	return builds, rs.Err()
}

// GetAllPkgsMatching returns all packages (category/dir) that contain
// substr as a substring match.
func (d *DB) GetAllPkgsMatching(ctx context.Context, substr string) ([]string, error) {
	rs, err := d.stmts[getAllPkgs].QueryContext(ctx, "%"+substr+"%")
	if err != nil {
		return nil, err
	}
	return extractStringList(rs)
}

// GetCategories returns all distinct categories.
func (d *DB) GetCategories(ctx context.Context) ([]string, error) {
	rs, err := d.stmts[getCategories].QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	return extractStringList(rs)
}

// GetPkgsInCategory returns all package dirs in the given category.
func (d *DB) GetPkgsInCategory(ctx context.Context, category string) ([]string, error) {
	rs, err := d.stmts[getPkgsInCategory].QueryContext(ctx, category)
	if err != nil {
		return nil, err
	}
	return extractStringList(rs)
}

// extractStringLists converts a single string-valued Rows set to a list of
// strings.
func extractStringList(rs *sql.Rows) ([]string, error) {
	var (
		name  string
		names []string
	)
	for rs.Next() {
		err := rs.Scan(&name)
		if err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, nil
}
