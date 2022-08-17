package haminer

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

const (
	envHostname    = "HOSTNAME"
	defHostname    = "localhost"
	defContentType = "application/octet-stream"

	influxdMeasurement = `haproxy`

	influxdTags = `,host=%s,` +
		`server=%s,` +
		`backend=%s,` +
		`frontend=%s,` +
		`http_method=%s,` +
		`http_url=%s,` +
		`http_query=%q,` +
		`http_proto=%s,` +
		`http_status=%d,` +
		`term_state=%s,` +
		`client_ip=%s,` +
		`client_port=%d`

	influxdFields = `time_req=%d,` +
		`time_wait=%d,` +
		`time_connect=%d,` +
		`time_rsp=%d,` +
		`time_all=%d,` +
		`conn_active=%d,` +
		`conn_frontend=%d,` +
		`conn_backend=%d,` +
		`conn_server=%d,` +
		`conn_retries=%d,` +
		`queue_server=%d,` +
		`queue_backend=%d,` +
		`bytes_read=%d`
)

// InfluxdClient contains HTTP connection for writing logs to Influxd.
type InfluxdClient struct {
	conn     *http.Client
	cfg      *InfluxdConfig
	hostname string
	buf      bytes.Buffer
}

// NewInfluxdClient will create, initialize, and return new Influxd client.
func NewInfluxdClient(cfg *InfluxdConfig) (cl *InfluxdClient) {
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
func (cl *InfluxdClient) Forwards(halogs []*HttpLog) {
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

	rspBody, err := io.ReadAll(httpRes.Body)
	if err != nil {
		log.Printf(`%s: %s`, logp, err)
	}

	fmt.Printf(`%s: response: %d %s\n`, logp, httpRes.StatusCode, rspBody)
}

func (cl *InfluxdClient) write(halogs []*HttpLog) (err error) {
	var (
		l *HttpLog
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
