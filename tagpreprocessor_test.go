// Copyright 2019, M. Shulhan (ms@kilabit.info).  All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package haminer

import (
	"regexp"
	"testing"

	"github.com/shuLhan/share/lib/test"
)

func TestNewTagPreprocessor(t *testing.T) {
	cases := []struct {
		desc   string
		name   string
		regex  string
		repl   string
		exp    *tagPreprocessor
		expErr string
	}{{
		desc:   "With empty name",
		expErr: "newTagPreprocessor: empty name parameter",
	}, {
		desc:   "With empty regex",
		name:   "http_url",
		expErr: "newTagPreprocessor: empty regex parameter",
	}, {
		desc:   "With invalid regex",
		name:   "http_url",
		regex:  `/[a-z]*+`,
		expErr: "error parsing regexp: invalid nested repetition operator: `*+`",
	}, {
		desc:  "With valid parameters",
		name:  "http_url",
		regex: `/[0-9]+`,
		repl:  `/-`,
		exp: &tagPreprocessor{
			name:  "http_url",
			regex: regexp.MustCompile(`/[0-9]+`),
			repl:  `/-`,
		},
	}}

	for _, c := range cases {
		t.Log(c.desc)

		got, err := newTagPreprocessor(c.name, c.regex, c.repl)
		if err != nil {
			test.Assert(t, "error", c.expErr, err.Error(), true)
			continue
		}

		test.Assert(t, "TagPreprocessor", c.exp, got, true)
	}
}

func TestPreprocess(t *testing.T) {
	reIDUUID := regexp.MustCompile(`/[0-9]+-\w+-\w+-\w+-\w+-\w+`)
	reUUID := regexp.MustCompile(`/-?\w+-\w+-\w+-\w+-\w+`)
	reID := regexp.MustCompile(`/[0-9]+`)

	retags := []*tagPreprocessor{{
		name:  "http_url",
		regex: reIDUUID,
		repl:  `/-`,
	}, {
		name:  "http_url",
		regex: reUUID,
		repl:  `/-`,
	}, {
		name:  "http_url",
		regex: reID,
		repl:  `/-`,
	}}

	cases := []struct {
		desc string
		name string
		in   string
		exp  string
	}{{
		desc: "With empty name",
	}, {
		desc: "With different name",
		name: "tag",
		in:   "/test/1000",
		exp:  "/test/1000",
	}, {
		desc: "With one replacement",
		name: "http_url",
		in:   "/test/1000",
		exp:  "/test/-",
	}, {
		desc: "With two replacements",
		name: "http_url",
		in:   "/test/1000/param/9845a0b4-f4c3-4600-af13-45b5b0e61630",
		exp:  "/test/-/param/-",
	}, {
		desc: "With three replacements",
		name: "http_url",
		in:   "/group/9845a0b4-f4c3-4600-af13-45b5b0e61630/test/1000/param/1-9845a0b4-f4c3-4600-af13-45b5b0e61630",
		exp:  "/group/-/test/-/param/-",
	}, {
		desc: "With invalid UUID",
		name: "http_url",
		in:   `/v1/threads/900001-fefcd79-0b03-4794-ae90-abe4b51dec75/count-previous/90001`,
		exp:  `/v1/threads/-/count-previous/-`,
	}, {
		desc: "With missing ID",
		name: "http_url",
		in:   `/v1/threads/-fefcd79-0b03-4794-ae90-abe4b51dec75/count-previous/90001`,
		exp:  `/v1/threads/-/count-previous/-`,
	}}

	for _, c := range cases {
		t.Log(c.desc)

		got := c.in

		for _, tagp := range retags {
			got = tagp.preprocess(c.name, got)
			t.Log("got: ", got)
		}

		test.Assert(t, "preprocess", c.exp, got, true)
	}
}
