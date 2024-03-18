// SPDX-FileCopyrightText: 2024 Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-or-later

package haminer

import (
	"fmt"
	"sync"

	libhttp "git.sr.ht/~shulhan/pakakeh.go/lib/http"
	"git.sr.ht/~shulhan/pakakeh.go/lib/memfs"
	"git.sr.ht/~shulhan/pakakeh.go/lib/mlog"
)

const (
	pathAPILogTail = `/api/log/tail`
)

var memfsWUI *memfs.MemFS

type httpServer struct {
	*libhttp.Server

	// rawlogq channel that receive raw log to be published by HTTP API
	// apiLogTail.
	rawlogq chan string

	tailer    map[int64]chan string
	tailerIdx int64
	tailerMtx sync.Mutex
}

func newHTTPServer(cfg *Config) (httpd *httpServer, err error) {
	var logp = `newHTTPServer`

	if memfsWUI != nil {
		memfsWUI.Opts.TryDirect = cfg.IsDevelopment
	}

	httpd = &httpServer{
		rawlogq: make(chan string, 512),
		tailer:  make(map[int64]chan string),
	}

	var opts = libhttp.ServerOptions{
		Memfs:   memfsWUI,
		Address: cfg.WuiAddress,
	}

	httpd.Server, err = libhttp.NewServer(opts)
	if err != nil {
		return nil, fmt.Errorf(`%s: %w`, logp, err)
	}

	err = httpd.registerEndpoints()
	if err != nil {
		return nil, fmt.Errorf(`%s: %w`, logp, err)
	}

	return httpd, nil
}

func (httpd *httpServer) logPublisher() {
	var (
		rawlog string
		tailer chan string
	)
	for rawlog = range httpd.rawlogq {
		httpd.tailerMtx.Lock()
		for _, tailer = range httpd.tailer {
			tailer <- rawlog
		}
		httpd.tailerMtx.Unlock()
	}
}

func (httpd *httpServer) registerEndpoints() (err error) {
	var logp = `registerEndpoints`

	err = httpd.RegisterSSE(libhttp.SSEEndpoint{
		Call: httpd.apiLogTail,
		Path: pathAPILogTail,
	})
	if err != nil {
		return fmt.Errorf(`%s: %w`, logp, err)
	}
	return nil
}

func (httpd *httpServer) registerTailer() (idx int64, tailer chan string) {
	var ok bool

	httpd.tailerMtx.Lock()

	for {
		_, ok = httpd.tailer[httpd.tailerIdx]
		if !ok {
			// Index not exist, use it.
			break
		}
		httpd.tailerIdx++
	}
	idx = httpd.tailerIdx
	tailer = make(chan string, 512)
	httpd.tailer[idx] = tailer

	httpd.tailerMtx.Unlock()

	return idx, tailer
}

func (httpd *httpServer) unregisterTailer(idx int64) {
	var (
		tailer chan string
		ok     bool
	)

	httpd.tailerMtx.Lock()

	tailer, ok = httpd.tailer[idx]
	if ok {
		close(tailer)
		delete(httpd.tailer, idx)
	}

	httpd.tailerMtx.Unlock()
}

func (httpd *httpServer) start() (err error) {
	var logp = `start`

	mlog.Outf(`%s: starting HTTP server at http://%s`, logp, httpd.Options.Address)

	go func() {
		err = httpd.Server.Start()
		if err != nil {
			mlog.Errf(`%s: %s`, logp, err)
		}
	}()
	go httpd.logPublisher()

	return nil
}

// apiLogTail tail the log using Server-Sent event.
func (httpd *httpServer) apiLogTail(sse *libhttp.SSEConn) {
	var (
		logp = `apiLogTail`

		tailer chan string
		rawlog string
		idx    int64
		err    error
	)

	idx, tailer = httpd.registerTailer()

	for rawlog = range tailer {
		mlog.Outf(`%s: %s`, logp, rawlog)

		err = sse.WriteEvent(``, rawlog, nil)
		if err != nil {
			mlog.Errf(`%s: %s`, logp, err)
			httpd.unregisterTailer(idx)
			return
		}
	}
}
