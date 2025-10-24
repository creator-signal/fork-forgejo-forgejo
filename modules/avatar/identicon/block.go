// Copyright 2015 caixw. All rights reserved.
// Copyright 2021 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// Derived from https://github.com/issue9/identicon/

package identicon

import (
	"fmt"
	"image"
	"strings"
)

var (
	// the blocks can appear in center, these blocks can be more beautiful
	centerBlocks    = []blockFunc{b0, b1, b2, b3, b19, b26, b27}
	svgCenterBlocks = []svgBlockFunc{svgB0, svgB1, svgB2, svgB3, svgB19, svgB26, svgB27}

	// all blocks
	blocks    = []blockFunc{b0, b1, b2, b3, b4, b5, b6, b7, b8, b9, b10, b11, b12, b13, b14, b15, b16, b17, b18, b19, b20, b21, b22, b23, b24, b25, b26, b27}
	svgBlocks = []svgBlockFunc{svgB0, svgB1, svgB2, svgB3, svgB4, svgB5, svgB6, svgB7, svgB8, svgB9, svgB10, svgB11, svgB12, svgB13, svgB14, svgB15, svgB16, svgB17, svgB18, svgB19, svgB20, svgB21, svgB22, svgB23, svgB24, svgB25, svgB26, svgB27}
)

type blockFunc func(img *image.Paletted, x, y, size, angle int)
type svgBlockFunc func(x, y, size, angle int) string

// drawBlock draws a polygon by given points. The polygon can be rotated optionally
func drawBlock(img *image.Paletted, x, y, size, angle int, points []int) {
	// The last point should be same as the first to end the shape
	adjPoints := append(points, points[0], points[1])
	// Points are stored as 1/4, 2/4, 3/4, 4/4 fractions
	for i := range adjPoints {
		adjPoints[i] = adjPoints[i] * size / 4
	}

	if angle != 0 {
		m := size / 2
		rotate(adjPoints, m, m, angle)
	}

	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			if pointInPolygon(i, j, adjPoints) {
				img.SetColorIndex(x+i, y+j, 1)
			}
		}
	}
}

// formatPolygon converts points into svg polygon
func formatPolygon(points []int, x, y, size, angle int) string {
	adjPoints := make([]int, len(points))
	copy(adjPoints, points)
	if angle != 0 {
		sixteenth := size / 16
		rotate(adjPoints, sixteenth, sixteenth, angle)
	}
	var b strings.Builder
	for i := 0; i+1 < len(adjPoints); i += 2 {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%d,%d", (size/3*x + size*adjPoints[i]/4/3), (size/3*y + size*adjPoints[i+1]/4/3))
	}
	return fmt.Sprintf(`<polygon points="%s"/>`, b.String())
}

// blank
//
//	--------
//	|      |
//	|      |
//	|      |
//	--------
func b0(img *image.Paletted, x, y, size, angle int) {}

// full-filled
//
//	--------
//	|######|
//	|######|
//	|######|
//	--------
func b1(img *image.Paletted, x, y, size, angle int) {
	for i := x; i < x+size; i++ {
		for j := y; j < y+size; j++ {
			img.SetColorIndex(i, j, 1)
		}
	}
}

// a small block
//
//	----------
//	|        |
//	|  ####  |
//	|  ####  |
//	|        |
//	----------
func b2(img *image.Paletted, x, y, size, angle int) {
	l := size / 4
	x += l
	y += l

	for i := x; i < x+2*l; i++ {
		for j := y; j < y+2*l; j++ {
			img.SetColorIndex(i, j, 1)
		}
	}
}

func b3(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, 0, b3p1)
}

func b4(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block4poly1)
}

func b5(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block5poly1)
}

func b6(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block6poly1)
}

func b7(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block7poly1)
}

func b8(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block8poly1)
	drawBlock(img, x, y, size, angle, block8poly2)
	drawBlock(img, x, y, size, angle, block8poly3)
}

func b9(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block9poly1)
}

func b10(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block10poly1)
	drawBlock(img, x, y, size, angle, block10poly2)
}

func b11(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block11poly1)
}

func b12(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block12poly1)
}

func b13(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block13poly1)
}

func b14(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block14poly1)
}

func b15(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block15poly1)
}

func b16(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block16poly1)
	drawBlock(img, x, y, size, angle, block16poly2)
}

func b17(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block17poly1)
	drawBlock(img, x, y, size, angle, block17poly2)
}

func b18(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block18poly1)
}

func b19(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block19poly1)
	drawBlock(img, x, y, size, angle, block19poly2)
	drawBlock(img, x, y, size, angle, block19poly3)
	drawBlock(img, x, y, size, angle, block19poly4)
}

func b20(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block20poly1)
}

func b21(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block21poly1)
	drawBlock(img, x, y, size, angle, block21poly2)
}

func b22(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block22poly1)
	drawBlock(img, x, y, size, angle, block22poly2)
}

func b23(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block23poly1)
	drawBlock(img, x, y, size, angle, block23poly2)
}

func b24(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block24poly1)
	drawBlock(img, x, y, size, angle, block24poly2)
}

func b25(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block25poly1)
	drawBlock(img, x, y, size, angle, block25poly2)
}

func b26(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block26poly1)
	drawBlock(img, x, y, size, angle, block26poly2)
	drawBlock(img, x, y, size, angle, block26poly3)
	drawBlock(img, x, y, size, angle, block26poly4)
}

func b27(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block27poly1)
	drawBlock(img, x, y, size, angle, block27poly2)
	drawBlock(img, x, y, size, angle, block27poly3)
	drawBlock(img, x, y, size, angle, block27poly4)
}
