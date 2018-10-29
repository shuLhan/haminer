// Copyright 2018, M. Shulhan (ms@kilabit.info).  All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package haminer

import (
	"bytes"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

// List of config keys.
const (
	ConfigKeyAcceptBackend        = "accept_backend"
	ConfigKeyCaptureRequestHeader = "capture_request_header"
	ConfigKeyInfluxAPIWrite       = "influxdb_api_write"
	ConfigKeyListen               = "listen"
)

// List of default config key values.
const (
	DefListenAddr      = "127.0.0.1"
	DefListenPort      = 5140
	DefInfluxAPIWrite  = "http://127.0.0.1:8086/write?db=haproxy"
	DefMaxBufferedLogs = 10
)

//
// Config define options to create and run Haminer instance.
//
type Config struct {
	// ListenAddr is an IP address where Haminer will bind and receiving
	// log from HAProxy.
	ListenAddr string
	ListenPort int

	// AcceptBackend list of backend to be filtered.
	AcceptBackend []string

	// List of request headers to be parsed and mapped as keys in halog
	// output.
	RequestHeaders []string

	// InfluxAPIWrite define HTTP API to write to Influxdb.
	InfluxAPIWrite string

	// MaxBufferedLogs define a number of logs that will be keep in buffer
	// before being forwarded.
	MaxBufferedLogs int
}

//
// NewConfig will create, initialize, and return new config with defautl
// values.
//
func NewConfig() (cfg *Config) {
	return &Config{
		ListenAddr:      DefListenAddr,
		ListenPort:      DefListenPort,
		MaxBufferedLogs: DefMaxBufferedLogs,
	}
}

//
// SetListen will parse `v` value as "addr:port", and set config address and port
// based on it.
//
func (cfg *Config) SetListen(v string) {
	var err error

	addrPort := strings.Split(v, ":")
	switch len(addrPort) {
	case 0:
		return
	case 1:
		cfg.ListenAddr = addrPort[0]
	case 2:
		cfg.ListenAddr = addrPort[0]
		cfg.ListenPort, err = strconv.Atoi(addrPort[1])
		if err != nil {
			cfg.ListenPort = DefListenPort
		}
	}
}

//
// parseCaptureRequestHeader Parse request header names where each name is
// separated by "|".
//
func (cfg *Config) parseCaptureRequestHeader(v []byte) {
	sep := []byte{'|'}
	headers := bytes.Split(v, sep)
	for x := 0; x < len(headers); x++ {
		headers[x] = bytes.TrimSpace(headers[x])
		cfg.RequestHeaders = append(cfg.RequestHeaders, string(headers[x]))
	}
}

//
// Load will read configuration from file defined by `path`.
//
func (cfg *Config) Load(path string) {
	bb, err := ioutil.ReadFile(path)
	if err != nil {
		log.Println(err)
		return
	}

	lines := bytes.Split(bb, []byte("\n"))

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		if line[0] == '#' {
			continue
		}

		kv := bytes.SplitN(line, []byte("="), 2)
		if len(kv) != 2 {
			continue
		}

		switch string(kv[0]) {
		case ConfigKeyListen:
			cfg.SetListen(string(kv[1]))
		case ConfigKeyCaptureRequestHeader:
			cfg.parseCaptureRequestHeader(kv[1])
		case ConfigKeyAcceptBackend:
			v := string(bytes.TrimSpace(kv[1]))
			if len(v) > 0 {
				cfg.AcceptBackend = strings.Split(v, ",")
			}
		case ConfigKeyInfluxAPIWrite:
			cfg.InfluxAPIWrite = string(kv[1])
		}
	}
}
