// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package lfs

import (
	"bytes"
	"testing"
)

func TestPktAdapter_WriteHTTPError(t *testing.T) {
	tests := []struct {
		name     string // description of test
		msg      string
		expected []byte
	}{
		{"Error message", "Not implemented", []byte("000fstatus 400\n00010014Not implemented\n0000")},
		{"Error message", "size mismatch", []byte("000fstatus 400\n00010012size mismatch\n0000")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var in bytes.Buffer
			var out bytes.Buffer
			p := NewPktAdapter(&in, &out)
			gotErr := p.WriteHTTPError(400, tt.msg)
			if gotErr != nil {
				t.Errorf("WriteHttpError() failed: %v", gotErr)
			}
			if !bytes.Equal(out.Bytes(), tt.expected) {
				t.Errorf("WriteHttpError() output should be %q but got %q", tt.expected, out.Bytes())
			}
		})
	}
}

func TestPktAdapter_WriteHTTPOK(t *testing.T) {
	expected := []byte("000fstatus 200\n0000")
	var in bytes.Buffer
	var out bytes.Buffer
	p := NewPktAdapter(&in, &out)
	gotErr := p.WriteHTTPOK()
	if gotErr != nil {
		t.Errorf("WriteHttpOK() failed: %v", gotErr)
	}
	if !bytes.Equal(out.Bytes(), expected) {
		t.Errorf("WriteHttpError() output should be %q but got %q", expected, out.Bytes())
	}
}

func TestPktAdapter_Write2SplitPacket(t *testing.T) {
	expected := []byte("000fstatus 400\n00010020Unexpected version received\n0000")
	var in bytes.Buffer
	var out bytes.Buffer
	p := NewPktAdapter(&in, &out)
	pkt1, err := p.NewStrPktLine("status 400")
	if err != nil {
		t.Errorf("NewStrPktLine() failed: %v", err)
	}
	pkt2, err := p.NewStrPktLine("Unexpected version received")
	if err != nil {
		t.Errorf("NewStrPktLine() failed: %v", err)
	}
	gotErr := p.WriteSplitPacket(pkt1, pkt2)
	if gotErr != nil {
		t.Errorf("WriteHttpOK() failed: %v", gotErr)
	}
	if !bytes.Equal(out.Bytes(), expected) {
		t.Errorf("WriteHttpError() output should be %q but got %q", expected, out.Bytes())
	}
}
