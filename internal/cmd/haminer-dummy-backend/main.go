// SPDX-FileCopyrightText: 2024 Shulhan <ms@kilabit.info>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	libhttp "git.sr.ht/~shulhan/pakakeh.go/lib/http"
	"git.sr.ht/~shulhan/pakakeh.go/lib/mlog"
)

func main() {
	go runHTTPServer(`127.0.0.1:5001`)
	go runHTTPServer(`127.0.0.1:5002`)

	var chSignal = make(chan os.Signal, 1)
	signal.Notify(chSignal, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-chSignal
	signal.Stop(chSignal)
}

func runHTTPServer(addr string) {
	var (
		logp       = `runHTTPServer ` + addr
		serverOpts = libhttp.ServerOptions{
			Address: addr,
		}

		httpServer *libhttp.Server
		err        error
	)

	httpServer, err = libhttp.NewServer(serverOpts)
	if err != nil {
		mlog.Fatalf(`%s: %w`, logp, err)
	}

	err = registerEndpoints(httpServer)
	if err != nil {
		mlog.Fatalf(`%s: %w`, logp, err)
	}

	mlog.Outf(`%s: %s`, os.Args[0], logp)

	err = httpServer.Start()
	if err != nil {
		mlog.Fatalf(`%s: %w`, logp, err)
	}
}

func registerEndpoints(httpServer *libhttp.Server) (err error) {
	var logp = `registerEndpoints`

	err = httpServer.RegisterEndpoint(libhttp.Endpoint{
		Method:       libhttp.RequestMethodGet,
		Path:         `/`,
		ResponseType: libhttp.ResponseTypePlain,
		Call:         handleGet,
	})
	if err != nil {
		return fmt.Errorf(`%s: %w`, logp, err)
	}

	err = httpServer.RegisterEndpoint(libhttp.Endpoint{
		Method:       libhttp.RequestMethodPost,
		Path:         `/`,
		ResponseType: libhttp.ResponseTypePlain,
		Call:         handlePost,
	})
	if err != nil {
		return fmt.Errorf(`%s: %w`, logp, err)
	}

	return nil
}

func handleGet(_ *libhttp.EndpointRequest) (resbody []byte, err error) {
	resbody = []byte(`Example of plain GET response`)
	return resbody, nil
}

func handlePost(_ *libhttp.EndpointRequest) (resbody []byte, err error) {
	resbody = []byte(`Example of plain POST response`)
	return resbody, nil
}
