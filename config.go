// SPDX-FileCopyrightText: 2018 M. Shulhan <ms@kilabit.info>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package haminer

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"git.sr.ht/~shulhan/pakakeh.go/lib/ini"
)

// List of default config key values.
const (
	defListenAddr      = "127.0.0.1"
	defListenPort      = 5140
	defForwardInterval = 15 * time.Second
)

// Config define options to create and run Haminer instance.
type Config struct {
	Forwarders map[string]*ConfigForwarder `ini:"forwarder"`

	// Listen is the address where Haminer will bind and receiving
	// log from HAProxy.
	Listen string `ini:"haminer::listen"`

	listenAddr string

	// WuiAddress the address to serve for web user interface.
	WuiAddress string `ini:"haminer::wui_address"`

	// AcceptBackend list of backend to be filtered.
	AcceptBackend []string `ini:"haminer::accept_backend"`

	// List of request headers to be parsed and mapped as tags in halog
	// output.
	RequestHeaders []string `ini:"haminer::capture_request_header"`

	HTTPURL []string `ini:"preprocess:tag:http_url"`

	// retags contains list of pre-processing rules for tag.
	retags []*tagPreprocessor

	// ForwardInterval define an interval where logs will be forwarded.
	ForwardInterval time.Duration `ini:"haminer::forward_interval"`

	listenPort int

	// IsDevelopment only enabled during local development.
	IsDevelopment bool
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

		in     *ini.Ini
		fwCfg  *ConfigForwarder
		fwName string
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

	for fwName, fwCfg = range cfg.Forwarders {
		err = fwCfg.init(fwName)
		if err != nil {
			return fmt.Errorf(`%s: %s: %w`, logp, fwName, err)
		}
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
		httpURL string
		vals    []string
	)

	for _, httpURL = range cfg.HTTPURL {
		vals = strings.Split(httpURL, `=>`)
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
