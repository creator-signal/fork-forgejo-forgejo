// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package lfs

import (
	"bufio"
	"fmt"
	"io"
	"slices"
	"strconv"
	"sync"
	"time"
)

type PktLine []byte

const (
	MaxPacketLength = 65516
)

var (
	flushPkt = PktLine("0000")
	delimPkt = PktLine("0001") // See https://git-scm.com/docs/protocol-v2.html#_packet_line_framing
)

type PktAdapter struct {
	r *bufio.Reader
	w *bufio.Writer
}

type PktBinaryDataReader struct {
	r               io.Reader
	remainingBytes  int
	currentPktBytes int64
	eofReached      bool
	mu              sync.Mutex
}

func NewPktBinaryDataReader(r io.Reader, remainingBytes int) *PktBinaryDataReader {
	return &PktBinaryDataReader{r: r, remainingBytes: remainingBytes, currentPktBytes: 0, eofReached: false}
}

func (r *PktBinaryDataReader) IsDone() bool {
	return r.eofReached
}

func (r *PktBinaryDataReader) Read(d []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.eofReached {
		return 0, io.EOF
	}
	if r.currentPktBytes <= 0 {
		pktLen, err := readPacketLen(r.r)
		if err != nil {
			return 0, err
		}
		if pktLen == 0 {
			r.eofReached = true
			return 0, io.EOF
		}
		r.currentPktBytes = pktLen - 4
	}
	bytes, err := io.LimitReader(r.r, r.currentPktBytes).Read(d)
	r.currentPktBytes -= int64(bytes)
	r.remainingBytes -= bytes
	return bytes, err
}

func (r *PktBinaryDataReader) Close() error {
	buf := make([]byte, MaxPacketLength)
	for {
		if r.eofReached {
			return nil
		}
		_, err := r.Read(buf)
		if err != nil {
			return err
		}
	}
}

func readPacketLen(r io.Reader) (int64, error) {
	var pktLenHex [4]byte
	if byteRead, err := r.Read(pktLenHex[:]); err != nil {
		return 0, err
	} else if byteRead != 4 {
		return 0, fmt.Errorf("Cannot read pkt length: must received 4 bytes but only got %d", byteRead)
	}
	pktLen, err := strconv.ParseInt(string(pktLenHex[:]), 16, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to parse packet length: %q", pktLenHex)
	}
	return pktLen, nil
}

func readPacket(r io.Reader) ([]byte, int, error) {
	pktLen, err := readPacketLen(r)
	if err != nil {
		return nil, 0, err
	}
	if pktLen > MaxPacketLength {
		return nil, 0, fmt.Errorf("Failed to read packet length: length is to high %d > %d", pktLen, MaxPacketLength)
	}
	if pktLen <= 1 {
		return nil, int(pktLen), nil
	}
	if pktLen < 4 {
		return nil, int(pktLen), fmt.Errorf("Packet length is too small: %d must be >4", pktLen)
	}

	pkt, err := io.ReadAll(io.LimitReader(r, pktLen-4))
	if err != nil {
		return nil, 0, fmt.Errorf("Failed to read pkt: %s", err)
	}
	return pkt, int(pktLen), err
}

func NewPktAdapter(r io.Reader, w io.Writer) *PktAdapter {
	return &PktAdapter{r: bufio.NewReader(r), w: bufio.NewWriter(w)}
}

func (p *PktAdapter) GetBinaryReader(size int) *PktBinaryDataReader {
	return NewPktBinaryDataReader(p.r, size)
}

func (p *PktAdapter) Read() ([][]byte, error) {
	var list [][]byte
	for {
		pkt, pktLen, err := readPacket(p.r)
		if err != nil {
			return nil, err
		}

		if pktLen == 0 { // flush pkt
			break
		} else if pktLen == 1 { // delim pkt
			list = append(list, nil)
			break
		}
		list = append(list, pkt)
	}
	return list, nil
}

func NewPktLine(data []byte) (PktLine, error) {
	if len(data) > MaxPacketLength {
		return nil, fmt.Errorf("Packet length is too big (%d > %d)", len(data), MaxPacketLength)
	}
	return slices.Concat(fmt.Appendf(nil, "%04x", len(data)+4), data), nil
}

func (p *PktAdapter) NewStrPktLine(msg string) (PktLine, error) {
	return NewPktLine([]byte(msg + "\n"))
}

func (p *PktAdapter) createArgsData(args map[string]any) ([]byte, error) {
	var data []byte
	for key, val := range args {
		if valTime, ok := val.(time.Time); ok {
			val = valTime.UTC().Format(time.RFC3339)
		}
		pkt, err := NewPktLine(fmt.Appendf(nil, "%s=%v\n", key, val))
		if err != nil {
			return nil, err
		}
		data = append(data, []byte(pkt)...)
	}
	return data, nil
}

func (p *PktAdapter) WriteRaw(data []byte) error {
	_, err := p.w.Write(data)
	return err
}

func (p *PktAdapter) WriteData(data []byte) error {
	packet, err := NewPktLine(data)
	if err != nil {
		return err
	}
	return p.Write(packet)
}

func (p *PktAdapter) Write(packet PktLine) error {
	_, err := p.w.Write(packet)
	return err
}

func (p *PktAdapter) WriteDelim() error {
	_, err := p.w.Write(delimPkt)
	return err
}

func (p *PktAdapter) WriteFlush() error {
	if _, err := p.w.Write(flushPkt); err != nil {
		return fmt.Errorf("Error sending flushPkt: %v", err)
	}
	return p.w.Flush()
}

func (p *PktAdapter) WriteStr(msg string) error {
	packet, err := p.NewStrPktLine(msg)
	if err != nil {
		return err
	}
	return p.Write(packet)
}

func (p *PktAdapter) WriteHTTPOK() error {
	if err := p.WriteStr("status 200"); err != nil {
		return err
	}
	return p.WriteFlush()
}

func (p *PktAdapter) WriteStatusWithArgs(status int, args map[string]any) error {
	if err := p.WriteData(fmt.Appendf(nil, "status %d\n", status)); err != nil {
		return err
	}
	argData, err := p.createArgsData(args)
	if err != nil {
		return err
	}
	if err := p.WriteRaw(argData); err != nil {
		return err
	}
	return p.WriteFlush()
}

func (p *PktAdapter) WriteSplitPacket(packets ...PktLine) error {
	if len(packets) == 0 {
		return nil
	}

	if err := p.Write(packets[0]); err != nil {
		return err
	}
	if len(packets) > 1 {
		for _, pkt := range packets[1:] {
			if err := p.WriteDelim(); err != nil {
				return err
			}
			if err := p.Write(pkt); err != nil {
				return err
			}
		}
	}
	return p.WriteFlush()
}

func (p *PktAdapter) WriteHTTPErrorWithArgs(status int, msg string, args map[string]any) error {
	statusPkt, err := NewPktLine(fmt.Appendf(nil, "status %d\n", status))
	if err != nil {
		return err
	}
	if args != nil {
		argData, err := p.createArgsData(args)
		if err != nil {
			return err
		}
		statusPkt = append(statusPkt, argData...)
	}
	msgPkt, err := p.NewStrPktLine(msg)
	if err != nil {
		return err
	}
	return p.WriteSplitPacket(statusPkt, msgPkt)
}

func (p *PktAdapter) WriteHTTPError(status int, msg string) error {
	return p.WriteHTTPErrorWithArgs(status, msg, nil)
}

func (p *PktAdapter) WriteBinaryData(size int64, content io.Reader) error {
	if err := p.WriteStr("status 200"); err != nil {
		return fmt.Errorf("error writing OK status pkt-line: %v", err)
	}
	if err := p.WriteStr(fmt.Sprintf("size=%d", size)); err != nil {
		return fmt.Errorf("error writing arguments: %v", err)
	}
	if err := p.WriteDelim(); err != nil {
		return fmt.Errorf("error writing delim-pkt: %v", err)
	}
	remainingBytes := size
	if remainingBytes > MaxPacketLength {
		for remainingBytes >= MaxPacketLength-4 {
			if err := p.WriteRaw(fmt.Appendf(nil, "%04x", MaxPacketLength)); err != nil {
				return fmt.Errorf("error writing data size: %04d", MaxPacketLength)
			}
			written, err := io.CopyN(p.w, content, MaxPacketLength-4)
			remainingBytes -= written
			if err != nil {
				return fmt.Errorf("error whilst copying binary data after %d bytes. Error: %v", size-remainingBytes, err)
			}
		}
	}
	if remainingBytes > 0 {
		if err := p.WriteRaw(fmt.Appendf(nil, "%04x", remainingBytes+4)); err != nil {
			return fmt.Errorf("error writing data size: %04d", remainingBytes+4)
		}
		written, err := io.CopyN(p.w, content, remainingBytes)
		if err != nil {
			return fmt.Errorf("error whilst copying binary data after %d bytes. Error: %v", written, err)
		}
		if written != remainingBytes {
			return fmt.Errorf("error copying binary data: sent %d instead of %d", written, size)
		}
	}
	return p.WriteFlush()
}
