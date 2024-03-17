// SPDX-FileCopyrightText: 2018 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-or-later

package haminer

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	libsql "git.sr.ht/~shulhan/pakakeh.go/lib/sql"
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

const tableNameHTTPLog = `http_log`

// HTTPLog contains the mapping of haproxy HTTP log format to Go struct.
//
// Reference: https://cbonte.github.io/haproxy-dconv/1.7/configuration.html#8.2.3
type HTTPLog struct {
	RequestDate time.Time

	HeaderRequest  map[string]string
	HeaderResponse map[string]string

	rawHeaderRequest  string
	rawHeaderResponse string

	ClientIP string

	FrontendName string
	BackendName  string
	ServerName   string

	HTTPProto  string
	HTTPMethod string
	HTTPURL    string
	HTTPQuery  string
	tagHTTPURL string

	CookieRequest    string
	CookieResponse   string
	TerminationState string

	BytesRead int64

	StatusCode int32
	ClientPort int32

	TimeRequest  int32
	TimeWait     int32
	TimeConnect  int32
	TimeResponse int32
	TimeAll      int32

	ConnActive   int32
	ConnFrontend int32
	ConnBackend  int32
	ConnServer   int32
	Retries      int32

	ServerQueue  int32
	BackendQueue int32
}

// listHTTPLog fetch all HTTPLog record from database.
func listHTTPLog(dbc libsql.Session) (list []HTTPLog, err error) {
	var (
		logp    = `ListHTTPLog`
		httpLog = HTTPLog{}
		meta    = httpLog.generateSQLMeta(libsql.DriverNamePostgres, libsql.DMLKindSelect)
	)

	var q = fmt.Sprintf(`SELECT %s FROM %s ORDER BY request_date DESC;`,
		meta.Names(), tableNameHTTPLog)

	var rows *sql.Rows

	rows, err = dbc.Query(q)
	if err != nil {
		return nil, fmt.Errorf(`%s: %w`, logp, err)
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(meta.ListValue...)
		if err != nil {
			return nil, fmt.Errorf(`%s: %w`, logp, err)
		}

		var dup = httpLog
		list = append(list, dup)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf(`%s: %w`, logp, err)
	}

	return list, nil
}

// ParseUDPPacket convert UDP packet (in bytes) to instance of HTTPLog.
//
// It will return nil if UDP packet is nil, have zero length, or cannot be
// parsed (rejected).
func ParseUDPPacket(packet []byte, reqHeaders []string) (httpLog *HTTPLog) {
	if len(packet) == 0 {
		return nil
	}

	if packet[0] == '<' {
		var endIdx = bytes.IndexByte(packet, '>')
		if endIdx < 0 {
			return nil
		}
		packet = packet[endIdx+1:]
	}

	return Parse(packet, reqHeaders)
}

// Parse single line of HAProxy log format into HTTPLog.
//
// nolint: gocyclo
func Parse(in []byte, reqHeaders []string) (httpLog *HTTPLog) {
	in = cleanPrefix(in)
	if in == nil {
		return nil
	}

	var ok bool

	httpLog = &HTTPLog{}

	httpLog.ClientIP, ok = parseToString(in, ':')
	if !ok {
		return nil
	}

	httpLog.ClientPort, ok = parseToInt32(in, ' ')
	if !ok {
		return nil
	}

	// parse timestamp, remove '[' and parse until ']'
	in = in[1:]
	ts, ok := parseToString(in, ']')
	if !ok {
		return nil
	}

	var err error

	httpLog.RequestDate, err = time.Parse(`2/Jan/2006:15:04:05.000`, ts)
	if err != nil {
		return nil
	}

	in = in[1:]
	httpLog.FrontendName, ok = parseToString(in, ' ')
	if !ok {
		return nil
	}

	httpLog.BackendName, ok = parseToString(in, '/')
	if !ok {
		return nil
	}

	httpLog.ServerName, ok = parseToString(in, ' ')
	if !ok {
		return nil
	}

	ok = httpLog.parseConnectionTimes(in)
	if !ok {
		return nil
	}

	httpLog.StatusCode, ok = parseToInt32(in, ' ')
	if !ok {
		return nil
	}

	httpLog.BytesRead, ok = parseToInt64(in, ' ')
	if !ok {
		return nil
	}

	httpLog.CookieRequest, ok = parseToString(in, ' ')
	if !ok {
		return nil
	}
	httpLog.CookieResponse, ok = parseToString(in, ' ')
	if !ok {
		return nil
	}

	httpLog.TerminationState, ok = parseToString(in, ' ')
	if !ok {
		return nil
	}

	ok = httpLog.parseConns(in)
	if !ok {
		return nil
	}

	ok = httpLog.parseQueue(in)
	if !ok {
		return nil
	}

	if len(reqHeaders) > 0 {
		ok = httpLog.parseHeaderRequest(in, reqHeaders)
		if !ok {
			return nil
		}
	}

	in = in[1:]
	ok = httpLog.parseHTTP(in)
	if !ok {
		return nil
	}

	return httpLog
}

// cleanPrefix will remove `<date-time> <process-name>[pid]: ` prefix which
// come from systemd/rsyslog in input.
func cleanPrefix(in []byte) []byte {
	var start = bytes.IndexByte(in, '[')
	if start < 0 {
		return nil
	}

	var end = bytes.IndexByte(in[start:], ']')
	if end < 0 {
		return nil
	}

	end = start + end + 3

	return in[end:]
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

func (httpLog *HTTPLog) parseConnectionTimes(in []byte) (ok bool) {
	httpLog.TimeRequest, ok = parseToInt32(in, '/')
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

	httpLog.TimeResponse, ok = parseToInt32(in, '/')
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

	httpLog.Retries, ok = parseToInt32(in, ' ')
	if !ok {
		return
	}

	return
}

func (httpLog *HTTPLog) generateSQLMeta(driver string, kind libsql.DMLKind) (meta *libsql.Meta) {
	meta = libsql.NewMeta(driver, kind)

	meta.Bind(`request_date`, &httpLog.RequestDate)
	meta.Bind(`client_ip`, &httpLog.ClientIP)

	meta.Bind(`frontend_name`, &httpLog.FrontendName)
	meta.Bind(`backend_name`, &httpLog.BackendName)
	meta.Bind(`server_name`, &httpLog.ServerName)

	meta.Bind(`http_proto`, &httpLog.HTTPProto)
	meta.Bind(`http_method`, &httpLog.HTTPMethod)
	meta.Bind(`http_url`, &httpLog.HTTPURL)
	meta.Bind(`http_query`, &httpLog.HTTPQuery)

	meta.Bind(`header_request`, &httpLog.rawHeaderRequest)
	meta.Bind(`header_response`, &httpLog.rawHeaderResponse)

	meta.Bind(`cookie_request`, &httpLog.CookieRequest)
	meta.Bind(`cookie_response`, &httpLog.CookieResponse)
	meta.Bind(`termination_state`, &httpLog.TerminationState)

	meta.Bind(`bytes_read`, &httpLog.BytesRead)
	meta.Bind(`status_code`, &httpLog.StatusCode)
	meta.Bind(`client_port`, &httpLog.ClientPort)

	meta.Bind(`time_request`, &httpLog.TimeRequest)
	meta.Bind(`time_wait`, &httpLog.TimeWait)
	meta.Bind(`time_connect`, &httpLog.TimeConnect)
	meta.Bind(`time_response`, &httpLog.TimeResponse)
	meta.Bind(`time_all`, &httpLog.TimeAll)

	meta.Bind(`conn_active`, &httpLog.ConnActive)
	meta.Bind(`conn_frontend`, &httpLog.ConnFrontend)
	meta.Bind(`conn_backend`, &httpLog.ConnBackend)
	meta.Bind(`conn_server`, &httpLog.ConnServer)
	meta.Bind(`retries`, &httpLog.Retries)

	meta.Bind(`server_queue`, &httpLog.ServerQueue)
	meta.Bind(`backend_queue`, &httpLog.BackendQueue)

	return meta
}

func (httpLog *HTTPLog) parseQueue(in []byte) (ok bool) {
	httpLog.ServerQueue, ok = parseToInt32(in, '/')
	if !ok {
		return
	}

	httpLog.BackendQueue, ok = parseToInt32(in, ' ')

	return
}

// parserHeaderRequest parse the request header values in log file.
// The request headers start with '{' and end with '}'.
// Each header is separated by '|'.
func (httpLog *HTTPLog) parseHeaderRequest(in []byte, reqHeaders []string) (ok bool) {
	if in[0] != '{' {
		// Skip if we did not find the beginning.
		return true
	}

	end := bytes.IndexByte(in, '}')
	// Either '}' not found or its empty as in '{}'.
	if end <= 1 {
		return
	}

	httpLog.rawHeaderRequest = string(in[1:end])

	var headers = strings.Split(httpLog.rawHeaderRequest, `|`)

	if len(reqHeaders) != len(headers) {
		return
	}

	httpLog.HeaderRequest = make(map[string]string)
	for x, name := range reqHeaders {
		httpLog.HeaderRequest[name] = headers[x]
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
		httpLog.StatusCode,
		httpLog.TerminationState,
		httpLog.ClientIP,
		httpLog.ClientPort,
	)
	if err != nil {
		return err
	}

	for k, v = range httpLog.HeaderRequest {
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
		httpLog.TimeRequest, httpLog.TimeWait, httpLog.TimeConnect,
		httpLog.TimeResponse, httpLog.TimeAll,
		httpLog.ConnActive, httpLog.ConnFrontend, httpLog.ConnBackend,
		httpLog.ConnServer, httpLog.Retries,
		httpLog.ServerQueue, httpLog.BackendQueue,
		httpLog.BytesRead,
	)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(out, " %d\n", httpLog.RequestDate.UnixNano())
	if err != nil {
		return err
	}

	return nil
}
