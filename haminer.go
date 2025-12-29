// SPDX-FileCopyrightText: 2018 M. Shulhan <ms@kilabit.info>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package haminer

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"git.sr.ht/~shulhan/pakakeh.go/lib/memfs"
	"git.sr.ht/~shulhan/pakakeh.go/lib/mlog"
)

const (
	defHostname = `localhost`
	envHostname = `HOSTNAME`
)

// Version of this module and program.
var Version = `0.3.0`

var (
	_hostname string
)

// memfsDatabase embed all ".sql" files in directory _database.
// It will be used to migrate the database when using postgresql as
// forwarder.
var memfsDatabase *memfs.MemFS

// Haminer define the log consumer and producer.
type Haminer struct {
	cfg     *Config
	udpConn *net.UDPConn

	httpd *httpServer

	httpLogq  chan *HTTPLog
	ff        []Forwarder
	isRunning bool
}

func initHostname() {
	var err error

	_hostname, err = os.Hostname()
	if err != nil {
		_hostname = os.Getenv(envHostname)
	}
	if len(_hostname) == 0 {
		_hostname = defHostname
	}
}

// NewHaminer create, initialize, and return new Haminer instance. If config
// parameter is nil, it will use the default options.
func NewHaminer(cfg *Config) (h *Haminer, err error) {
	var logp = `NewHaminer`

	if cfg == nil {
		cfg = NewConfig()
	}

	h = &Haminer{
		cfg:      cfg,
		httpLogq: make(chan *HTTPLog, 30),
		ff:       make([]Forwarder, 0),
	}

	initHostname()

	if len(cfg.WuiAddress) != 0 {
		h.httpd, err = newHTTPServer(cfg)
		if err != nil {
			return nil, fmt.Errorf(`%s: %w`, logp, err)
		}
	}

	err = h.createForwarder()
	if err != nil {
		return nil, fmt.Errorf(`%s: %w`, logp, err)
	}

	return h, nil
}

func (h *Haminer) createForwarder() (err error) {
	var (
		logp = `createForwarder`

		fwCfg  *ConfigForwarder
		fwName string
	)

	for fwName, fwCfg = range h.cfg.Forwarders {
		var influxc *forwarderInfluxd

		switch fwName {
		case forwarderKindInfluxd:
			influxc = newForwarderInfluxd(fwCfg)
			if influxc != nil {
				h.ff = append(h.ff, influxc)
			}

		case forwarderKindQuestdb:
			var questc *forwarderQuestdb

			questc, err = newForwarderQuestdb(fwCfg)
			if err != nil {
				log.Printf(`%s: %s: %s`, logp, fwName, err)
				continue
			}
			if questc == nil {
				continue
			}
			h.ff = append(h.ff, questc)

		case forwarderKindPostgresql:
			if fwCfg.URL == `` {
				continue
			}

			var pgc *forwarderPostgresql

			pgc, err = newForwarderPostgresql(*fwCfg)
			if err != nil {
				log.Printf(`%s: %s: %s`, logp, fwName, err)
				continue
			}

			err = pgc.conn.Migrate(``, memfsDatabase)
			if err != nil {
				return fmt.Errorf(`%s: %w`, logp, err)
			}

			h.ff = append(h.ff, pgc)
		}
	}
	return nil
}

// Start will listen for UDP packet and start consuming log, parse, and
// publish it to analytic server.
func (h *Haminer) Start() (err error) {
	var logp = `Start`

	var udpAddr = &net.UDPAddr{
		IP:   net.ParseIP(h.cfg.listenAddr),
		Port: h.cfg.listenPort,
	}

	h.udpConn, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf(`%s: %w`, logp, err)
	}

	if h.httpd != nil {
		h.httpd.start()
	}

	h.isRunning = true

	go h.consume()
	go h.produce()
	return
}

// filter will return true if log is accepted; otherwise it will return false.
func (h *Haminer) filter(halog *HTTPLog) bool {
	if halog.BackendName == `-` {
		return false
	}
	if len(h.cfg.AcceptBackend) == 0 {
		return true
	}

	for _, be := range h.cfg.AcceptBackend {
		if halog.BackendName == be {
			return true
		}
	}

	return false
}

func (h *Haminer) consume() {
	var (
		packet = make([]byte, 4096)

		halog *HTTPLog
		err   error
		n     int
		ok    bool
	)

	for h.isRunning {
		n, err = h.udpConn.Read(packet)
		if err != nil {
			continue
		}

		if h.httpd != nil {
			select {
			case h.httpd.rawlogq <- string(packet[:n]):
			default:
				// Log queue is full.
			}
		}

		halog = ParseUDPPacket(packet[:n], h.cfg.RequestHeaders)
		if halog == nil {
			continue
		}

		ok = h.filter(halog)
		if !ok {
			continue
		}

		h.httpLogq <- halog
	}
}

func (h *Haminer) preprocess(halog *HTTPLog) {
	halog.tagHTTPURL = halog.HTTPURL
	for _, retag := range h.cfg.retags {
		halog.tagHTTPURL = retag.preprocess("http_url", halog.tagHTTPURL)
	}
}

func (h *Haminer) produce() {
	ticker := time.NewTicker(h.cfg.ForwardInterval)
	halogs := make([]*HTTPLog, 0)

	for h.isRunning {
		select {
		case halog := <-h.httpLogq:
			h.preprocess(halog)
			halogs = append(halogs, halog)

		case <-ticker.C:
			if len(halogs) == 0 {
				continue
			}

			for _, fwder := range h.ff {
				fwder.Forwards(halogs)
			}

			halogs = halogs[:0]
		}
	}
}

// Stop will close UDP server and clear all resources.
func (h *Haminer) Stop() {
	var (
		logp = `Stop`

		err error
	)

	if h.httpd != nil {
		err = h.httpd.Stop(1 * time.Second)
		if err != nil {
			mlog.Errf(`%s: %s`, logp, err)
		}
	}

	h.isRunning = false

	if h.udpConn != nil {
		err = h.udpConn.Close()
		if err != nil {
			log.Println(err)
		}
	}

	fmt.Println("Stopped")
}
