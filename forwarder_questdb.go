// SPDX-FileCopyrightText: 2022 M. Shulhan <ms@kilabit.info>
// SPDX-License-Identifier: GPL-3.0-or-later

package haminer

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/url"
	"time"

	libnet "git.sr.ht/~shulhan/pakakeh.go/lib/net"
)

const (
	defQuestdbPort = 9009
)

// forwarderQuestdb client for questdb.
type forwarderQuestdb struct {
	conn net.Conn
	buf  bytes.Buffer
}

// newForwarderQuestdb create and initialize client connection using the URL in
// the ConfigForwarder.
func newForwarderQuestdb(cfg *ConfigForwarder) (questc *forwarderQuestdb, err error) {
	if cfg == nil || len(cfg.URL) == 0 {
		return nil, nil
	}

	var (
		logp    = `newForwarderQuestdb`
		timeout = 10 * time.Second

		surl    *url.URL
		address string
		ip      net.IP
		port    uint16
	)

	surl, err = url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf(`%s: %w`, logp, err)
	}

	if len(surl.Scheme) == 0 {
		surl.Scheme = "udp"
	}

	address, ip, port = libnet.ParseIPPort(surl.Host, defQuestdbPort)
	if len(address) == 0 {
		address = fmt.Sprintf(`%s:%d`, ip, port)
	} else {
		address = fmt.Sprintf(`%s:%d`, address, port)
	}

	questc = &forwarderQuestdb{}

	questc.conn, err = net.DialTimeout(surl.Scheme, address, timeout)
	if err != nil {
		return nil, fmt.Errorf(`%s: %w`, logp, err)
	}

	return questc, nil
}

// Forwards implement the Forwarder interface.
// It will write all logs to questdb.
func (questc *forwarderQuestdb) Forwards(logs []*HTTPLog) {
	var (
		logp = `forwarderQuestdb: Forwards`
		now  = time.Now()

		httpLog *HTTPLog
		data    []byte
		err     error
	)

	questc.buf.Reset()

	for _, httpLog = range logs {
		err = httpLog.writeIlp(&questc.buf)
		if err != nil {
			log.Printf(`%s: %s`, logp, err)
			return
		}
	}

	data = questc.buf.Bytes()

	err = questc.conn.SetWriteDeadline(now.Add(5 * time.Second))
	if err != nil {
		log.Printf(`%s: SetWriteDeadline: %s`, logp, err)
		return
	}

	_, err = questc.conn.Write(data)
	if err != nil {
		log.Printf(`%s: Write: %s`, logp, err)
	}
}
