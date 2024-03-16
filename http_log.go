// SPDX-FileCopyrightText: 2018 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-or-later

package haminer

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	influxdMeasurement = `haproxy`

	influxdTags = `,host=%s` +
		`,server=%s` +
		`,backend=%s` +
		`,frontend=%s` +
		`,http_method=%s` +
		`,http_url=%s` +
		`,http_query=%q` +
		`,http_proto=%s` +
		`,http_status=%d` +
		`,term_state=%s` +
		`,client_ip=%s` +
		`,client_port=%d`

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

// HTTPLog contains the mapping of haproxy HTTP log format to Go struct.
//
// Reference: https://cbonte.github.io/haproxy-dconv/1.7/configuration.html#8.2.3
type HTTPLog struct {
	Timestamp time.Time

	RequestHeaders map[string]string

	ClientIP string

	FrontendName string
	BackendName  string
	ServerName   string

	CookieReq string
	CookieRsp string
	TermState string

	HTTPMethod string
	HTTPURL    string
	HTTPQuery  string
	HTTPProto  string
	tagHTTPURL string

	BytesRead int64

	ClientPort int32

	TimeReq     int32
	TimeWait    int32
	TimeConnect int32
	TimeRsp     int32
	TimeAll     int32

	ConnActive   int32
	ConnFrontend int32
	ConnBackend  int32
	ConnServer   int32
	ConnRetries  int32

	QueueServer  int32
	QueueBackend int32

	HTTPStatus int32
}

// cleanPrefix will remove `<date-time> <process-name>[pid]: ` prefix (which
// come from systemd/rsyslog) in input.
func cleanPrefix(in []byte) bool {
	start := bytes.IndexByte(in, '[')
	if start < 0 {
		return false
	}

	end := bytes.IndexByte(in[start:], ']')
	if end < 0 {
		return false
	}

	end = start + end + 3

	copy(in[0:], in[end:])

	return true
}

func parseToString(in []byte, sep byte) (string, bool) {
	end := bytes.IndexByte(in, sep)
	if end < 0 {
		return "", false
	}

	v := string(in[:end])
	copy(in, in[end+1:])

	return v, true
}

func parseToInt32(in []byte, sep byte) (int32, bool) {
	var end = bytes.IndexByte(in, sep)
	if end < 0 {
		return 0, false
	}

	var (
		v   int64
		err error
	)

	v, err = strconv.ParseInt(string(in[:end]), 10, 32)
	if err != nil {
		return 0, false
	}

	copy(in, in[end+1:])

	if v > math.MaxInt32 {
		return 0, false
	}

	var vi32 = int32(v)

	return vi32, true
}

func parseToInt64(in []byte, sep byte) (int64, bool) {
	end := bytes.IndexByte(in, sep)
	if end < 0 {
		return 0, false
	}

	v, err := strconv.ParseInt(string(in[:end]), 10, 64)
	if err != nil {
		return 0, false
	}

	copy(in, in[end+1:])

	return v, true
}

func (httpLog *HTTPLog) parseTimes(in []byte) (ok bool) {
	httpLog.TimeReq, ok = parseToInt32(in, '/')
	if !ok {
		return
	}

	httpLog.TimeWait, ok = parseToInt32(in, '/')
	if !ok {
		return
	}

	httpLog.TimeConnect, ok = parseToInt32(in, '/')
	if !ok {
		return
	}

	httpLog.TimeRsp, ok = parseToInt32(in, '/')
	if !ok {
		return
	}

	httpLog.TimeAll, ok = parseToInt32(in, ' ')
	if !ok {
		return
	}

	return
}

func (httpLog *HTTPLog) parseConns(in []byte) (ok bool) {
	httpLog.ConnActive, ok = parseToInt32(in, '/')
	if !ok {
		return
	}

	httpLog.ConnFrontend, ok = parseToInt32(in, '/')
	if !ok {
		return
	}

	httpLog.ConnBackend, ok = parseToInt32(in, '/')
	if !ok {
		return
	}

	httpLog.ConnServer, ok = parseToInt32(in, '/')
	if !ok {
		return
	}

	httpLog.ConnRetries, ok = parseToInt32(in, ' ')
	if !ok {
		return
	}

	return
}

func (httpLog *HTTPLog) parseQueue(in []byte) (ok bool) {
	httpLog.QueueServer, ok = parseToInt32(in, '/')
	if !ok {
		return
	}

	httpLog.QueueBackend, ok = parseToInt32(in, ' ')

	return
}

// parserRequestHeaders parse the request header values in log file.
// The request headers start with '{' and end with '}'.
// Each header is separated by '|'.
func (httpLog *HTTPLog) parseRequestHeaders(in []byte, reqHeaders []string) (ok bool) {
	if in[0] != '{' {
		// Skip if we did not find the beginning.
		return true
	}

	end := bytes.IndexByte(in, '}')
	// Either '}' not found or its empty as in '{}'.
	if end <= 1 {
		return
	}

	sep := []byte{'|'}
	bheaders := bytes.Split(in[1:end], sep)

	if len(reqHeaders) != len(bheaders) {
		return
	}

	httpLog.RequestHeaders = make(map[string]string)
	for x, name := range reqHeaders {
		httpLog.RequestHeaders[name] = string(bheaders[x])
	}

	copy(in, in[end+2:])

	return true
}

func (httpLog *HTTPLog) parseHTTP(in []byte) (ok bool) {
	httpLog.HTTPMethod, ok = parseToString(in, ' ')
	if !ok {
		return
	}

	v, ok := parseToString(in, ' ')
	if !ok {
		return
	}
	urlQuery := strings.SplitN(v, "?", 2)
	httpLog.HTTPURL = urlQuery[0]
	if len(urlQuery) == 2 {
		httpLog.HTTPQuery = urlQuery[1]
	}

	httpLog.HTTPProto, ok = parseToString(in, '"')

	return ok
}

// Parse will parse one line of HAProxy log format into HTTPLog.
//
// nolint: gocyclo
func (httpLog *HTTPLog) Parse(in []byte, reqHeaders []string) (ok bool) {
	var err error

	// Remove prefix from systemd/rsyslog
	ok = cleanPrefix(in)
	if !ok {
		return
	}

	// parse client IP
	httpLog.ClientIP, ok = parseToString(in, ':')
	if !ok {
		return
	}

	// parse client port
	httpLog.ClientPort, ok = parseToInt32(in, ' ')
	if !ok {
		return
	}

	// parse timestamp, remove '[' and parse until ']'
	in = in[1:]
	ts, ok := parseToString(in, ']')
	if !ok {
		return
	}

	httpLog.Timestamp, err = time.Parse(`2/Jan/2006:15:04:05.000`, ts)
	if err != nil {
		return false
	}

	// parse frontend name
	in = in[1:]
	httpLog.FrontendName, ok = parseToString(in, ' ')
	if !ok {
		return
	}

	// parse backend name
	httpLog.BackendName, ok = parseToString(in, '/')
	if !ok {
		return
	}

	// parse server name
	httpLog.ServerName, ok = parseToString(in, ' ')
	if !ok {
		return
	}

	// parse times
	ok = httpLog.parseTimes(in)
	if !ok {
		return
	}

	// parse HTTP status code
	httpLog.HTTPStatus, ok = parseToInt32(in, ' ')
	if !ok {
		return
	}

	// parse bytes read
	httpLog.BytesRead, ok = parseToInt64(in, ' ')
	if !ok {
		return
	}

	// parse request cookie
	httpLog.CookieReq, ok = parseToString(in, ' ')
	if !ok {
		return
	}

	// parse response cookie
	httpLog.CookieRsp, ok = parseToString(in, ' ')
	if !ok {
		return
	}

	// parse termination state
	httpLog.TermState, ok = parseToString(in, ' ')
	if !ok {
		return
	}

	// parse number of connections
	ok = httpLog.parseConns(in)
	if !ok {
		return
	}

	// parse number of queue state
	ok = httpLog.parseQueue(in)
	if !ok {
		return
	}

	if len(reqHeaders) > 0 {
		ok = httpLog.parseRequestHeaders(in, reqHeaders)
		if !ok {
			return
		}
	}

	// parse HTTP
	in = in[1:]
	ok = httpLog.parseHTTP(in)

	return ok
}

// ParseUDPPacket will convert UDP packet (in bytes) to instance of
// HTTPLog.
//
// It will return nil and false if UDP packet is nil, have zero length, or
// cannot be parsed (rejected).
func (httpLog *HTTPLog) ParseUDPPacket(packet []byte, reqHeaders []string) bool {
	if len(packet) == 0 {
		return false
	}

	var (
		endIdx int
		in     []byte
	)

	if packet[0] == '<' {
		endIdx = bytes.IndexByte(packet, '>')
		if endIdx < 0 {
			return false
		}

		in = packet[endIdx+1:]
	} else {
		in = packet
	}

	return httpLog.Parse(in, reqHeaders)
}

// writeIlp write the HTTP log as Influxdb Line Protocol.
func (httpLog *HTTPLog) writeIlp(out io.Writer) (err error) {
	var (
		k string
		v string
	)

	_, err = out.Write([]byte(influxdMeasurement))
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(out, influxdTags,
		// tags
		_hostname,
		httpLog.ServerName,
		httpLog.BackendName,
		httpLog.FrontendName,
		httpLog.HTTPMethod,
		httpLog.HTTPURL,
		httpLog.HTTPQuery,
		httpLog.HTTPProto,
		httpLog.HTTPStatus,
		httpLog.TermState,
		httpLog.ClientIP,
		httpLog.ClientPort,
	)
	if err != nil {
		return err
	}

	for k, v = range httpLog.RequestHeaders {
		_, err = fmt.Fprintf(out, `,%s=%s`, k, v)
		if err != nil {
			return err
		}
	}

	_, err = out.Write([]byte(` `))
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(out, influxdFields,
		httpLog.TimeReq, httpLog.TimeWait, httpLog.TimeConnect,
		httpLog.TimeRsp, httpLog.TimeAll,
		httpLog.ConnActive, httpLog.ConnFrontend, httpLog.ConnBackend,
		httpLog.ConnServer, httpLog.ConnRetries,
		httpLog.QueueServer, httpLog.QueueBackend,
		httpLog.BytesRead,
	)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(out, " %d\n", httpLog.Timestamp.UnixNano())
	if err != nil {
		return err
	}

	return nil
}
