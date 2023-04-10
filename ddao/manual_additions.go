/*-
 * Copyright (c) 2023
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

package ddao

import (
	"context"
	"database/sql"
	"net/url"
	"path"
	"strings"

	"github.com/bsiegert/BulkTracker/log"
)

// Date returns the date part of the build timestamp.
func (b *Build) Date() string {
	return b.BuildTs.Format("2006-01-02")
}

func (r *GetSingleResultRow) BaseURL() string {
	if n := strings.Index(r.ReportUrl, "meta/"); n != -1 {
		return r.ReportUrl[:n]
	}
	u, err := url.Parse(r.ReportUrl)
	if err != nil {
		return ""
	}
	u.Path = path.Dir(u.Path)
	return u.String()
}

// A PkgResult is a build result for a package.
type PkgResult struct {
	Pkg
	Result
}

// DB is a wrapper aound a SQL database that provides ready-made functions for
// interacting with the database.
type DB struct {
	Queries
}

// PutResults writes the results for the given build ID to the database.
func (d *DB) PutResults(ctx context.Context, results []PkgResult, buildID int64) error {
	tx, err := d.db.(*sql.DB).BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := d.WithTx(tx)
	err = q.DeleteAllForBuild(ctx, sql.NullInt64{
		Int64: buildID,
		Valid: true,
	})
	if err != nil {
		return err
	}

	l := len(results)
	for i, result := range results {
		if i%1000 == 0 || i == l {
			log.Debugf(ctx, "Inserting record %v/%v ...", i, len(results))
		}
		params := PutPkgParams{
			Category: result.Category,
			Dir:      result.Dir,
		}
		err := q.PutPkg(ctx, params)
		if err != nil {
			return err
		}
		pkgID, err := q.GetPkgID(ctx, GetPkgIDParams(params))
		if err != nil {
			return err
		}

		// TODO add (array-valued) failed_deps field
		err = q.PutResult(ctx, PutResultParams{
			BuildID: sql.NullInt64{
				Int64: buildID,
				Valid: true,
			},
			PkgID: sql.NullInt64{
				Int64: pkgID,
				Valid: true,
			},
			PkgName:     result.PkgName,
			BuildStatus: result.BuildStatus,
			Breaks:      result.Breaks,
		})
		if err != nil {
			return err
		}
	}

	log.Infof(ctx, "Successfully added results for build %v", buildID)
	return tx.Commit()
}

func (d *DB) LatestBuilds(ctx context.Context, filter bool) ([]Build, error) {
	if filter {
		return d.GetLatestBuildsPerPlatform(ctx)
	}
	return d.getLatestBuilds(ctx)
}

// GetAllPkgsMatching returns all packages (category/dir) that contain
// substr as a substring match.
func (d *DB) GetAllPkgsMatching(ctx context.Context, substr string) ([]string, error) {
	rows, err := d.Queries.db.QueryContext(ctx, getAllPkgsMatching, "%"+substr+"%")
	if err != nil {
		return nil, err
	}

	// This is a copy of the generated code.
	defer rows.Close()
	var items []string
	for rows.Next() {
		var pkgpath string
		if err := rows.Scan(&pkgpath); err != nil {
			return nil, err
		}
		items = append(items, pkgpath)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetAllPkgResults returns all results for the given category and dir.
func (d *DB) GetAllPkgResults(ctx context.Context, category, dir string) ([]GetAllPkgResultsRow, error) {
	tx, err := d.db.(*sql.DB).BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	q := d.Queries.WithTx(tx)

	pkgID, err := q.GetPkgID(ctx, GetPkgIDParams{
		Category: category,
		Dir:      dir,
	})
	if err != nil {
		return nil, err
	}
	return q.GetAllPkgResults(ctx, sql.NullInt64{
		Int64: pkgID,
		Valid: true,
	})
}
