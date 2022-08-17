// Copyright 2019, M. Shulhan (ms@kilabit.info).  All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package haminer

import (
	"regexp"
	"testing"
	"time"

	"github.com/shuLhan/share/lib/test"
)

func TestNewConfig(t *testing.T) {
	cases := []struct {
		exp  *Config
		desc string
	}{{
		desc: "With default config",
		exp: &Config{
			listenAddr:      defListenAddr,
			listenPort:      defListenPort,
			ForwardInterval: defForwardInterval,
		},
	}}

	for _, c := range cases {
		t.Log(c.desc)

		got := NewConfig()

		test.Assert(t, `Config`, c.exp, got)
	}
}

func TestLoad(t *testing.T) {
	type testCase struct {
		exp      *Config
		desc     string
		in       string
		expError string
	}

	var cases = []testCase{{
		desc: "With empty path",
		exp: &Config{
			listenAddr:      defListenAddr,
			listenPort:      defListenPort,
			ForwardInterval: defForwardInterval,
		},
	}, {
		desc:     `With path not exist`,
		in:       `testdata/notexist.conf`,
		expError: `Load: open testdata/notexist.conf: no such file or directory`,
	}, {
		desc: "With path exist",
		in:   "testdata/haminer.conf",
		exp: &Config{
			Influxd: ConfigForwarder{
				Version:     `v2`,
				Url:         `http://127.0.0.1:8086`,
				Org:         `kilabit.info`,
				Bucket:      `haproxy`,
				apiWrite:    `http://127.0.0.1:8086/api/v2/write?bucket=haproxy&org=kilabit.info&precision=ns`,
				headerToken: `Token `,
			},
			Listen:          `0.0.0.0:8080`,
			listenAddr:      `0.0.0.0`,
			listenPort:      8080,
			ForwardInterval: time.Second * 20,
			AcceptBackend: []string{
				"a",
				"b",
			},
			RequestHeaders: []string{
				"host",
				"referrer",
			},
			HttpUrl: []string{
				`/[0-9]+-\w+-\w+-\w+-\w+-\w+ => /-`,
				`/\w+-\w+-\w+-\w+-\w+ => /-`,
				`/[0-9]+ => /-`,
			},
			retags: []*tagPreprocessor{{
				name:  "http_url",
				regex: regexp.MustCompile(`/[0-9]+-\w+-\w+-\w+-\w+-\w+`),
				repl:  `/-`,
			}, {
				name:  "http_url",
				regex: regexp.MustCompile(`/\w+-\w+-\w+-\w+-\w+`),
				repl:  `/-`,
			}, {
				name:  "http_url",
				regex: regexp.MustCompile(`/[0-9]+`),
				repl:  `/-`,
			}},
		},
	}}

	var (
		c   testCase
		got *Config
		err error
	)

	for _, c = range cases {
		t.Log(c.desc)

		got = NewConfig()
		err = got.Load(c.in)
		if err != nil {
			test.Assert(t, `error`, c.expError, err.Error())
			continue
		}

		test.Assert(t, `Config`, c.exp, got)
	}
}

func TestSetListen(t *testing.T) {
	cases := []struct {
		exp  *Config
		desc string
		in   string
	}{{
		desc: "With empty listen",
		exp: &Config{
			listenAddr:      defListenAddr,
			listenPort:      defListenPort,
			ForwardInterval: defForwardInterval,
		},
	}, {
		desc: "With empty port",
		in:   "127.0.0.2",
		exp: &Config{
			listenAddr:      `127.0.0.2`,
			listenPort:      defListenPort,
			ForwardInterval: defForwardInterval,
		},
	}, {
		desc: "With no port",
		in:   "127.0.0.3:",
		exp: &Config{
			listenAddr:      `127.0.0.3`,
			listenPort:      defListenPort,
			ForwardInterval: defForwardInterval,
		},
	}}

	for _, c := range cases {
		t.Log(c.desc)

		got := NewConfig()
		got.SetListen(c.in)

		test.Assert(t, `Config`, c.exp, got)
	}
}

func TestParsePreprocessTag(t *testing.T) {
	type testCase struct {
		desc    string
		httpUrl []string
		exp     []*tagPreprocessor
	}

	var (
		cfg = NewConfig()
	)

	var cases = []testCase{{
		desc:    `With invalid format`,
		httpUrl: []string{``},
	}, {
		desc:    `With empty regex`,
		httpUrl: []string{`=>`},
	}, {
		desc: `With valid value`,
		httpUrl: []string{
			`/[0-9]+ => /-`,
		},
		exp: []*tagPreprocessor{{
			name:  "http_url",
			regex: regexp.MustCompile(`/[0-9]+`),
			repl:  "/-",
		}},
	}}

	var (
		c   testCase
		err error
	)

	for _, c = range cases {
		t.Log(c.desc)

		cfg.retags = nil
		cfg.HttpUrl = c.httpUrl

		err = cfg.parsePreprocessTag()
		if err != nil {
			t.Fatal(err)
		}

		test.Assert(t, `retags`, c.exp, cfg.retags)
	}
}
