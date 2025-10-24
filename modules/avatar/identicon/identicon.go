// Copyright 2015 caixw. All rights reserved.
// Copyright 2021 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// Dereved from https://github.com/issue9/identicon/
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

// IdenticonOptions is used to generate pseudo-random avatars
type IdenticonOptions struct {
	foreColors []color.Color
	backColor  color.Color
	size       int
	rect       image.Rectangle
}

// Identicon stores the generated identicon
type Identicon struct {
	Raster image.Paletted
	Vector string
}

// New returns an IdenticonOptions struct with the correct settings
// size image size
// back background color
// fore all possible foreground colors. only one foreground color will be picked randomly for one image
func New(size int, back color.Color, fore ...color.Color) (*IdenticonOptions, error) {
	if len(fore) == 0 {
		return nil, errors.New("foreground is not set")
	}

	if size < minImageSize {
		return nil, fmt.Errorf("size %d is smaller than min size %d", size, minImageSize)
	}

	return &IdenticonOptions{
		foreColors: fore,
		backColor:  back,
		size:       size,
		rect:       image.Rect(0, 0, size, size),
	}, nil
}

// Make generates an avatar by data
func (options *IdenticonOptions) Make(data []byte) *Identicon {
	h := sha256.New()
	h.Write(data)
	sum := h.Sum(nil)

	b1 := int(sum[0]+sum[1]+sum[2]) % len(blocks)
	b2 := int(sum[3]+sum[4]+sum[5]) % len(blocks)
	c := int(sum[6]+sum[7]+sum[8]) % len(centerBlocks)
	b1Angle := int(sum[9]+sum[10]) % 4
	b2Angle := int(sum[11]+sum[12]) % 4
	foreColor := int(sum[11]+sum[12]+sum[15]) % len(options.foreColors)

	return options.render(c, b1, b2, b1Angle, b2Angle, foreColor)
}

func (options *IdenticonOptions) render(c, b1, b2, b1Angle, b2Angle, foreColor int) *Identicon {
	raster := image.NewPaletted(options.rect, []color.Color{options.backColor, options.foreColors[foreColor]})
	vectorParts := [9]string{}

	drawBlocks(raster, &vectorParts, options.size, centerBlocks[c], blocks[b1], blocks[b2],
		svgCenterBlocks[c], svgBlocks[b1], svgBlocks[b2], b1Angle, b2Angle)
	vector := strings.Join(vectorParts[:], "")
	println("render: " + vector)

	return &Identicon{
		Raster: *raster,
		Vector: vector,
	}
}

/*
# Algorithm

Origin: An image is split into 9 areas

```
  -------------
  | 1 | 2 | 3 |
  -------------
  | 4 | 5 | 6 |
  -------------
  | 7 | 8 | 9 |
  -------------
```

Area 1/3/9/7 use a 90-degree rotating pattern.
Area 1/3/9/7 use another 90-degree rotating pattern.
Area 5 uses a random pattern.

The Patched Fix: make the image left-right mirrored to get rid of something like "swastika"
*/

// draw blocks to the paletted
// c: the block drawer for the center block
// b1,b2: the block drawers for other blocks (around the center block)
// b1Angle,b2Angle: the angle for the rotation of b1/b2
func drawBlocks(image *image.Paletted, vectorParts *[9]string, size int, c, b1, b2 blockFunc, svgC, svgShape1, svgShape2 svgBlockFunc, b1Angle, b2Angle int) {
	nextAngle := func(a int) int {
		return (a + 1) % 4
	}

	padding := (size % 3) / 2 // in cased the size can not be aligned by 3 blocks.

	blockSize := size / 3
	twoBlockSize := 2 * blockSize

	// center
	c(image, blockSize+padding, blockSize+padding, blockSize, 0)
	svgC(&vectorParts[0], blockSize+padding, blockSize+padding, blockSize)
	fmt.Println("ceter: " + vectorParts[0])

	// left top (1)
	b1(image, 0+padding, 0+padding, blockSize, b1Angle)
	svgShape1(&vectorParts[1], 0+padding, 0+padding, blockSize)
	// center top (2)
	b2(image, blockSize+padding, 0+padding, blockSize, b2Angle)
	svgShape2(&vectorParts[2], blockSize+padding, 0+padding, blockSize)

	b1Angle = nextAngle(b1Angle)
	b2Angle = nextAngle(b2Angle)
	// right top (3)
	// b1(p, twoBlockSize+padding, 0+padding, blockSize, b1Angle)
	// right middle (6)
	// b2(p, twoBlockSize+padding, blockSize+padding, blockSize, b2Angle)

	b1Angle = nextAngle(b1Angle)
	b2Angle = nextAngle(b2Angle)
	// right bottom (9)
	// b1(p, twoBlockSize+padding, twoBlockSize+padding, blockSize, b1Angle)
	// center bottom (8)
	b2(image, blockSize+padding, twoBlockSize+padding, blockSize, b2Angle)
	svgShape2(&vectorParts[8], blockSize+padding, twoBlockSize+padding, blockSize)

	b1Angle = nextAngle(b1Angle)
	b2Angle = nextAngle(b2Angle)
	// lef bottom (7)
	b1(image, 0+padding, twoBlockSize+padding, blockSize, b1Angle)
	svgShape1(&vectorParts[7], 0+padding, twoBlockSize+padding, blockSize)
	// left middle (4)
	b2(image, 0+padding, blockSize+padding, blockSize, b2Angle)
	svgShape2(&vectorParts[4], 0+padding, blockSize+padding, blockSize)

	// then we make it left-right mirror, so we didn't draw 3/6/9 before
	for x := 0; x < size/2; x++ {
		for y := 0; y < size; y++ {
			image.SetColorIndex(size-x, y, image.ColorIndexAt(x, y))
		}
	}
}
