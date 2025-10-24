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

// diamond
//
//	---------
//	|   #   |
//	|  ###  |
//	| ##### |
//	|#######|
//	| ##### |
//	|  ###  |
//	|   #   |
//	---------
func b3(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2
	polygon := []int{
		m, 0,
		size, m,
		m, size,
		0, m,
	}
	drawBlock(img, x, y, size, 0, polygon)
	*vector = fmt.Sprintf(`<polygon points="%s"/>`, pointsToAttr(polygon))
	println("b3: " + *vector)
}

// b4
//
//	-------
//	|#####|
//	|#### |
//	|###  |
//	|##   |
//	|#    |
//	|------
func b4(img *image.Paletted, vector *string, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, []int{
		0, 0,
		size, 0,
		0, size,
	})
}

// b5
//
//	---------
//	|   #   |
//	|  ###  |
//	| ##### |
//	|#######|
func b5(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2
	drawBlock(img, x, y, size, angle, []int{
		m, 0,
		size, size,
		0, size,
	})
}

// b6
//
//	--------
//	|###   |
//	|###   |
//	|###   |
//	--------
func b6(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2
	drawBlock(img, x, y, size, angle, []int{
		0, 0,
		m, 0,
		m, size,
		0, size,
	})
}

// b7 italic cone
//
//	---------
//	| #     |
//	|  ##   |
//	|  #####|
//	|   ####|
//	|--------
func b7(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2
	drawBlock(img, x, y, size, angle, []int{
		0, 0,
		size, m,
		size, size,
		m, size,
	})
}

// b8 three small triangles
//
//	-----------
//	|    #    |
//	|   ###   |
//	|  #####  |
//	|  #   #  |
//	| ### ### |
//	|#########|
//	-----------
func b8(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2
	mm := m / 2

	// top
	drawBlock(img, x, y, size, angle, []int{
		m, 0,
		3 * mm, m,
		mm, m,
	})

	// bottom left
	drawBlock(img, x, y, size, angle, []int{
		mm, m,
		m, size,
		0, size,
	})

	// bottom right
	drawBlock(img, x, y, size, angle, []int{
		3 * mm, m,
		size, size,
		m, size,
	})
}

// b9 italic triangle
//
//	---------
//	|#      |
//	| ####  |
//	|  #####|
//	|  #### |
//	|   #   |
//	---------
func b9(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2
	drawBlock(img, x, y, size, angle, []int{
		0, 0,
		size, m,
		m, size,
	})
}

// b10
//
//	----------
//	|    ####|
//	|    ### |
//	|    ##  |
//	|    #   |
//	|####    |
//	|###     |
//	|##      |
//	|#       |
//	----------
func b10(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2
	drawBlock(img, x, y, size, angle, []int{
		m, 0,
		size, 0,
		m, m,
	})

	drawBlock(img, x, y, size, angle, []int{
		0, m,
		m, m,
		0, size,
	})
}

// b11
//
//	----------
//	|####    |
//	|####    |
//	|####    |
//	|        |
//	|        |
//	----------
func b11(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2
	drawBlock(img, x, y, size, angle, []int{
		0, 0,
		m, 0,
		m, m,
		0, m,
	})
}

// b12
//
//	-----------
//	|         |
//	|         |
//	|#########|
//	|  #####  |
//	|    #    |
//	-----------
func b12(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2
	drawBlock(img, x, y, size, angle, []int{
		0, m,
		size, m,
		m, size,
	})
}

// b13
//
//	-----------
//	|         |
//	|         |
//	|    #    |
//	|  #####  |
//	|#########|
//	-----------
func b13(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2
	drawBlock(img, x, y, size, angle, []int{
		m, m,
		size, size,
		0, size,
	})
}

// b14
//
//	---------
//	|   #   |
//	| ###   |
//	|####   |
//	|       |
//	|       |
//	---------
func b14(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2
	drawBlock(img, x, y, size, angle, []int{
		m, 0,
		m, m,
		0, m,
	})
}

// b15
//
//	----------
//	|#####   |
//	|###     |
//	|#       |
//	|        |
//	|        |
//	----------
func b15(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2
	drawBlock(img, x, y, size, angle, []int{
		0, 0,
		m, 0,
		0, m,
	})
}

// b16
//
//	---------
//	|   #   |
//	| ##### |
//	|#######|
//	|   #   |
//	| ##### |
//	|#######|
//	---------
func b16(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2
	drawBlock(img, x, y, size, angle, []int{
		m, 0,
		size, m,
		0, m,
	})

	drawBlock(img, x, y, size, angle, []int{
		m, m,
		size, size,
		0, size,
	})
}

// b17
//
//	----------
//	|#####   |
//	|###     |
//	|#       |
//	|      ##|
//	|      ##|
//	----------
func b17(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2

	drawBlock(img, x, y, size, angle, []int{
		0, 0,
		m, 0,
		0, m,
	})

	quarter := size / 4
	drawBlock(img, x, y, size, angle, []int{
		size - quarter, size - quarter,
		size, size - quarter,
		size, size,
		size - quarter, size,
	})
}

// b18
//
//	----------
//	|#####   |
//	|####    |
//	|###     |
//	|##      |
//	|#       |
//	----------
func b18(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2

	drawBlock(img, x, y, size, angle, []int{
		0, 0,
		m, 0,
		0, size,
	})
}

// b19
//
//	----------
//	|########|
//	|###  ###|
//	|#      #|
//	|###  ###|
//	|########|
//	----------
func b19(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2

	drawBlock(img, x, y, size, angle, []int{
		0, 0,
		m, 0,
		0, m,
	})

	drawBlock(img, x, y, size, angle, []int{
		m, 0,
		size, 0,
		size, m,
	})

	drawBlock(img, x, y, size, angle, []int{
		size, m,
		size, size,
		m, size,
	})

	drawBlock(img, x, y, size, angle, []int{
		0, m,
		m, size,
		0, size,
	})
}

// b20
//
//	----------
//	|  ##     |
//	|###      |
//	|##       |
//	|##       |
//	|#        |
//	----------
func b20(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2
	q := size / 4

	drawBlock(img, x, y, size, angle, []int{
		q, 0,
		0, size,
		0, m,
	})
}

// b21
//
//	----------
//	| ####   |
//	|## #####|
//	|##    ##|
//	|##      |
//	|#       |
//	----------
func b21(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2
	q := size / 4

	drawBlock(img, x, y, size, angle, []int{
		q, 0,
		0, size,
		0, m,
	})

	drawBlock(img, x, y, size, angle, []int{
		q, 0,
		size, q,
		size, m,
	})
}

// b22
//
//	----------
//	| ####   |
//	|##  ### |
//	|##    ##|
//	|##    ##|
//	|#      #|
//	----------
func b22(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2
	q := size / 4

	drawBlock(img, x, y, size, angle, []int{
		q, 0,
		0, size,
		0, m,
	})

	drawBlock(img, x, y, size, angle, []int{
		q, 0,
		size, q,
		size, size,
	})
}

// b23
//
//	----------
//	| #######|
//	|###    #|
//	|##      |
//	|##      |
//	|#       |
//	----------
func b23(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2
	q := size / 4

	drawBlock(img, x, y, size, angle, []int{
		q, 0,
		0, size,
		0, m,
	})

	drawBlock(img, x, y, size, angle, []int{
		q, 0,
		size, 0,
		size, q,
	})
}

// b24
//
//	----------
//	| ##  ###|
//	|###  ###|
//	|##  ##  |
//	|##  ##  |
//	|#   #   |
//	----------
func b24(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2
	q := size / 4

	drawBlock(img, x, y, size, angle, []int{
		q, 0,
		0, size,
		0, m,
	})

	drawBlock(img, x, y, size, angle, []int{
		m, 0,
		size, 0,
		m, size,
	})
}

// b25
//
//	----------
//	|#      #|
//	|##   ###|
//	|##  ##  |
//	|######  |
//	|####    |
//	----------
func b25(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2
	q := size / 4

	drawBlock(img, x, y, size, angle, []int{
		0, 0,
		0, size,
		q, size,
	})

	drawBlock(img, x, y, size, angle, []int{
		0, m,
		size, 0,
		q, size,
	})
}

// b26
//
//	----------
//	|#      #|
//	|###  ###|
//	|  ####  |
//	|###  ###|
//	|#      #|
//	----------
func b26(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2
	q := size / 4

	drawBlock(img, x, y, size, angle, []int{
		0, 0,
		m, q,
		q, m,
	})

	drawBlock(img, x, y, size, angle, []int{
		size, 0,
		m + q, m,
		m, q,
	})

	drawBlock(img, x, y, size, angle, []int{
		size, size,
		m, m + q,
		q + m, m,
	})

	drawBlock(img, x, y, size, angle, []int{
		0, size,
		q, m,
		m, q + m,
	})
}

// b27
//
//	----------
//	|########|
//	|##   ###|
//	|#      #|
//	|###   ##|
//	|########|
//	----------
func b27(img *image.Paletted, vector *string, x, y, size, angle int) {
	m := size / 2
	q := size / 4

	drawBlock(img, x, y, size, angle, []int{
		0, 0,
		size, 0,
		0, q,
	})

	drawBlock(img, x, y, size, angle, []int{
		q + m, 0,
		size, 0,
		size, size,
	})

	drawBlock(img, x, y, size, angle, []int{
		size, q + m,
		size, size,
		0, size,
	})

	drawBlock(img, x, y, size, angle, []int{
		0, size,
		0, 0,
		q, size,
	})
}
