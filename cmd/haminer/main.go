// SPDX-FileCopyrightText: 2018 M. Shulhan <ms@kilabit.info>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"git.sr.ht/~shulhan/haminer"
)

const (
	defLogPrefix = "haminer: "
	defConfig    = "/etc/haminer.conf"
)

func main() {
	var (
		chSignal = make(chan os.Signal, 1)
		cfg      = haminer.NewConfig()

		err        error
		flagConfig string
	)

	log.SetPrefix(defLogPrefix)

	flag.StringVar(&flagConfig, `config`, defConfig, `Path to configuration`)
	flag.BoolVar(&cfg.IsDevelopment, `dev`, false, `Enable development mode`)

	flag.Parse()

	err = cfg.Load(flagConfig)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Starting Haminer with config: %+v\n", cfg)

	var h *haminer.Haminer

	h, err = haminer.NewHaminer(cfg)
	if err != nil {
		log.Fatal(err)
	}

	signal.Notify(chSignal, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		err = h.Start()
		if err != nil {
			log.Fatal(err)
		}
	}()

	<-chSignal
	h.Stop()
	signal.Stop(chSignal)
}
