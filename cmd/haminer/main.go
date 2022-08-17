// Copyright 2018, M. Shulhan (ms@kilabit.info).  All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"

	"git.sr.ht/~shulhan/haminer"
)

const (
	defLogPrefix = "haminer: "
	defConfig    = "/etc/haminer.conf"
)

func main() {
	var (
		cfg        *haminer.Config
		err        error
		flagConfig string
	)

	log.SetPrefix(defLogPrefix)

	cfg = haminer.NewConfig()

	flag.StringVar(&flagConfig, `config`, defConfig, `Path to configuration`)

	flag.Parse()

	err = cfg.Load(flagConfig)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Starting Haminer with config: %+v\n", cfg)

	h := haminer.NewHaminer(cfg)

	err = h.Start()
	if err != nil {
		log.Fatal(err)
	}
}
