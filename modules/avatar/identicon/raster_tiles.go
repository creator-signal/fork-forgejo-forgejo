// Copyright 2015 caixw. All rights reserved.
// Copyright 2021 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// Derived from https://github.com/issue9/identicon/

package identicon

import "image"

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
