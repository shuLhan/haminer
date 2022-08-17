package haminer

import (
	"errors"
	"net/url"
)

const (
	defInfluxdBucket = `haproxy`

	influxdVersion1 = `v1`
	influxdVersion2 = `v2`

	forwarderInfluxd = `influxd`
	forwarderQuestdb = `questdb`
)

// ConfigForwarder contains configuration for forwarding the logs.
type ConfigForwarder struct {
	Version string `ini:"::version"`

	Url         string `ini:"::url"`
	apiWrite    string
	headerToken string

	Bucket string `ini:"::bucket"`

	// Fields for HTTP API v1.

	User string `ini:"::user"`
	Pass string `ini:"::pass"`

	// Fields for HTTP API v2.

	Org   string `ini:"::org"`
	Token string `ini:"::token"`
}

// init check, validate, and initialize the configuration values.
func (cfg *ConfigForwarder) init(fwName string) (err error) {
	if len(cfg.Url) == 0 {
		return
	}

	if fwName == forwarderInfluxd {
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

	surl, err = url.Parse(cfg.Url)
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
