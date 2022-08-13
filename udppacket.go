// Copyright 2018, M. Shulhan (ms@kilabit.info).  All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package haminer

const (
	defSize = 1024
)

// UDPPacket wrap the slice of bytes for easy manipulation.
type UDPPacket struct {
	Bytes []byte
}

// NewUDPPacket will create and initialize UDP packet.
func NewUDPPacket(size uint32) (p *UDPPacket) {
	if size <= 0 {
		size = defSize
	}
	p = &UDPPacket{
		Bytes: make([]byte, size),
	}

	return
}

// Reset will set the content of packet data to zero, so it can be used
// against on Read().
func (p *UDPPacket) Reset() {
	p.Bytes[0] = 0
	for x := 1; x < len(p.Bytes); x *= 2 {
		copy(p.Bytes[x:], p.Bytes[:x])
	}
}
