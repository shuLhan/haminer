// Copyright 2018, M. Shulhan (ms@kilabit.info).  All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package haminer

import (
	"bytes"
	"strconv"
	"strings"
	"time"
)

// heartbeat of haminer, in case we want to monitor deadman from haminer.
var heartbeat = &Halog{ // nolint: gochecknoglobals
	Timestamp:    time.Now(),
	FrontendName: "-",
	BackendName:  "-",
	ServerName:   "-",
	tagHTTPURL:   "-",
	HTTPMethod:   "-",
}

// Halog contains the mapping of haproxy HTTP log format to Go struct.
//
// Reference: https://cbonte.github.io/haproxy-dconv/1.7/configuration.html#8.2.3
type Halog struct { // nolint: maligned
	Timestamp time.Time

	ClientIP   string
	ClientPort int32

	FrontendName string
	BackendName  string
	ServerName   string

	TimeReq     int32
	TimeWait    int32
	TimeConnect int32
	TimeRsp     int32
	TimeAll     int32

	BytesRead int64

	CookieReq string
	CookieRsp string

	TermState string

	ConnActive   int32
	ConnFrontend int32
	ConnBackend  int32
	ConnServer   int32
	ConnRetries  int32

	QueueServer  int32
	QueueBackend int32

	RequestHeaders map[string]string

	HTTPStatus int32
	HTTPMethod string
	HTTPURL    string
	HTTPQuery  string
	HTTPProto  string

	tagHTTPURL string
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
	end := bytes.IndexByte(in, sep)
	if end < 0 {
		return 0, false
	}

	v, err := strconv.Atoi(string(in[:end]))
	if err != nil {
		return 0, false
	}

	copy(in, in[end+1:])

	return int32(v), true
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

func (halog *Halog) parseTimes(in []byte) (ok bool) {
	halog.TimeReq, ok = parseToInt32(in, '/')
	if !ok {
		return
	}

	halog.TimeWait, ok = parseToInt32(in, '/')
	if !ok {
		return
	}

	halog.TimeConnect, ok = parseToInt32(in, '/')
	if !ok {
		return
	}

	halog.TimeRsp, ok = parseToInt32(in, '/')
	if !ok {
		return
	}

	halog.TimeAll, ok = parseToInt32(in, ' ')
	if !ok {
		return
	}

	return
}

func (halog *Halog) parseConns(in []byte) (ok bool) {
	halog.ConnActive, ok = parseToInt32(in, '/')
	if !ok {
		return
	}

	halog.ConnFrontend, ok = parseToInt32(in, '/')
	if !ok {
		return
	}

	halog.ConnBackend, ok = parseToInt32(in, '/')
	if !ok {
		return
	}

	halog.ConnServer, ok = parseToInt32(in, '/')
	if !ok {
		return
	}

	halog.ConnRetries, ok = parseToInt32(in, ' ')
	if !ok {
		return
	}

	return
}

func (halog *Halog) parseQueue(in []byte) (ok bool) {
	halog.QueueServer, ok = parseToInt32(in, '/')
	if !ok {
		return
	}

	halog.QueueBackend, ok = parseToInt32(in, ' ')

	return
}

// parserRequestHeaders parse the request header values in log file.
// The request headers start with '{' and end with '}'.
// Each header is separated by '|'.
func (halog *Halog) parseRequestHeaders(in []byte, reqHeaders []string) (ok bool) {
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

	halog.RequestHeaders = make(map[string]string)
	for x, name := range reqHeaders {
		halog.RequestHeaders[name] = string(bheaders[x])
	}

	copy(in, in[end+2:])

	return true
}

func (halog *Halog) parseHTTP(in []byte) (ok bool) {
	halog.HTTPMethod, ok = parseToString(in, ' ')
	if !ok {
		return
	}

	v, ok := parseToString(in, ' ')
	if !ok {
		return
	}
	urlQuery := strings.SplitN(v, "?", 2)
	halog.HTTPURL = urlQuery[0]
	if len(urlQuery) == 2 {
		halog.HTTPQuery = urlQuery[1]
	}

	halog.HTTPProto, ok = parseToString(in, '"')

	return ok
}

// Parse will parse one line of HAProxy log format into Halog.
//
// nolint: gocyclo
func (halog *Halog) Parse(in []byte, reqHeaders []string) (ok bool) {
	var err error

	// Remove prefix from systemd/rsyslog
	ok = cleanPrefix(in)
	if !ok {
		return
	}

	// parse client IP
	halog.ClientIP, ok = parseToString(in, ':')
	if !ok {
		return
	}

	// parse client port
	halog.ClientPort, ok = parseToInt32(in, ' ')
	if !ok {
		return
	}

	// parse timestamp, remove '[' and parse until ']'
	in = in[1:]
	ts, ok := parseToString(in, ']')
	if !ok {
		return
	}

	halog.Timestamp, err = time.Parse("2/Jan/2006:15:04:05.000", ts)
	if err != nil {
		return false
	}

	// parse frontend name
	in = in[1:]
	halog.FrontendName, ok = parseToString(in, ' ')
	if !ok {
		return
	}

	// parse backend name
	halog.BackendName, ok = parseToString(in, '/')
	if !ok {
		return
	}

	// parse server name
	halog.ServerName, ok = parseToString(in, ' ')
	if !ok {
		return
	}

	// parse times
	ok = halog.parseTimes(in)
	if !ok {
		return
	}

	// parse HTTP status code
	halog.HTTPStatus, ok = parseToInt32(in, ' ')
	if !ok {
		return
	}

	// parse bytes read
	halog.BytesRead, ok = parseToInt64(in, ' ')
	if !ok {
		return
	}

	// parse request cookie
	halog.CookieReq, ok = parseToString(in, ' ')
	if !ok {
		return
	}

	// parse response cookie
	halog.CookieRsp, ok = parseToString(in, ' ')
	if !ok {
		return
	}

	// parse termination state
	halog.TermState, ok = parseToString(in, ' ')
	if !ok {
		return
	}

	// parse number of connections
	ok = halog.parseConns(in)
	if !ok {
		return
	}

	// parse number of queue state
	ok = halog.parseQueue(in)
	if !ok {
		return
	}

	if len(reqHeaders) > 0 {
		ok = halog.parseRequestHeaders(in, reqHeaders)
		if !ok {
			return
		}
	}

	// parse HTTP
	in = in[1:]
	ok = halog.parseHTTP(in)

	return ok
}

// ParseUDPPacket will convert UDP packet (in bytes) to instance of
// Halog.
//
// It will return nil and false if UDP packet is nil, have zero length, or
// cannot be parsed (rejected).
func (halog *Halog) ParseUDPPacket(p *UDPPacket, reqHeaders []string) bool {
	if p == nil {
		return false
	}
	if len(p.Bytes) == 0 {
		return false
	}

	var in []byte

	if p.Bytes[0] == '<' {
		endIdx := bytes.IndexByte(p.Bytes, '>')
		if endIdx < 0 {
			return false
		}

		in = make([]byte, len(p.Bytes))
		copy(in, p.Bytes[endIdx+1:])
	} else {
		in = p.Bytes
	}

	return halog.Parse(in, reqHeaders)
}
