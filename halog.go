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

var (
	timestampLayout = "2/Jan/2006:15:04:05.000"
)

//
// Halog contains the mapping of haproxy HTTP log format to Go struct.
//
// Reference: https://cbonte.github.io/haproxy-dconv/1.7/configuration.html#8.2.3
//
type Halog struct {
	Timestamp    time.Time
	ClientIP     string
	ClientPort   int32
	FrontendName string
	BackendName  string
	ServerName   string
	TimeReq      int32
	TimeWait     int32
	TimeConnect  int32
	TimeRsp      int32
	TimeAll      int32
	HTTPStatus   int32
	BytesRead    int64
	CookieReq    string
	CookieRsp    string
	TermState    string
	ConnActive   int32
	ConnFrontend int32
	ConnBackend  int32
	ConnServer   int32
	ConnRetries  int32
	QueueServer  int32
	QueueBackend int32
	HTTPMethod   string
	HTTPURL      string
	HTTPQuery    string
	HTTPProto    string
}

//
// cleanPrefix will remove `<date-time> <process-name>[pid]: ` prefix (which
// come from systemd/rsyslog) in input.
//
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

	return
}

//
// Parse will parse one line of HAProxy log format into Halog.
//
// (1) Remove prefix from systemd/rsyslog
// (2) parse client IP
// (3) parse client port
// (4) parse timestamp, remove '[' and parse until ']'
// (5) parse frontend name
// (6) parse backend name
// (7) parse server name
// (8) parse times
// (9) parse HTTP status code
// (10) parse bytes read
// (11) parse request cookie
// (12) parse response cookie
// (13) parse termination state
// (14) parse number of connections
// (15) parse number of queue state
// (16) parse HTTP
//
// nolint: gocyclo
func (halog *Halog) Parse(in []byte) (ok bool) {
	var err error

	// (1)
	ok = cleanPrefix(in)
	if !ok {
		return
	}

	// (2)
	halog.ClientIP, ok = parseToString(in, ':')
	if !ok {
		return
	}

	// (3)
	halog.ClientPort, ok = parseToInt32(in, ' ')
	if !ok {
		return
	}

	// (4)
	in = in[1:]
	ts, ok := parseToString(in, ']')
	if !ok {
		return
	}

	halog.Timestamp, err = time.Parse(timestampLayout, ts)
	if err != nil {
		return false
	}

	// (5)
	in = in[1:]
	halog.FrontendName, ok = parseToString(in, ' ')
	if !ok {
		return
	}

	// (6)
	halog.BackendName, ok = parseToString(in, '/')
	if !ok {
		return
	}

	// (7)
	halog.ServerName, ok = parseToString(in, ' ')
	if !ok {
		return
	}

	// (8)
	ok = halog.parseTimes(in)
	if !ok {
		return
	}

	// (9)
	halog.HTTPStatus, ok = parseToInt32(in, ' ')
	if !ok {
		return
	}

	// (10)
	halog.BytesRead, ok = parseToInt64(in, ' ')
	if !ok {
		return
	}

	// (11)
	halog.CookieReq, ok = parseToString(in, ' ')
	if !ok {
		return
	}

	// (12)
	halog.CookieRsp, ok = parseToString(in, ' ')
	if !ok {
		return
	}

	// (13)
	halog.TermState, ok = parseToString(in, ' ')
	if !ok {
		return
	}

	// (14)
	ok = halog.parseConns(in)
	if !ok {
		return
	}

	// (15)
	ok = halog.parseQueue(in)
	if !ok {
		return
	}

	// (16)
	in = in[1:]
	ok = halog.parseHTTP(in)

	return
}

//
// ParseUDPPacket will convert UDP packet (in bytes) to instance of
// Halog.
//
// It will return nil and false if UDP packet is nil, have zero length, or
// cannot be parsed (rejected).
//
func (halog *Halog) ParseUDPPacket(p *UDPPacket) bool {
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

	return halog.Parse(in)
}
