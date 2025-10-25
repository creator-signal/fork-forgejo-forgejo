// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package identicon

import (
	"fmt"
	"image/color"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUint32RGBA(t *testing.T) {
	tests := []struct {
		in   uint32
		want color.RGBA
	}{
		{in: 0xffffffff, want: color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}},
		{in: 0x00000000, want: color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x00}},
		{in: 0x112233ff, want: color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff}},
		{in: 0x01020304, want: color.RGBA{R: 0x01, G: 0x02, B: 0x03, A: 0x04}},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("0x%08x", tt.in), func(t *testing.T) {
			gotColor := uint32RGBA(tt.in)
			gotR, gotG, gotB, gotA := gotColor.RGBA()
			require.Equal(t, tt.want, color.RGBA{R: uint8(gotR), G: uint8(gotG), B: uint8(gotB), A: uint8(gotA)})
		})
	}
}

func TestUint32HEX(t *testing.T) {
	tests := []struct {
		in   uint32
		want string
	}{
		{in: 0xffffffff, want: "ffffffff"},
		{in: 0x00000000, want: "00000000"},
		{in: 0x112233ff, want: "112233ff"},
		{in: 0x01020304, want: "01020304"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("0x%08x", tt.in), func(t *testing.T) {
			require.Equal(t, tt.want, uint32HEX(tt.in))
		})
	}
}
