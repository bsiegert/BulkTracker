// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0

package ddao

import (
	"database/sql"
	"time"
)

type Build struct {
	BuildID              int64
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

type Pkg struct {
	PkgID    int64
	Category string
	Dir      string
}

type Pkgpath struct {
	Pkgpath string
}

type Result struct {
	ResultID    int64
	BuildID     sql.NullInt64
	PkgID       sql.NullInt64
	PkgName     string
	BuildStatus int64
	FailedDeps  string
	Breaks      int64
}
