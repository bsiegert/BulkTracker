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
	deleteAllForBuildSQL = `DELETE from results
				WHERE build_id = ?;`
	getBuildSQL = `SELECT * FROM builds
				WHERE build_id = ?;`
	getLatestBuildsSQL = `SELECT * FROM builds
				ORDER BY build_ts DESC
				LIMIT 1000;`
	getAllPkgsSQL = `SELECT category || dir AS name
				FROM pkgs
				WHERE name LIKE ?
				ORDER BY name;`
	getPkgIDSQL = `SELECT pkg_id FROM pkgs
				WHERE category == ? and dir == ?;`
	putBuildSQL = `INSERT INTO builds
				(platform, build_ts, branch, compiler, build_user, report_url,
				num_ok, num_prefailed, num_failed,
				num_indirect_failed, num_indirect_prefailed)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				RETURNING build_id;`
	putPkgSQL = `INSERT OR IGNORE INTO pkgs
				(category, dir)
				VALUES (?, ?);`
	putResultSQL = `INSERT INTO results
				(build_id, pkg_id, pkg_name, build_status, breaks)
				VALUES (?, ?, ?, ?, ?)`
)

// New opens a new DB instance with the given SQL driver and connection string.
func New(ctx context.Context, driver, dbPath string) (*DB, error) {
	sqldb, err := sql.Open(driver, dbPath)
	if err != nil {
		return nil, err
	}
	db := &DB{
		DB: sqldb,
	}
	db.deleteAllForBuildStmt, err = db.DB.PrepareContext(ctx, deleteAllForBuildSQL)
	if err != nil {
		db.DB.Close()
		return nil, err
	}
	db.getBuildStmt, err = db.DB.PrepareContext(ctx, getLatestBuildsSQL)
	if err != nil {
		db.DB.Close()
		return nil, err
	}
	db.getLatestBuildsStmt, err = db.DB.PrepareContext(ctx, getLatestBuildsSQL)
	if err != nil {
		db.DB.Close()
		return nil, err
	}
	db.getAllPkgsStmt, err = db.DB.PrepareContext(ctx, getAllPkgsSQL)
	if err != nil {
		db.DB.Close()
		return nil, err
	}
	db.getPkgIDStmt, err = db.DB.PrepareContext(ctx, getPkgIDSQL)
	if err != nil {
		db.DB.Close()
		return nil, err
	}
	db.putBuildStmt, err = db.DB.PrepareContext(ctx, putBuildSQL)
	if err != nil {
		db.DB.Close()
		return nil, err
	}
	db.putPkgStmt, err = db.DB.PrepareContext(ctx, putPkgSQL)
	if err != nil {
		db.DB.Close()
		return nil, err
	}
	db.putResultStmt, err = db.DB.PrepareContext(ctx, putResultSQL)
	if err != nil {
		db.DB.Close()
		return nil, err
	}
	return db, nil
}

// DB is a wrapper around a SQL database that provides ready-made functions
// for interacting with the database.
type DB struct {
	DB *sql.DB

	// Prepared SQL statements.
	deleteAllForBuildStmt *sql.Stmt
	getBuildStmt          *sql.Stmt
	getLatestBuildsStmt   *sql.Stmt
	getAllPkgsStmt        *sql.Stmt
	getPkgIDStmt          *sql.Stmt
	putBuildStmt          *sql.Stmt
	putPkgStmt            *sql.Stmt
	putResultStmt         *sql.Stmt
}

// PutBuild writes the Build record to the DB and returns the ID.
func (d *DB) PutBuild(ctx context.Context, build *bulk.Build) (int, error) {
	tx, err := d.DB.BeginTx(ctx, nil)
	if err != nil {
		return -1, err
	}
	row := tx.StmtContext(ctx, d.putBuildStmt).QueryRowContext(ctx,
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

func (d *DB) PutResults(ctx context.Context, results []bulk.Pkg, buildID int) error {
	tx, err := d.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete all previous results for this build.
	_, err = tx.StmtContext(ctx, d.deleteAllForBuildStmt).ExecContext(ctx, buildID)
	if err != nil {
		return err
	}

	l := len(results)
	for i, pkg := range results {
		if i%1000 == 0 || i == l {
			log.Debugf(ctx, "Inserting record %v/%v ...", i, len(results))
		}
		_, err := tx.StmtContext(ctx, d.putPkgStmt).ExecContext(ctx, pkg.Category, pkg.Dir)
		if err != nil {
			return err
		}
		row := tx.StmtContext(ctx, d.getPkgIDStmt).QueryRowContext(ctx, pkg.Category, pkg.Dir)
		var pkgID int
		err = row.Scan(&pkgID)
		if err != nil {
			return err
		}
		// TODO add (array-valued) failed_deps field
		_, err = tx.StmtContext(ctx, d.putResultStmt).ExecContext(ctx, buildID, pkgID, pkg.PkgName, pkg.BuildStatus, pkg.Breaks)
		if err != nil {
			return err
		}
	}

	log.Infof(ctx, "Successfully added results for build %v", buildID)
	return tx.Commit()
}

// GetBuild returns the build record with the given ID.
func (d *DB) GetBuild(ctx context.Context, buildID int) (*bulk.Build, error) {
	builds, err := d.builds(ctx, false /* filter */, d.getBuildStmt, buildID)
	if err != nil {
		return nil, err
	}
	if len(builds) == 0 {
		return nil, errors.New("build not found")
	}
	return &builds[0], nil
}

// LatestBuilds returns a list of the latest 1000 (max) builds in the DB.
// If filter is true, filter out older builds for the same platform.
func (d *DB) LatestBuilds(ctx context.Context, filter bool) ([]bulk.Build, error) {
	return d.builds(ctx, filter, d.getLatestBuildsStmt)
}

func (d *DB) builds(ctx context.Context, filter bool, stmt *sql.Stmt, args ...interface{}) ([]bulk.Build, error) {
	rs, err := stmt.QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	var (
		b      bulk.Build
		builds []bulk.Build

		ok, prefailed, failed, indirectFailed, indirectPrefailed *int64
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
		if ok == nil {
			b.NumOK = 0
		} else {
			b.NumOK = *ok
		}
		if failed == nil {
			b.NumFailed = 0
		} else {
			b.NumFailed = *failed
		}
		if prefailed == nil {
			b.NumPrefailed = 0
		} else {
			b.NumPrefailed = *prefailed
		}
		if indirectFailed == nil {
			b.NumIndirectFailed = 0
		} else {
			b.NumIndirectFailed = *indirectFailed
		}
		if indirectPrefailed == nil {
			b.NumIndirectPrefailed = 0
		} else {
			b.NumIndirectPrefailed = *indirectPrefailed
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

func (d *DB) GetAllPkgsMatching(ctx context.Context, substr string) ([]string, error) {
	rs, err := d.getAllPkgsStmt.QueryContext(ctx, "%"+substr+"%")
	if err != nil {
		return nil, err
	}
	var (
		name  string
		names []string
	)
	for rs.Next() {
		err = rs.Scan(&name)
		if err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, nil
}
