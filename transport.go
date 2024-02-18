// Copyright (c) 2024 Gang Chen
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

const (
	profileMPEGTS    = 0x21 // MPEG TS
	profileMPEGVideo = 0x20 // MPEG video
	profileMPEGAudio = 0x0E // MPEG audio

	rtpHeaderSize = 12   // fixed RTP header size
	packetMTU     = 1500 // ethernet MTU
	maxPackets    = 16   // max in-flight packets
)

type Packet struct {
	data []byte
	seq  int // sequence
	off  int // offset
	len  int // length
}

type transport struct {
	conn *net.UDPConn // UDP connection
	pool sync.Pool    // packet pool for reuse
}

func (p *Packet) getByte(offset int) int {
	return int(p.data[offset]) & 0xFF
}

func (p *Packet) getUint16(offset int) uint16 {
	return uint16(p.data[offset+1]) | uint16(p.data[offset])<<8
}

func (p *Packet) Write(w io.Writer) error {
	for p.off < p.len {
		if n, err := w.Write(p.data[p.off:p.len]); err == nil {
			p.off += n
		} else {
			return err
		}
	}
	return nil
}

func (p *Packet) stripRtp() {
	ptype := p.getByte(1)       // payload type
	p.seq = int(p.getUint16(2)) // sequence number

	offset := rtpHeaderSize
	switch ptype {
	case profileMPEGTS:
	case profileMPEGVideo, profileMPEGAudio:
		offset += 4 // skip 4 bytes for MPEG video/audio
	}

	sign := p.getByte(0) // signature bits
	if sign&0x10 != 0 {  // extension available
		exlen := p.getUint16(rtpHeaderSize + 2)
		offset += 4 + int(exlen)*4
	}

	csrcCount := sign & 0x0F // CSRC count
	if csrcCount > 0 {
		log.Printf("csrc count: %d\n", csrcCount)
		offset += csrcCount * 4
	}

	p.off = offset
	if sign&0x20 != 0 { // padding
		p.len -= p.getByte(p.len - 1)
	}
}

func (p *Packet) nextSeq() int {
	return int(uint16(p.seq + 1))
}

// newTransport create a new transport with interface name and multicast address
func newTransport(ifname string, addr string) (*transport, error) {
	iface, err := net.InterfaceByName(ifname)
	if err != nil {
		return nil, err
	}

	gaddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return nil, err
	}

	if conn, err := net.ListenMulticastUDP("udp4", iface, gaddr); err == nil {
		return &transport{conn: conn}, nil
	} else {
		return nil, err
	}
}

// check packet if it's a RTP packet
func (p *Packet) check() (bool, error) {
	if p.len < rtpHeaderSize {
		return false, fmt.Errorf("invalid packet length: %d", p.len)
	}

	sign := p.getByte(0) // signature bits
	if sign == 0x47 {    // magic number for MPEG-TS
		log.Printf("MPEG TS stream detected\n")
		return false, nil
	}

	if ver := (sign & 0xC0) >> 6; ver != 2 { // only RTP version 2 are supported
		return false, fmt.Errorf("unsupported RTP version: %d", ver)
	}

	ptype := p.getByte(1) & 0x7F // payload type
	switch ptype {
	case profileMPEGTS, profileMPEGVideo, profileMPEGAudio:
		log.Printf("RTP stream detected: %x\n", ptype)
		return true, nil
	default:
		return false, fmt.Errorf("unknown payload profile: %x", ptype)
	}
}

// start processing and returns first packet
func (t *transport) start(ctx context.Context) (<-chan *Packet, error) {
	pkt, err := t.readPacket()
	if err != nil {
		return nil, err
	}

	rtp, err := pkt.check()
	if err != nil {
		return nil, err
	}

	pch := make(chan *Packet, maxPackets)
	if rtp {
		pkt.stripRtp()
	}
	pch <- pkt // add first packet

	if rtp {
		go t.transferRtp(ctx, pkt.nextSeq(), pch)
	} else {
		go t.transferRaw(ctx, pch)
	}
	return pch, nil
}

func (t *transport) readPacket() (*Packet, error) {
	var pk *Packet
	if p := t.pool.Get(); p == nil {
		pk = &Packet{data: make([]byte, packetMTU)}
	} else {
		pk = p.(*Packet)
	}

	if n, err := t.conn.Read(pk.data); err == nil {
		pk.len = n
		return pk, nil
	} else {
		log.Printf("Error occurs while reading: %s", err)
		t.pool.Put(pk)
		return nil, err
	}
}

func (t *transport) release(pkt *Packet) {
	pkt.off = 0
	pkt.len = 0
	t.pool.Put(pkt)
}

func (t *transport) transferRaw(ctx context.Context, ch chan<- *Packet) {
	defer close(ch)

	for {
		pkt, err := t.readPacket()
		if err != nil {
			break
		}

		select {
		case ch <- pkt:
		case <-ctx.Done(): // context canceled
			return
		}
	}
}

func (t *transport) transferRtp(ctx context.Context, nextSeq int, ch chan<- *Packet) {
	defer close(ch)

	enqueue := func(pkt *Packet) bool {
		select {
		case ch <- pkt:
			return true
		case <-ctx.Done(): // context canceled
			return false
		}
	}

	buf := make(map[int]*Packet)
	for {
		pkt, err := t.readPacket()
		if err != nil {
			break
		}

		pkt.stripRtp()
		seq := pkt.seq
		if seq != nextSeq {
			log.Printf("disorder packet received: %d", seq)
			if len(buf) < maxPackets {
				buf[seq] = pkt
				continue
			}

			// there are too many disorder packets
			log.Printf("too many disorder packets received: %d", len(buf))
			for seq != nextSeq {
				if pkt, ok := buf[nextSeq]; ok {
					if !enqueue(pkt) {
						return
					}
				}
				nextSeq = int(uint16(nextSeq + 1))
			}
			buf = make(map[int]*Packet) // reset map
		}

		nextSeq = pkt.nextSeq()
		if !enqueue(pkt) {
			return
		}

		for len(buf) > 0 {
			pkt, ok := buf[nextSeq]
			if !ok {
				break
			}

			delete(buf, nextSeq)
			if !enqueue(pkt) {
				return
			}
			nextSeq = pkt.nextSeq()
		}
	}
}

func (t *transport) Close() error {
	return t.conn.Close()
}
