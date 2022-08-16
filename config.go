// Copyright 2018, M. Shulhan (ms@kilabit.info).  All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package haminer

import (
	"fmt"
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

// Config define options to create and run Haminer instance.
type Config struct {
	Influxd InfluxdConfig

	// Listen is the address where Haminer will bind and receiving
	// log from HAProxy.
	Listen string `ini:"haminer::listen"`

	listenAddr string

	// AcceptBackend list of backend to be filtered.
	AcceptBackend []string `ini:"haminer::accept_backend"`

	// List of request headers to be parsed and mapped as tags in halog
	// output.
	RequestHeaders []string `ini:"haminer::capture_request_header"`

	HttpUrl []string `ini:"preprocess:tag:http_url"`

	// retags contains list of pre-processing rules for tag.
	retags []*tagPreprocessor

	// ForwardInterval define an interval where logs will be forwarded.
	ForwardInterval time.Duration `ini:"haminer::forward_interval"`

	listenPort int
}

// NewConfig will create, initialize, and return new config with default
// values.
func NewConfig() (cfg *Config) {
	return &Config{
		listenAddr:      defListenAddr,
		listenPort:      defListenPort,
		ForwardInterval: defForwardInterval,
	}
}

// Load configuration from file defined by `path`.
func (cfg *Config) Load(path string) (err error) {
	if len(path) == 0 {
		return
	}

	var (
		logp = `Load`

		in *ini.Ini
	)

	in, err = ini.Open(path)
	if err != nil {
		return fmt.Errorf(`%s: %w`, logp, err)
	}

	err = in.Unmarshal(cfg)
	if err != nil {
		return fmt.Errorf(`%s: %w`, logp, err)
	}

	if len(cfg.Listen) != 0 {
		cfg.SetListen(cfg.Listen)
	}

	err = cfg.parsePreprocessTag()
	if err != nil {
		return fmt.Errorf(`%s: %w`, logp, err)
	}

	err = cfg.Influxd.init()
	if err != nil {
		return fmt.Errorf(`%s: %w`, logp, err)
	}

	return nil
}

// SetListen will parse `v` value as "addr:port", and set config address and
// port based on it.
func (cfg *Config) SetListen(v string) {
	if len(v) == 0 {
		return
	}

	var err error

	addrPort := strings.Split(v, ":")
	switch len(addrPort) {
	case 1:
		cfg.listenAddr = addrPort[0]
	case 2:
		cfg.listenAddr = addrPort[0]
		cfg.listenPort, err = strconv.Atoi(addrPort[1])
		if err != nil {
			cfg.listenPort = defListenPort
		}
	}
}

func (cfg *Config) parsePreprocessTag() (err error) {
	var (
		logp = `parsePreprocessTag`

		retag   *tagPreprocessor
		httpUrl string
		vals    []string
	)

	for _, httpUrl = range cfg.HttpUrl {
		vals = strings.Split(httpUrl, "=>")
		if len(vals) != 2 {
			continue
		}

		retag, err = newTagPreprocessor(`http_url`, vals[0], vals[1])
		if err != nil {
			return fmt.Errorf(`%s: %w`, logp, err)
		}
		if retag == nil {
			continue
		}

		cfg.retags = append(cfg.retags, retag)
	}

	return nil
}
