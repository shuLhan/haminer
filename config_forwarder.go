// SPDX-FileCopyrightText: 2018 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-or-later

package haminer

import (
	"errors"
	"net/url"
)

const (
	defInfluxdBucket = `haproxy`

	influxdVersion1 = `v1`
	influxdVersion2 = `v2`

	forwarderKindInfluxd    = `influxd`
	forwarderKindQuestdb    = `questdb`
	forwarderKindPostgresql = `postgresql`
)

// ConfigForwarder contains configuration for forwarding the logs.
type ConfigForwarder struct {
	kind    string
	Version string `ini:"::version"`

	URL         string `ini:"::url"`
	apiWrite    string
	headerToken string

	Bucket string `ini:"::bucket"`

	// Fields for Influxd HTTP API v1.

	User string `ini:"::user"`
	Pass string `ini:"::pass"`

	// Fields for Influxd HTTP API v2.

	Org   string `ini:"::org"`
	Token string `ini:"::token"`
}

// init check, validate, and initialize the configuration values.
func (cfg *ConfigForwarder) init(fwName string) (err error) {
	cfg.kind = fwName

	if len(cfg.URL) == 0 {
		return
	}

	if fwName == forwarderKindInfluxd {
		return cfg.initInfluxd()
	}

	return nil
}

func (cfg *ConfigForwarder) initInfluxd() (err error) {
	switch cfg.Version {
	case influxdVersion1:
	case influxdVersion2:
	default:
		cfg.Version = influxdVersion2
	}

	if len(cfg.Bucket) == 0 {
		cfg.Bucket = defInfluxdBucket
	}

	var (
		q = url.Values{}

		surl *url.URL
	)

	surl, err = url.Parse(cfg.URL)
	if err != nil {
		return err
	}

	q.Set(`precision`, `ns`)

	if cfg.Version == influxdVersion1 {
		surl.Path = `/write`

		q.Set(`db`, cfg.Bucket)
		if len(cfg.User) > 0 && len(cfg.Pass) > 0 {
			q.Set(`u`, cfg.User)
			q.Set(`p`, cfg.Pass)
		}
	} else {
		cfg.headerToken = `Token ` + cfg.Token
		surl.Path = `/api/v2/write`

		if len(cfg.Org) == 0 {
			return errors.New(`empty organization field`)
		}

		q.Set(`org`, cfg.Org)
		q.Set(`bucket`, cfg.Bucket)
	}

	surl.RawQuery = q.Encode()

	cfg.apiWrite = surl.String()

	return nil
}
