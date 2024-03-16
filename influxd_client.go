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

// InfluxdClient contains HTTP connection for writing logs to Influxd.
type InfluxdClient struct {
	conn     *http.Client
	cfg      *ConfigForwarder
	hostname string
	buf      bytes.Buffer
}

// NewInfluxdClient will create, initialize, and return new Influxd client.
func NewInfluxdClient(cfg *ConfigForwarder) (cl *InfluxdClient) {
	if len(cfg.URL) == 0 {
		return nil
	}

	cl = &InfluxdClient{
		cfg: cfg,
	}

	cl.initHostname()
	cl.initConn()

	return
}

func (cl *InfluxdClient) initHostname() {
	var err error

	cl.hostname, err = os.Hostname()
	if err != nil {
		cl.hostname = os.Getenv(envHostname)
	}
	if len(cl.hostname) == 0 {
		cl.hostname = defHostname
	}
}

func (cl *InfluxdClient) initConn() {
	tr := &http.Transport{}

	cl.conn = &http.Client{
		Transport: tr,
	}
}

// Forwards implement the Forwarder interface. It will write all logs to
// Influxd.
func (cl *InfluxdClient) Forwards(halogs []*HTTPLog) {
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

func (cl *InfluxdClient) write(halogs []*HTTPLog) (err error) {
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
			l.HTTPStatus,
			l.TermState,
			l.ClientIP,
			l.ClientPort,
		)
		if err != nil {
			return err
		}

		for k, v = range l.RequestHeaders {
			_, err = fmt.Fprintf(&cl.buf, ",%s=%s", k, v)
			if err != nil {
				return err
			}
		}

		cl.buf.WriteByte(' ')

		_, err = fmt.Fprintf(&cl.buf, influxdFields,
			l.TimeReq, l.TimeWait, l.TimeConnect,
			l.TimeRsp, l.TimeAll,
			l.ConnActive, l.ConnFrontend, l.ConnBackend,
			l.ConnServer, l.ConnRetries,
			l.QueueServer, l.QueueBackend,
			l.BytesRead,
		)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(&cl.buf, " %d\n", l.Timestamp.UnixNano())
		if err != nil {
			return err
		}
	}

	return nil
}
