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

	return &Identicon{
		Raster: options.render(c, b1, b2, b1Angle, b2Angle, foreColor),
		Vector: `<path d="M1 7.775V2.75C1 1.784 1.784 1 2.75 1h5.025c.464 0 .91.184 1.238.513l6.25 6.25a1.75 1.75 0 0 1 0 2.474l-5.026 5.026a1.75 1.75 0 0 1-2.474 0l-6.25-6.25A1.75 1.75 0 0 1 1 7.775m1.5 0c0 .066.026.13.073.177l6.25 6.25a.25.25 0 0 0 .354 0l5.025-5.025a.25.25 0 0 0 0-.354l-6.25-6.25a.25.25 0 0 0-.177-.073H2.75a.25.25 0 0 0-.25.25ZM6 5a1 1 0 1 1 0 2 1 1 0 0 1 0-2"/>`,
	}
}

func (options *IdenticonOptions) render(c, b1, b2, b1Angle, b2Angle, foreColor int) image.Paletted {
	image := image.NewPaletted(options.rect, []color.Color{options.backColor, options.foreColors[foreColor]})
	drawBlocks(image, options.size, centerBlocks[c], blocks[b1], blocks[b2], b1Angle, b2Angle)
	return *image
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
func drawBlocks(p *image.Paletted, size int, c, b1, b2 blockFunc, b1Angle, b2Angle int) {
	nextAngle := func(a int) int {
		return (a + 1) % 4
	}

	padding := (size % 3) / 2 // in cased the size can not be aligned by 3 blocks.

	blockSize := size / 3
	twoBlockSize := 2 * blockSize

	// center
	c(p, blockSize+padding, blockSize+padding, blockSize, 0)

	// left top (1)
	b1(p, 0+padding, 0+padding, blockSize, b1Angle)
	// center top (2)
	b2(p, blockSize+padding, 0+padding, blockSize, b2Angle)

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
	b2(p, blockSize+padding, twoBlockSize+padding, blockSize, b2Angle)

	b1Angle = nextAngle(b1Angle)
	b2Angle = nextAngle(b2Angle)
	// lef bottom (7)
	b1(p, 0+padding, twoBlockSize+padding, blockSize, b1Angle)
	// left middle (4)
	b2(p, 0+padding, blockSize+padding, blockSize, b2Angle)

	// then we make it left-right mirror, so we didn't draw 3/6/9 before
	for x := 0; x < size/2; x++ {
		for y := 0; y < size; y++ {
			p.SetColorIndex(size-x, y, p.ColorIndexAt(x, y))
		}
	}
}
