package haminer

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

const (
	envHostname    = "HOSTNAME"
	defHostname    = "localhost"
	defContentType = "application/octet-stream"
	influxdbFormat = "" +
		// measurements
		"haproxy," +
		// tags
		"host=%q," +
		"frontend=%q,backend=%q,server=%q," +
		"tag_http_status=%d,tag_http_url=%s,tag_http_method=%s" +
		" " +
		// fields
		"http_proto=%q,http_method=%q,http_url=%q," +
		"http_query=%q,http_status=%d," +
		"term_state=%q," +
		"client_ip=%q,client_port=%d," +
		"time_req=%d,time_wait=%d,time_connect=%d," +
		"time_rsp=%d,time_all=%d," +
		"conn_active=%d,conn_frontend=%d,conn_backend=%d," +
		"conn_server=%d,conn_retries=%d," +
		"queue_server=%d,queue_backend=%d," +
		"bytes_read=%d"
)

// InfluxdbClient contains HTTP connection for writing logs to Influxdb.
type InfluxdbClient struct {
	conn     *http.Client
	cfg      *InfluxdConfig
	hostname string
	buf      bytes.Buffer
}

// NewInfluxdbClient will create, initialize, and return new Influxdb client.
func NewInfluxdbClient(cfg *InfluxdConfig) (cl *InfluxdbClient) {
	cl = &InfluxdbClient{
		cfg: cfg,
	}

	cl.initHostname()
	cl.initConn()

	return
}

func (cl *InfluxdbClient) initHostname() {
	var err error

	cl.hostname, err = os.Hostname()
	if err != nil {
		cl.hostname = os.Getenv(envHostname)
	}
	if len(cl.hostname) == 0 {
		cl.hostname = defHostname
	}
}

func (cl *InfluxdbClient) initConn() {
	tr := &http.Transport{}

	cl.conn = &http.Client{
		Transport: tr,
	}
}

// Forwards implement the Forwarder interface. It will write all logs to
// Influxdb.
func (cl *InfluxdbClient) Forwards(halogs []*Halog) {
	var (
		logp = `Forwards`

		httpReq *http.Request
		httpRes *http.Response
		err     error
	)

	err = cl.write(halogs)
	if err != nil {
		log.Printf(`%s: %s`, logp, err)
		return
	}

	httpReq, err = http.NewRequest(http.MethodPost, cl.cfg.apiWrite, &cl.buf)
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

	rspBody, err := ioutil.ReadAll(httpRes.Body)
	if err != nil {
		log.Printf(`%s: ioutil.ReadAll: %s`, logp, err)
	}

	fmt.Printf(`%s: response: %d %s\n`, logp, httpRes.StatusCode, rspBody)
}

func (cl *InfluxdbClient) write(halogs []*Halog) (err error) {
	cl.buf.Reset()

	for _, l := range halogs {
		_, err = fmt.Fprintf(&cl.buf, influxdbFormat,
			// tags
			cl.hostname,
			l.FrontendName, l.BackendName, l.ServerName,
			l.HTTPStatus, l.tagHTTPURL, l.HTTPMethod,
			// fields
			l.HTTPProto, l.HTTPMethod, l.HTTPURL,
			l.HTTPQuery, l.HTTPStatus,
			l.TermState,
			l.ClientIP, l.ClientPort,
			l.TimeReq, l.TimeWait, l.TimeConnect,
			l.TimeRsp, l.TimeAll,
			l.ConnActive, l.ConnFrontend, l.ConnBackend,
			l.ConnServer, l.ConnRetries,
			l.QueueServer, l.QueueBackend,
			l.BytesRead,
		)
		if err != nil {
			return
		}

		for k, v := range l.RequestHeaders {
			_, err = fmt.Fprintf(&cl.buf, ",%s=%q", k, v)
			if err != nil {
				return
			}
		}

		_, err = fmt.Fprintf(&cl.buf, " %d\n", l.Timestamp.UnixNano())
		if err != nil {
			return
		}
	}

	return nil
}
