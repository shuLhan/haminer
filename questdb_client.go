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

	libnet "github.com/shuLhan/share/lib/net"
)

const (
	defQuestdbPort = 9009
)

// questdbClient client for questdb.
type questdbClient struct {
	buf  bytes.Buffer
	conn net.Conn
}

// newQuestdbClient create and initialize client connection using the Url in
// the ConfigForwarder.
func newQuestdbClient(cfg *ConfigForwarder) (questc *questdbClient, err error) {
	if cfg == nil || len(cfg.Url) == 0 {
		return nil, nil
	}

	var (
		logp    = `newQuestdbClient`
		timeout = 10 * time.Second

		surl    *url.URL
		address string
		ip      net.IP
		port    uint16
	)

	surl, err = url.Parse(cfg.Url)
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

	questc = &questdbClient{}

	questc.conn, err = net.DialTimeout(surl.Scheme, address, timeout)
	if err != nil {
		return nil, fmt.Errorf(`%s: %w`, logp, err)
	}

	return questc, nil
}

// Forwards implement the Forwarder interface.
// It will write all logs to questdb.
func (questc *questdbClient) Forwards(logs []*HttpLog) {
	var (
		logp = `questdbClient: Forwards`
		now  = time.Now()

		httpLog *HttpLog
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
