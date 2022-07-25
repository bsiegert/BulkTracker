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

	"github.com/bsiegert/BulkTracker/bulk"
)

const (
	putBuildSQL = `INSERT INTO builds
				(platform, build_ts, branch, compiler, build_user, report_url)
				VALUES (?, ?, ?, ?, ?, ?)
				RETURNING build_id;`
	deleteAllForBuildSQL = `DELETE from results
							WHERE build_id = ?;`
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
	db.putBuildStmt, err = db.DB.PrepareContext(ctx, putBuildSQL)
	if err != nil {
		db.DB.Close()
		return nil, err
	}
	db.deleteAllForBuildStmt, err = db.DB.PrepareContext(ctx, deleteAllForBuildSQL)
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
	putBuildStmt          *sql.Stmt
	deleteAllForBuildStmt *sql.Stmt
}

// PutBuild writes the Build record to the DB and returns the ID.
func (d *DB) PutBuild(ctx context.Context, build *bulk.Build) (int, error) {
	tx, err := d.DB.BeginTx(ctx, nil)
	if err != nil {
		return -1, err
	}
	row := tx.StmtContext(ctx, d.putBuildStmt).QueryRowContext(ctx, build.Platform, build.Timestamp, build.Branch, build.Compiler, build.User, build.ReportURL)

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

	// TODO: actually write results!

	return tx.Commit()
}
