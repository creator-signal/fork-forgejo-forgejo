// Copyright 2015 caixw. All rights reserved.
// Copyright 2021 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// Derived from https://github.com/issue9/identicon/, with SVG additions

package identicon

import (
	"fmt"
	"image"
	"strings"
)

// drawRasterTile draws a shape for a tile. The shape consists of 0..4 polygons
func drawRasterTile(image *image.Paletted, x, y, size, angle int, shape [][]int) {
	for i := range shape {
		drawPolygon(image, x, y, size, angle, shape[i])
	}
}

// drawPolygon draws a polygon by given points. The polygon can be rotated optionally
func drawPolygon(img *image.Paletted, x, y, size, angle int, points []int) {
	// The last point should be same as the first to end the shape
	adjPoints := make([]int, len(points)+2)
	copy(adjPoints, points)
	adjPoints[len(points)] = points[0]
	adjPoints[len(points)+1] = points[1]
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

// renderVectorTile renders a shape for a tile into a string. The shape consists of 0..4 polygons
func renderVectorTile(x, y, angle int, shape [][]int) string {
	rendered := make([]string, len(shape))
	for i := range shape {
		rendered[i] = formatPolygon(shape[i], x, y, svgIdenticonSize, angle)
	}
	return strings.Join(rendered, "")
}

// formatPolygon converts points into a string usable as part of an SVG path
func formatPolygon(points []int, x, y, size, angle int) string {
	adjPoints := make([]int, len(points))
	copy(adjPoints, points)
	if angle != 0 {
		rotate(adjPoints, 2, 2, angle)
	}
	var b strings.Builder
	b.WriteByte('M')
	for i := 0; i+1 < len(adjPoints); i += 2 {
		if i > 0 {
			b.WriteByte('L')
		}
		fmt.Fprintf(&b, "%d %d", (size/3*x + size*adjPoints[i]/4/3), (size/3*y + size*adjPoints[i+1]/4/3))
	}
	b.WriteByte('Z')
	return b.String()
}
