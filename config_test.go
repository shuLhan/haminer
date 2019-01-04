// Copyright 2019, M. Shulhan (ms@kilabit.info).  All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package haminer

import (
	"regexp"
	"testing"
	"time"

	"github.com/shuLhan/share/lib/ini"
	"github.com/shuLhan/share/lib/test"
)

func TestNewConfig(t *testing.T) {
	cases := []struct {
		desc string
		exp  *Config
	}{{
		desc: "With default config",
		exp: &Config{
			ListenAddr:      defListenAddr,
			ListenPort:      defListenPort,
			ForwardInterval: defForwardInterval,
		},
	}}

	for _, c := range cases {
		t.Log(c.desc)

		got := NewConfig()

		test.Assert(t, "Config", c.exp, got, true)
	}
}

func TestLoad(t *testing.T) {
	cases := []struct {
		desc string
		in   string
		exp  *Config
	}{{
		desc: "With empty path",
		exp: &Config{
			ListenAddr:      defListenAddr,
			ListenPort:      defListenPort,
			ForwardInterval: defForwardInterval,
		},
	}, {
		desc: "With path not exist",
		in:   "testdata/notexist.conf",
		exp: &Config{
			ListenAddr:      defListenAddr,
			ListenPort:      defListenPort,
			ForwardInterval: defForwardInterval,
		},
	}, {
		desc: "With path exist",
		in:   "testdata/haminer.conf",
		exp: &Config{
			ListenAddr:      "0.0.0.0",
			ListenPort:      8080,
			ForwardInterval: time.Second * 20,
			AcceptBackend: []string{
				"a",
				"b",
			},
			RequestHeaders: []string{
				"host",
				"referrer",
			},
			InfluxAPIWrite: "http://127.0.0.1:8086/write",
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

	for _, c := range cases {
		t.Log(c.desc)

		got := NewConfig()
		got.Load(c.in)

		test.Assert(t, "Config", c.exp, got, true)
	}
}

func TestSetListen(t *testing.T) {
	cases := []struct {
		desc string
		in   string
		exp  *Config
	}{{
		desc: "With empty listen",
		exp: &Config{
			ListenAddr:      defListenAddr,
			ListenPort:      defListenPort,
			ForwardInterval: defForwardInterval,
		},
	}, {
		desc: "With empty port",
		in:   "127.0.0.2",
		exp: &Config{
			ListenAddr:      "127.0.0.2",
			ListenPort:      defListenPort,
			ForwardInterval: defForwardInterval,
		},
	}, {
		desc: "With no port",
		in:   "127.0.0.3:",
		exp: &Config{
			ListenAddr:      "127.0.0.3",
			ListenPort:      defListenPort,
			ForwardInterval: defForwardInterval,
		},
	}}

	for _, c := range cases {
		t.Log(c.desc)

		got := NewConfig()
		got.SetListen(c.in)

		test.Assert(t, "Config", c.exp, got, true)
	}
}

func TestParseAcceptBackend(t *testing.T) {
	cases := []struct {
		desc string
		in   string
		exp  *Config
	}{{
		desc: "With empty value",
		exp: &Config{
			ListenAddr:      defListenAddr,
			ListenPort:      defListenPort,
			ForwardInterval: defForwardInterval,
		},
	}, {
		desc: "With no separator",
		in:   "a ; b",
		exp: &Config{
			ListenAddr:      defListenAddr,
			ListenPort:      defListenPort,
			ForwardInterval: defForwardInterval,
			AcceptBackend: []string{
				"a ; b",
			},
		},
	}, {
		desc: "With comma at beginning and end",
		in:   ",a,b,",
		exp: &Config{
			ListenAddr:      defListenAddr,
			ListenPort:      defListenPort,
			ForwardInterval: defForwardInterval,
			AcceptBackend: []string{
				"a", "b",
			},
		},
	}}

	for _, c := range cases {
		t.Log(c.desc)

		got := NewConfig()
		got.ParseAcceptBackend(c.in)

		test.Assert(t, "Config", c.exp, got, true)
	}
}

func TestParseCaptureRequestHeader(t *testing.T) {
	cases := []struct {
		desc string
		in   string
		exp  *Config
	}{{
		desc: "With empty value",
		exp: &Config{
			ListenAddr:      defListenAddr,
			ListenPort:      defListenPort,
			ForwardInterval: defForwardInterval,
		},
	}, {
		desc: "With no separator",
		in:   "a ; b",
		exp: &Config{
			ListenAddr:      defListenAddr,
			ListenPort:      defListenPort,
			ForwardInterval: defForwardInterval,
			RequestHeaders: []string{
				"a ; b",
			},
		},
	}, {
		desc: "With separator at beginning and end",
		in:   ",a,b,",
		exp: &Config{
			ListenAddr:      defListenAddr,
			ListenPort:      defListenPort,
			ForwardInterval: defForwardInterval,
			RequestHeaders: []string{
				"a", "b",
			},
		},
	}}

	for _, c := range cases {
		t.Log(c.desc)

		got := NewConfig()
		got.ParseCaptureRequestHeader(c.in)

		test.Assert(t, "Config", c.exp, got, true)
	}
}

func TestParsePreprocessTag(t *testing.T) {
	cfg := NewConfig()

	cases := []struct {
		desc string
		in   *ini.Section
		exp  []*tagPreprocessor
	}{{
		desc: "With nil",
	}, {
		desc: "With unknown key",
		in: &ini.Section{
			Vars: []*ini.Variable{{
				KeyLower: "unknown",
			}},
		},
	}, {
		desc: "With invalid format",
		in: &ini.Section{
			Vars: []*ini.Variable{{
				KeyLower: "http_url",
				Value:    "",
			}},
		},
	}, {
		desc: "With empty regex",
		in: &ini.Section{
			Vars: []*ini.Variable{{
				KeyLower: "http_url",
				Value:    "=>",
			}},
		},
	}, {
		desc: "With valid value",
		in: &ini.Section{
			Vars: []*ini.Variable{{
				KeyLower: "http_url",
				Value:    "/[0-9]+ => /-",
			}},
		},
		exp: []*tagPreprocessor{{
			name:  "http_url",
			regex: regexp.MustCompile(`/[0-9]+`),
			repl:  "/-",
		}},
	}}

	for _, c := range cases {
		t.Log(c.desc)

		cfg.retags = nil
		cfg.parsePreprocessTag(c.in)

		test.Assert(t, "retags", c.exp, cfg.retags, true)
	}
}

func TestSetForwardInterval(t *testing.T) {
	cfg := NewConfig()

	cases := []struct {
		desc string
		in   string
		exp  time.Duration
	}{{
		desc: "With empty string",
		exp:  defForwardInterval,
	}, {
		desc: "With no interval unit",
		in:   "20",
		exp:  defForwardInterval,
	}, {
		desc: "With minus",
		in:   "-20s",
		exp:  defForwardInterval,
	}}

	for _, c := range cases {
		t.Log(c.desc)

		cfg.SetForwardInterval(c.in)

		test.Assert(t, "ForwardInterval", c.exp, cfg.ForwardInterval, true)
	}
}
