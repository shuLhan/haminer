// SPDX-FileCopyrightText: 2018 M. Shulhan <ms@kilabit.info>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package haminer

import (
	"encoding/json"
	"testing"

	"git.sr.ht/~shulhan/pakakeh.go/lib/test"
)

func TestParseUDPPacket(t *testing.T) {
	var (
		logp  = `TestParseUDPPacket`
		tdata *test.Data
		err   error
	)
	tdata, err = test.LoadData(`testdata/ParseUDPPacket_test.txt`)
	if err != nil {
		t.Fatal(logp, err)
	}

	var listCase = []string{
		`http_log_0000`,
	}

	var (
		httpLog *HTTPLog
		tag     string
		exp     string
		got     []byte
	)
	for _, tag = range listCase {
		httpLog = ParseUDPPacket(tdata.Input[tag], nil)

		got, err = json.MarshalIndent(httpLog, ``, `  `)
		if err != nil {
			t.Fatal(logp, err)
		}

		exp = string(tdata.Output[tag])
		test.Assert(t, tag, exp, string(got))
	}
}
