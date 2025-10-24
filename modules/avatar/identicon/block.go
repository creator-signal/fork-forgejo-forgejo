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
	centerBlocks    = []rasterTileFunc{b0, b1, b2, b3, b19, b26, b27}
	svgCenterBlocks = []vectorTileFunc{svgB0, svgB1, svgB2, svgB3, svgB19, svgB26, svgB27}

	// all blocks
	blocks    = []rasterTileFunc{b0, b1, b2, b3, b4, b5, b6, b7, b8, b9, b10, b11, b12, b13, b14, b15, b16, b17, b18, b19, b20, b21, b22, b23, b24, b25, b26, b27}
	svgBlocks = []vectorTileFunc{svgB0, svgB1, svgB2, svgB3, svgB4, svgB5, svgB6, svgB7, svgB8, svgB9, svgB10, svgB11, svgB12, svgB13, svgB14, svgB15, svgB16, svgB17, svgB18, svgB19, svgB20, svgB21, svgB22, svgB23, svgB24, svgB25, svgB26, svgB27}
)

type (
	rasterTileFunc func(img *image.Paletted, x, y, size, angle int)
	vectorTileFunc func(x, y, size, angle int) string
)

// drawBlock draws a polygon by given points. The polygon can be rotated optionally
func drawBlock(img *image.Paletted, x, y, size, angle int, points []int) {
	// The last point should be same as the first to end the shape
	adjPoints := make([]int, 0, len(points)+2)
	adjPoints = append(adjPoints, points[0], points[1])
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
