// SPDX-FileCopyrightText: 2024 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-or-later

package haminer

import (
	"flag"
	"os"
	"testing"
)

var testIntegration bool

func TestMain(m *testing.M) {
	flag.BoolVar(&testIntegration, `integration`, false, `Run integration tests`)
	flag.Parse()

	var status = m.Run()
	os.Exit(status)
}
