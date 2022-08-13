// Copyright 2018, M. Shulhan (ms@kilabit.info).  All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/shuLhan/haminer"
)

const (
	defLogPrefix = "haminer: "
	defConfig    = "/etc/haminer.conf"
)

func initConfig() (cfg *haminer.Config, err error) {
	var (
		flagConfig         string
		flagListen         string
		flagAcceptBackend  string
		flagInfluxAPIWrite string
	)

	log.SetPrefix(defLogPrefix)

	cfg = haminer.NewConfig()

	flag.StringVar(&flagConfig, "config", defConfig,
		"Load configuration from file (default to '/etc/haminer.conf')",
	)
	flag.StringVar(&flagListen, haminer.ConfigKeyListen, "",
		"Listen for HAProxy log using UDP at ADDRESS:PORT",
	)
	flag.StringVar(&flagAcceptBackend, haminer.ConfigKeyAcceptBackend, "",
		"List of accepted backend to be filtered (comma separated)",
	)
	flag.StringVar(&flagInfluxAPIWrite, haminer.ConfigKeyInfluxAPIWrite,
		"",
		"HTTP API endpoint to write to Influxdb",
	)

	flag.Parse()

	if len(flagConfig) > 0 {
		err = cfg.Load(flagConfig)
		if err != nil {
			return nil, err
		}
	}
	if len(flagListen) > 0 {
		cfg.SetListen(flagListen)
	}
	if len(flagAcceptBackend) > 0 {
		cfg.AcceptBackend = strings.Split(flagAcceptBackend, ",")
	}
	if len(flagInfluxAPIWrite) > 0 {
		cfg.InfluxAPIWrite = flagInfluxAPIWrite
	}

	return cfg, nil
}

func main() {
	var (
		cfg *haminer.Config
		err error
	)

	cfg, err = initConfig()
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
