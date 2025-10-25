// Copyright 2015 caixw. All rights reserved.
// Copyright 2021 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// Rewrite of https://github.com/issue9/identicon/
// Generate pseudo-random avatars by IP, E-mail, etc.

package identicon

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"image"
	"image/color"
	"strings"
)

const minImageSize = 16

// Options is used to generate pseudo-random avatars
type Options struct {
	foreColors []uint32
	backColor  color.Color
	size       int
	rect       image.Rectangle
}

// Identicon stores the generated identicon
type Identicon struct {
	Raster image.Paletted
	Vector string
}

// New returns an Options struct with the correct settings
// size image size
// back background color
// fore all possible foreground colors. only one foreground color will be picked randomly for one image
func New(size int, back color.Color, fore ...uint32) (*Options, error) {
	if len(fore) == 0 {
		return nil, errors.New("foreground is not set")
	}

	if size < minImageSize {
		return nil, fmt.Errorf("size %d is smaller than min size %d", size, minImageSize)
	}

	return &Options{
		foreColors: fore,
		backColor:  back,
		size:       size,
		rect:       image.Rect(0, 0, size, size),
	}, nil
}

// Make generates an avatar by data
func (options *Options) Make(data []byte) *Identicon {
	h := sha256.New()
	h.Write(data)
	sum := h.Sum(nil)

	b1 := int(sum[0]+sum[1]+sum[2]) % len(allShapes)
	b2 := int(sum[3]+sum[4]+sum[5]) % len(allShapes)
	c := int(sum[6]+sum[7]+sum[8]) % len(middleShapes)
	tileOneAngle := int(sum[9]+sum[10]) % 4
	tileTwoAngle := int(sum[11]+sum[12]) % 4
	foreColorIndex := int(sum[11]+sum[12]+sum[15]) % len(options.foreColors)
	foreColor := options.foreColors[foreColorIndex]

	return options.render(c, b1, b2, tileOneAngle, tileTwoAngle, foreColor)
}

func (options *Options) render(c, b1, b2, tileOneAngle, tileTwoAngle int, foreColor uint32) *Identicon {
	raster := image.NewPaletted(options.rect, []color.Color{options.backColor, uint32RGBA(foreColor)})
	vectorParts := [9]string{}

	drawTiles(raster, &vectorParts, options.size, middleShapes[c], allShapes[b1], allShapes[b2], tileOneAngle, tileTwoAngle)
	vector := strings.Join(vectorParts[:], "")
	vector = `<g color="#` + uint32HEX(foreColor) + `">
	<g>` + vector + `</g>
	<g>` + vector + `</g>
</g>`

	return &Identicon{
		Raster: *raster,
		Vector: vector,
	}
}

// Draw both vector and raster tiles for the identicon
func drawTiles(image *image.Paletted, vectorParts *[9]string, size int, shapeMid, shapeOne, shapeTwo [][]int, tileOneAngle, tileTwoAngle int) {
	nextAngle := func(a int) int {
		return (a + 1) % 4
	}

	padding := (size % 3) / 2 // in case the size can not be aligned by 3 blocks.

	blockSize := size / 3
	twoBlockSize := 2 * blockSize

	svgSize := 36

	// Middle
	drawRasterTile(image, blockSize+padding, blockSize+padding, blockSize, 0, shapeMid)
	vectorParts[0] = renderVectorTile(1, 1, svgSize, 0, shapeMid)

	// Top left (1)
	drawRasterTile(image, 0+padding, 0+padding, blockSize, tileOneAngle, shapeOne)
	vectorParts[1] = renderVectorTile(0, 0, svgSize, tileOneAngle, shapeOne)
	// Top middle (2)
	drawRasterTile(image, blockSize+padding, 0+padding, blockSize, tileTwoAngle, shapeTwo)
	vectorParts[2] = renderVectorTile(1, 0, svgSize, tileTwoAngle, shapeTwo)

	tileOneAngle = nextAngle(tileOneAngle)
	tileTwoAngle = nextAngle(tileTwoAngle)

	tileOneAngle = nextAngle(tileOneAngle)
	tileTwoAngle = nextAngle(tileTwoAngle)

	// Bottom middle (8)
	drawRasterTile(image, blockSize+padding, twoBlockSize+padding, blockSize, tileTwoAngle, shapeTwo)
	vectorParts[8] = renderVectorTile(1, 2, svgSize, tileTwoAngle, shapeTwo)

	tileOneAngle = nextAngle(tileOneAngle)
	tileTwoAngle = nextAngle(tileTwoAngle)

	// Bottom left (7)
	drawRasterTile(image, 0+padding, twoBlockSize+padding, blockSize, tileOneAngle, shapeOne)
	vectorParts[7] = renderVectorTile(0, 2, svgSize, tileOneAngle, shapeOne)

	// Middle left (4)
	drawRasterTile(image, 0+padding, blockSize+padding, blockSize, tileTwoAngle, shapeTwo)
	vectorParts[4] = renderVectorTile(0, 1, svgSize, tileTwoAngle, shapeTwo)

	/*
		Original algorithm:
		-------------
		| 1 | 2 | 3 |
		-------------
		| 4 | 5 | 6 |
		-------------
		| 7 | 8 | 9 |
		-------------
	*/

	// Instead of drawing separate 3/6/9, mirror the image horizontally to avoid
	// rotational symmetry and identicons depicting swasticas
	for x := 0; x < size/2; x++ {
		for y := 0; y < size; y++ {
			image.SetColorIndex(size-x, y, image.ColorIndexAt(x, y))
		}
	}
}
