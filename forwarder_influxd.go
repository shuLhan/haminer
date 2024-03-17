// SPDX-FileCopyrightText: 2018 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-or-later

package haminer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

const (
	defContentType = "application/octet-stream"
)

// forwarderInfluxd contains HTTP connection for writing logs to Influxd.
type forwarderInfluxd struct {
	conn     *http.Client
	cfg      *ConfigForwarder
	hostname string
	buf      bytes.Buffer
}

// newForwarderInfluxd will create, initialize, and return new Influxd client.
func newForwarderInfluxd(cfg *ConfigForwarder) (cl *forwarderInfluxd) {
	if len(cfg.URL) == 0 {
		return nil
	}

	cl = &forwarderInfluxd{
		cfg: cfg,
	}

	cl.initHostname()
	cl.initConn()

	return
}

func (cl *forwarderInfluxd) initHostname() {
	var err error

	cl.hostname, err = os.Hostname()
	if err != nil {
		cl.hostname = os.Getenv(envHostname)
	}
	if len(cl.hostname) == 0 {
		cl.hostname = defHostname
	}
}

func (cl *forwarderInfluxd) initConn() {
	tr := &http.Transport{}

	cl.conn = &http.Client{
		Transport: tr,
	}
}

// Forwards implement the Forwarder interface. It will write all logs to
// Influxd.
func (cl *forwarderInfluxd) Forwards(halogs []*HTTPLog) {
	var (
		logp = `influxdClient: Forwards`

		httpReq *http.Request
		httpRes *http.Response
		err     error
	)

	err = cl.write(halogs)
	if err != nil {
		log.Printf(`%s: %s`, logp, err)
		return
	}

	var ctx = context.Background()

	httpReq, err = http.NewRequestWithContext(ctx, http.MethodPost, cl.cfg.apiWrite, &cl.buf)
	if err != nil {
		log.Printf(`%s: %s`, logp, err)
		return
	}

	httpReq.Header.Set(`Accept`, `application/json`)

	if cl.cfg.Version == influxdVersion1 {
		httpReq.Header.Set(`Content-Type`, defContentType)
	} else {
		httpReq.Header.Set(`Authorization`, cl.cfg.headerToken)
		httpReq.Header.Set(`Content-Type`, `text/plain; charset=utf-8`)
	}

	httpRes, err = cl.conn.Do(httpReq)
	if err != nil {
		log.Printf(`%s: %s`, logp, err)
		return
	}

	if httpRes.StatusCode >= 200 || httpRes.StatusCode <= 299 {
		return
	}

	defer func() {
		err = httpRes.Body.Close()
		if err != nil {
			log.Printf(`%s: Body.Close: %s`, logp, err)
		}
	}()

	rspBody, err := io.ReadAll(httpRes.Body)
	if err != nil {
		log.Printf(`%s: %s`, logp, err)
	}

	fmt.Printf(`%s: response: %d %s\n`, logp, httpRes.StatusCode, rspBody)
}

func (cl *forwarderInfluxd) write(halogs []*HTTPLog) (err error) {
	var (
		l *HTTPLog
		k string
		v string
	)

	cl.buf.Reset()

	for _, l = range halogs {
		cl.buf.WriteString(influxdMeasurement)

		_, err = fmt.Fprintf(&cl.buf, influxdTags,
			// tags
			cl.hostname,
			l.ServerName,
			l.BackendName,
			l.FrontendName,
			l.HTTPMethod,
			l.HTTPURL,
			l.HTTPQuery,
			l.HTTPProto,
			l.StatusCode,
			l.TerminationState,
			l.ClientIP,
			l.ClientPort,
		)
		if err != nil {
			return err
		}

		for k, v = range l.HeaderRequest {
			_, err = fmt.Fprintf(&cl.buf, ",%s=%s", k, v)
			if err != nil {
				return err
			}
		}

		cl.buf.WriteByte(' ')

		_, err = fmt.Fprintf(&cl.buf, influxdFields,
			l.TimeRequest, l.TimeWait, l.TimeConnect,
			l.TimeResponse, l.TimeAll,
			l.ConnActive, l.ConnFrontend, l.ConnBackend,
			l.ConnServer, l.Retries,
			l.ServerQueue, l.BackendQueue,
			l.BytesRead,
		)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(&cl.buf, " %d\n", l.RequestDate.UnixNano())
		if err != nil {
			return err
		}
	}

	return nil
}
