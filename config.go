// Copyright 2018, M. Shulhan (ms@kilabit.info).  All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package haminer

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/shuLhan/share/lib/ini"
)

// List of config keys.
const (
	ConfigKeyAcceptBackend        = "accept_backend"
	ConfigKeyCaptureRequestHeader = "capture_request_header"
	ConfigKeyForwardInterval      = "forward_interval"
	ConfigKeyInfluxAPIWrite       = "influxdb_api_write"
	ConfigKeyListen               = "listen"
)

// List of default config key values.
const (
	defListenAddr      = "127.0.0.1"
	defListenPort      = 5140
	defForwardInterval = 15 * time.Second
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

	// ForwardInterval define an interval where logs will be forwarded.
	ForwardInterval time.Duration

	// List of request headers to be parsed and mapped as keys in halog
	// output.
	RequestHeaders []string

	// InfluxAPIWrite define HTTP API to write to Influxdb.
	InfluxAPIWrite string

	// retags contains list of pre-processing rules for tag.
	retags []*tagPreprocessor
}

//
// NewConfig will create, initialize, and return new config with default
// values.
//
func NewConfig() (cfg *Config) {
	return &Config{
		ListenAddr:      defListenAddr,
		ListenPort:      defListenPort,
		ForwardInterval: defForwardInterval,
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

	v, _ = in.Get("haminer", "", ConfigKeyForwardInterval)
	cfg.SetForwardInterval(v)

	sec := in.GetSection("preprocess", "tag")

	cfg.parsePreprocessTag(sec)
}

//
// SetForwardInterval set forward interval using string formatted, e.g. "20s"
// where "s" represent unit time in "second".
//
func (cfg *Config) SetForwardInterval(v string) {
	if len(v) == 0 {
		return
	}

	var err error

	cfg.ForwardInterval, err = time.ParseDuration(v)
	if err != nil {
		log.Println("SetForwardInterval: ", err)
	}
	if cfg.ForwardInterval < defForwardInterval {
		cfg.ForwardInterval = defForwardInterval
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
// ParseCaptureRequestHeader parse request header names where each name is
// separated by ",".
//
func (cfg *Config) ParseCaptureRequestHeader(v string) {
	v = strings.TrimSpace(v)
	if len(v) == 0 {
		return
	}

	headers := strings.Split(v, ",")
	for x := 0; x < len(headers); x++ {
		headers[x] = strings.TrimSpace(headers[x])
		if len(headers[x]) == 0 {
			continue
		}
		cfg.RequestHeaders = append(cfg.RequestHeaders, headers[x])
	}
}

func (cfg *Config) parsePreprocessTag(sec *ini.Section) {
	if sec == nil {
		return
	}

	for _, v := range sec.Vars {
		if len(v.KeyLower) == 0 {
			continue
		}
		if v.KeyLower != "http_url" {
			log.Printf("parsePreprocessTag: unknown tag %q\n",
				v.KeyLower)
			continue
		}

		rep := strings.Split(v.Value, "=>")
		if len(rep) != 2 {
			log.Printf("parsePreprocessTag: invalid format %q\n",
				v.Value)
			continue
		}

		retag, err := newTagPreprocessor(v.KeyLower, rep[0], rep[1])
		if err != nil {
			log.Printf("parsePreprocessTag: %s\n", err)
			continue
		}

		cfg.retags = append(cfg.retags, retag)
	}
}
