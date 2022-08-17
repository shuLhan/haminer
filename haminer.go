// Copyright 2018, M. Shulhan (ms@kilabit.info).  All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package haminer

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Haminer define the log consumer and producer.
type Haminer struct {
	cfg       *Config
	udpConn   *net.UDPConn
	chSignal  chan os.Signal
	chHttpLog chan *HttpLog
	ff        []Forwarder
	isRunning bool
}

// NewHaminer create, initialize, and return new Haminer instance. If config
// parameter is nil, it will use the default options.
func NewHaminer(cfg *Config) (h *Haminer) {
	if cfg == nil {
		cfg = NewConfig()
	}

	h = &Haminer{
		cfg:       cfg,
		chSignal:  make(chan os.Signal, 1),
		chHttpLog: make(chan *HttpLog, 30),
		ff:        make([]Forwarder, 0),
	}

	signal.Notify(h.chSignal, syscall.SIGHUP, syscall.SIGINT,
		syscall.SIGTERM, syscall.SIGQUIT)

	h.createForwarder()

	return
}

func (h *Haminer) createForwarder() {
	if len(h.cfg.Influxd.Url) == 0 {
		return
	}

	var (
		fwder = NewInfluxdClient(&h.cfg.Influxd)
	)

	h.ff = append(h.ff, fwder)
}

// Start will listen for UDP packet and start consuming log, parse, and
// publish it to analytic server.
func (h *Haminer) Start() (err error) {
	udpAddr := &net.UDPAddr{
		IP:   net.ParseIP(h.cfg.listenAddr),
		Port: h.cfg.listenPort,
	}

	h.udpConn, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		return
	}

	h.isRunning = true

	go h.consume()
	go h.produce()

	<-h.chSignal

	h.Stop()

	return
}

// filter will return true if log is accepted; otherwise it will return false.
func (h *Haminer) filter(halog *HttpLog) bool {
	if halog == nil {
		return false
	}
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

		halog *HttpLog
		err   error
		n     int
		ok    bool
	)

	for h.isRunning {
		n, err = h.udpConn.Read(packet)
		if err != nil {
			continue
		}

		halog = &HttpLog{}

		ok = halog.ParseUDPPacket(packet[:n], h.cfg.RequestHeaders)
		if !ok {
			continue
		}

		ok = h.filter(halog)
		if !ok {
			continue
		}

		h.chHttpLog <- halog
	}
}

func (h *Haminer) preprocess(halog *HttpLog) {
	halog.tagHTTPURL = halog.HTTPURL
	for _, retag := range h.cfg.retags {
		halog.tagHTTPURL = retag.preprocess("http_url", halog.tagHTTPURL)
	}
}

func (h *Haminer) produce() {
	ticker := time.NewTicker(h.cfg.ForwardInterval)
	halogs := make([]*HttpLog, 0)

	for h.isRunning {
		select {
		case halog := <-h.chHttpLog:
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
	h.isRunning = false

	signal.Stop(h.chSignal)

	if h.udpConn != nil {
		err := h.udpConn.Close()
		if err != nil {
			log.Println(err)
		}
	}

	fmt.Println("Stopped")
}
