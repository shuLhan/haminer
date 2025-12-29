// SPDX-FileCopyrightText: 2024 M. Shulhan <ms@kilabit.info>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package haminer

import (
	"database/sql"
	"fmt"

	"git.sr.ht/~shulhan/pakakeh.go/lib/mlog"
	libsql "git.sr.ht/~shulhan/pakakeh.go/lib/sql"
	"github.com/lib/pq"
)

// forwarderPostgresql the client to write logs to Postgresql database.
type forwarderPostgresql struct {
	conn *libsql.Client
}

// newForwarderPostgresql create new forwarder for Postgresql.
func newForwarderPostgresql(cfg ConfigForwarder) (fw *forwarderPostgresql, err error) {
	var logp = `newForwarderPostgresql`

	fw = &forwarderPostgresql{}

	var opts = libsql.ClientOptions{
		DriverName: libsql.DriverNamePostgres,
		DSN:        cfg.URL,
	}

	fw.conn, err = libsql.NewClient(opts)
	if err != nil {
		return nil, fmt.Errorf(`%s: %w`, logp, err)
	}

	return fw, nil
}

// Forwards insert the list of HTTP log into the Postgresql.
func (fw *forwarderPostgresql) Forwards(listLog []*HTTPLog) {
	var (
		logp = `Forwards`

		sqltx *sql.Tx
		err   error
	)

	sqltx, err = fw.conn.Begin()
	if err != nil {
		mlog.Errf(`%s: %s`, logp, err)
		return
	}

	var (
		httpLog = HTTPLog{}
		meta    = httpLog.generateSQLMeta(libsql.DriverNamePostgres, libsql.DMLKindInsert)
	)

	var q = pq.CopyInSchema(`public`, `http_log`, meta.ListName...)

	var (
		stmt *sql.Stmt
		alog *HTTPLog
	)

	stmt, err = sqltx.Prepare(q)
	if err != nil {
		goto failed
	}

	for _, alog = range listLog {
		httpLog = *alog

		_, err = stmt.Exec(meta.ListValue...)
		if err != nil {
			goto failed
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		goto failed
	}

	err = stmt.Close()
	if err != nil {
		mlog.Errf(`%s: %s`, logp, err)
		_ = sqltx.Rollback()
		return
	}

	err = sqltx.Commit()
	if err != nil {
		mlog.Errf(`%s: %s`, logp, err)
		return
	}

	return

failed:
	mlog.Errf(`%s: %s`, logp, err)

	if stmt != nil {
		err = stmt.Close() //nolint:sqlclosecheck
		if err != nil {
			mlog.Errf(`%s: %s`, logp, err)
		}
	}

	err = sqltx.Rollback()
	if err != nil {
		mlog.Errf(`%s: %s`, logp, err)
	}
}
