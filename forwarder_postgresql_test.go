// SPDX-FileCopyrightText: 2024 M. Shulhan <ms@kilabit.info>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package haminer

import (
	"encoding/json"
	"testing"

	"git.sr.ht/~shulhan/pakakeh.go/lib/test"
)

func TestForwarderPostgresql_Forwards(t *testing.T) {
	if !testIntegration {
		t.Skip()
	}

	var (
		logp = `TestForwarderPostgresql_Forwards`

		tdata *test.Data
		err   error
	)

	tdata, err = test.LoadData(`testdata/forwarderPostgresql_Forwards_test.txt`)
	if err != nil {
		t.Fatal(logp, err)
	}

	var (
		fwdConfig = ConfigForwarder{
			URL: `postgres://haminer:haminer@169.254.194.180/haminer?sslmode=disable`,
		}

		fwdpg *forwarderPostgresql
	)

	fwdpg, err = newForwarderPostgresql(fwdConfig)
	if err != nil {
		t.Fatal(logp, err)
	}

	err = fwdpg.conn.TruncateTable(tableNameHTTPLog)
	if err != nil {
		t.Fatal(logp, err)
	}

	var (
		tag  = `http_log.json`
		rawb = tdata.Input[tag]

		logs []*HTTPLog
	)

	err = json.Unmarshal(rawb, &logs)
	if err != nil {
		t.Fatal(logp, err)
	}

	fwdpg.Forwards(logs)

	var listLog []HTTPLog

	listLog, err = listHTTPLog(fwdpg.conn)
	if err != nil {
		t.Fatal(logp, err)
	}

	rawb, err = json.MarshalIndent(listLog, ``, `  `)
	if err != nil {
		t.Fatal(logp, err)
	}

	var exp = tdata.Output[tag]

	test.Assert(t, `listHTTPLog`, string(exp), string(rawb))
}
