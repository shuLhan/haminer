package haminer

import (
	"errors"
	"net/url"
)

const (
	defInfluxdBucket = `haproxy`

	influxdVersion1 = `v1`
	influxdVersion2 = `v2`
)

// InfluxdConfig contains configuration for forwarding the logs to Influxd.
type InfluxdConfig struct {
	Version string `ini:"forwarder:influxd:version"`

	Url         string `ini:"forwarder:influxd:url"`
	apiWrite    string
	headerToken string

	Bucket string `ini:"forwarder:influxd:bucket"`

	// Fields for HTTP API v1.

	User string `ini:"forwarder:influxd:user"`
	Pass string `ini:"forwarder:influxd:pass"`

	// Fields for HTTP API v2.

	Org   string `ini:"forwarder:influxd:org"`
	Token string `ini:"forwarder:influxd:token"`
}

// init check, validate, and initialize the configuration values.
func (cfg *InfluxdConfig) init() (err error) {
	if len(cfg.Url) == 0 {
		return
	}

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

		url *url.URL
	)

	url, err = url.Parse(cfg.Url)
	if err != nil {
		return err
	}

	q.Set(`precision`, `ns`)

	if cfg.Version == influxdVersion1 {
		url.Path = `/write`

		q.Set(`db`, cfg.Bucket)
		if len(cfg.User) > 0 && len(cfg.Pass) > 0 {
			q.Set(`u`, cfg.User)
			q.Set(`p`, cfg.Pass)
		}
	} else {
		cfg.headerToken = `Token ` + cfg.Token
		url.Path = `/api/v2/write`

		if len(cfg.Org) == 0 {
			return errors.New(`empty organization field`)
		}

		q.Set(`org`, cfg.Org)
		q.Set(`bucket`, cfg.Bucket)
	}

	url.RawQuery = q.Encode()

	cfg.apiWrite = url.String()

	return nil
}
