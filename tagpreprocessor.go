// Copyright 2019, M. Shulhan (ms@kilabit.info).  All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package haminer

import (
	"regexp"
	"strings"
)

type tagPreprocessor struct {
	name  string
	regex *regexp.Regexp
	repl  string
}

// newTagPreprocessor create and initialize replace tag pre-processing.
// The regex and repl strings must be enclosed with double-quote, except for
// repl it can be empty.
func newTagPreprocessor(name, regex, repl string) (
	retag *tagPreprocessor, err error,
) {
	name = strings.TrimSpace(name)
	regex = strings.TrimSpace(regex)
	repl = strings.TrimSpace(repl)

	if len(name) == 0 {
		return nil, nil
	}
	if len(regex) == 0 {
		return nil, nil
	}

	re, err := regexp.Compile(regex)
	if err != nil {
		return nil, err
	}

	retag = &tagPreprocessor{
		name:  name,
		regex: re,
		repl:  repl,
	}

	return retag, nil
}

func (tagp *tagPreprocessor) preprocess(name, value string) string {
	if tagp.name != name {
		return value
	}
	out := tagp.regex.ReplaceAllString(value, tagp.repl)
	return out
}
