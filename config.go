// Copyright 2018, M. Shulhan (ms@kilabit.info).  All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package haminer

import (
	"log"
	"strconv"
	"strings"

	"github.com/shuLhan/share/lib/ini"
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
	defListenAddr      = "127.0.0.1"
	defListenPort      = 5140
	defMaxBufferedLogs = 10
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
// NewConfig will create, initialize, and return new config with default
// values.
//
func NewConfig() (cfg *Config) {
	return &Config{
		ListenAddr:      defListenAddr,
		ListenPort:      defListenPort,
		MaxBufferedLogs: defMaxBufferedLogs,
	}
}

//
// Load configuration from file defined by `path`.
//
func (cfg *Config) Load(path string) {
	if len(path) == 0 {
		return
	}

	in, err := ini.Open(path)
	if err != nil {
		log.Println(err)
		return
	}

	v, _ := in.Get("haminer", "", ConfigKeyListen)
	cfg.SetListen(v)

	v, _ = in.Get("haminer", "", ConfigKeyAcceptBackend)
	cfg.ParseAcceptBackend(v)

	v, _ = in.Get("haminer", "", ConfigKeyCaptureRequestHeader)
	cfg.ParseCaptureRequestHeader(v)

	v, _ = in.Get("haminer", "", ConfigKeyInfluxAPIWrite)
	if len(v) > 0 {
		cfg.InfluxAPIWrite = v
	}
}

//
// SetListen will parse `v` value as "addr:port", and set config address and
// port based on it.
//
func (cfg *Config) SetListen(v string) {
	if len(v) == 0 {
		return
	}

	var err error

	addrPort := strings.Split(v, ":")
	switch len(addrPort) {
	case 1:
		cfg.ListenAddr = addrPort[0]
	case 2:
		cfg.ListenAddr = addrPort[0]
		cfg.ListenPort, err = strconv.Atoi(addrPort[1])
		if err != nil {
			cfg.ListenPort = defListenPort
		}
	}
}

func (cfg *Config) ParseAcceptBackend(v string) {
	v = strings.TrimSpace(v)
	if len(v) == 0 {
		return
	}

	for _, v = range strings.Split(v, ",") {
		if len(v) == 0 {
			continue
		}
		cfg.AcceptBackend = append(cfg.AcceptBackend, strings.TrimSpace(v))
	}
}

//
// ParseCaptureRequestHeader Parse request header names where each name is
// separated by "|".
//
func (cfg *Config) ParseCaptureRequestHeader(v string) {
	v = strings.TrimSpace(v)
	if len(v) == 0 {
		return
	}

	headers := strings.Split(v, "|")
	for x := 0; x < len(headers); x++ {
		headers[x] = strings.TrimSpace(headers[x])
		if len(headers[x]) == 0 {
			continue
		}
		cfg.RequestHeaders = append(cfg.RequestHeaders, headers[x])
	}
}
