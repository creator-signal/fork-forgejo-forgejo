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
	centerBlocks = []blockFunc{b0, b1, b2, b3, b19, b26, b27}

	// all blocks
	blocks = []blockFunc{b0, b1, b2, b3, b4, b5, b6, b7, b8, b9, b10, b11, b12, b13, b14, b15, b16, b17, b18, b19, b20, b21, b22, b23, b24, b25, b26, b27}
)

type blockFunc func(img *image.Paletted, vector *string, x, y, size, angle int)

// drawBlock draws a polygon by given points. The polygon can be rotated optionally
func drawBlock(img *image.Paletted, x, y, size, angle int, points []int) {
	// Points are stored as 1/4, 2/4, 3/4, 4/4 fractions
	for i := range points {
		points[i] = points[i] * size / 4
	}
	// The last point should be same as the first to end the shape
	points = append(points, points[0], points[1])

	if angle != 0 {
		m := size / 2
		rotate(points, m, m, angle)
	}

	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			if pointInPolygon(i, j, points) {
				img.SetColorIndex(x+i, y+j, 1)
			}
		}
	}
}

// pointsToAttr converts points into svg polygon points attribute
func pointsToAttr(pts []int) string {
	var b strings.Builder
	for i := 0; i+1 < len(pts); i += 2 {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%d,%d", pts[i], pts[i+1])
	}
	return b.String()
}

// blank
//
//	--------
//	|      |
//	|      |
//	|      |
//	--------
func b0(img *image.Paletted, vector *string, x, y, size, angle int) {}

// full-filled
//
//	--------
//	|######|
//	|######|
//	|######|
//	--------
func b1(img *image.Paletted, vector *string, x, y, size, angle int) {
	for i := x; i < x+size; i++ {
		for j := y; j < y+size; j++ {
			img.SetColorIndex(i, j, 1)
		}
	}
	*vector = ""
}

// a small block
//
//	----------
//	|        |
//	|  ####  |
//	|  ####  |
//	|        |
//	----------
func b2(img *image.Paletted, vector *string, x, y, size, angle int) {
	l := size / 4
	x += l
	y += l

	for i := x; i < x+2*l; i++ {
		for j := y; j < y+2*l; j++ {
			img.SetColorIndex(i, j, 1)
		}
	}
}

func b3(img *image.Paletted, vector *string, x, y, size, angle int) {
	polygon := b3p1()
	drawBlock(img, x, y, size, 0, polygon)
	*vector = fmt.Sprintf(`<polygon points="%s"/>`, pointsToAttr(polygon))
	println("b3: " + *vector)
}

func b4(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block4poly1())
}

func b5(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block5poly1())
}

func b6(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block6poly1())
}

func b7(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block7poly1())
}

func b8(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block8poly1())
	drawBlock(img, x, y, size, angle, block8poly2())
	drawBlock(img, x, y, size, angle, block8poly3())
}

func b9(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block9poly1())
}

func b10(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block10poly1())
	drawBlock(img, x, y, size, angle, block10poly2())
}

func b11(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block11poly1())
}

func b12(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block12poly1())
}

func b13(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, block13poly1())
}

func b14(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, blockb14poly1())
}

func b15(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, blockb15poly1())
}

func b16(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, blockb16poly1())
	drawBlock(img, x, y, size, angle, blockb16poly2())
}

func b17(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, blockb17poly1())
	drawBlock(img, x, y, size, angle, blockb17poly2())
}

func b18(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, blockb18poly1())
}

func b19(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, blockb19poly1())
	drawBlock(img, x, y, size, angle, blockb19poly2())
	drawBlock(img, x, y, size, angle, blockb19poly3())
	drawBlock(img, x, y, size, angle, blockb19poly4())
}

func b20(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, blockb20poly1())
}

func b21(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, blockb21poly1())
	drawBlock(img, x, y, size, angle, blockb21poly2())
}

func b22(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, blockb22poly1())
	drawBlock(img, x, y, size, angle, blockb22poly2())
}

func b23(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, blockb23poly1())
	drawBlock(img, x, y, size, angle, blockb23poly2())
}

func b24(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, blockb24poly1())
	drawBlock(img, x, y, size, angle, blockb24poly2())
}

func b25(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, blockb25poly1())
	drawBlock(img, x, y, size, angle, blockb25poly2())
}

func b26(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, blockb26poly1())
	drawBlock(img, x, y, size, angle, blockb26poly2())
	drawBlock(img, x, y, size, angle, blockb26poly3())
	drawBlock(img, x, y, size, angle, blockb26poly4())
}

func b27(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, blockb27poly1())
	drawBlock(img, x, y, size, angle, blockb27poly2())
	drawBlock(img, x, y, size, angle, blockb27poly3())
	drawBlock(img, x, y, size, angle, blockb27poly4())
}
