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
		"http_status_code=%d" +
		" " +
		// fields
		"http_proto=%q,http_method=%q,http_url=%q," +
		"http_query=\"%s\",http_status=%d," +
		"term_state=%q," +
		"client_ip=%q,client_port=%d," +
		"time_req=%d,time_wait=%d,time_connect=%d," +
		"time_rsp=%d,time_all=%d," +
		"conn_active=%d,conn_frontend=%d,conn_backend=%d," +
		"conn_server=%d,conn_retries=%d," +
		"queue_server=%d,queue_backend=%d," +
		"bytes_read=%d"
)

//
// InfluxdbClient contains HTTP connection for writing logs to Influxdb.
//
type InfluxdbClient struct {
	conn     *http.Client
	apiWrite string
	hostname string
	buf      bytes.Buffer
}

//
// NewInfluxdbClient will create, initialize, and return new Influxdb client.
//
func NewInfluxdbClient(apiWrite string) (cl *InfluxdbClient) {
	cl = &InfluxdbClient{
		apiWrite: apiWrite,
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

//
// Forwards implement the Forwarder interface. It will write all logs to
// Influxdb.
//
func (cl *InfluxdbClient) Forwards(halogs []*Halog) {
	lsrc := "InfluxdbClient.Forwards"
	err := cl.write(halogs)
	if err != nil {
		log.Printf("InfluxdbClient.write: %s", err)
		return
	}

	rsp, err := cl.conn.Post(cl.apiWrite, defContentType, &cl.buf)
	if err != nil {
		log.Printf("InfluxdbClient.Forwards: %s", err)
		return
	}

	if rsp.StatusCode >= 200 || rsp.StatusCode <= 299 {
		return
	}

	defer func() {
		errClose := rsp.Body.Close()
		if errClose != nil {
			log.Printf("%s: Body.Close: %s\n", lsrc, err)
		}
	}()

	rspBody, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		log.Printf("%s: ioutil.ReadAll: %s", lsrc, err)
	}

	fmt.Printf("%s: response: %d %s\n", lsrc, rsp.StatusCode, rspBody)
}

func (cl *InfluxdbClient) write(halogs []*Halog) (err error) {
	cl.buf.Reset()

	for _, l := range halogs {
		_, err = fmt.Fprintf(&cl.buf, influxdbFormat,
			// tags
			cl.hostname,
			l.FrontendName, l.BackendName, l.ServerName,
			l.HTTPStatus,
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

	return
}
