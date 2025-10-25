// Copyright 2015 caixw. All rights reserved.
// Copyright 2021 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// Derived from https://github.com/issue9/identicon/

package identicon

import "image"

func b0(img *image.Paletted, x, y, size, angle int) {}

func b1(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, 0, shape1[0])
}

func b2(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, 0, shape2[0])
}

func b3(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, 0, shape3[0])
}

func b4(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape4[0])
}

func b5(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape5[0])
}

func b6(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape6[0])
}

func b7(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape7[0])
}

func b8(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape8[0])
	drawBlock(img, x, y, size, angle, shape8[1])
	drawBlock(img, x, y, size, angle, shape8[2])
}

func b9(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape9[0])
}

func b10(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape10[0])
	drawBlock(img, x, y, size, angle, shape10[1])
}

func b11(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape11[0])
}

func b12(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape12[0])
}

func b13(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape13[0])
}

func b14(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape14[0])
}

func b15(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape15[0])
}

func b16(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape16[0])
	drawBlock(img, x, y, size, angle, shape16[1])
}

func b17(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape17[0])
	drawBlock(img, x, y, size, angle, shape17[1])
}

func b18(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape18[0])
}

func b19(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape19[0])
	drawBlock(img, x, y, size, angle, shape19[1])
	drawBlock(img, x, y, size, angle, shape19[2])
	drawBlock(img, x, y, size, angle, shape19[3])
}

func b20(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape20[0])
}

func b21(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape21[0])
	drawBlock(img, x, y, size, angle, shape21[1])
}

func b22(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape22[0])
	drawBlock(img, x, y, size, angle, shape22[1])
}

func b23(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape23[0])
	drawBlock(img, x, y, size, angle, shape23[1])
}

func b24(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape24[0])
	drawBlock(img, x, y, size, angle, shape24[1])
}

func b25(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape25[0])
	drawBlock(img, x, y, size, angle, shape25[1])
}

func b26(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape26[0])
	drawBlock(img, x, y, size, angle, shape26[1])
	drawBlock(img, x, y, size, angle, shape26[2])
	drawBlock(img, x, y, size, angle, shape26[3])
}

func b27(img *image.Paletted, x, y, size, angle int) {
	drawBlock(img, x, y, size, angle, shape27[0])
	drawBlock(img, x, y, size, angle, shape27[1])
	drawBlock(img, x, y, size, angle, shape27[2])
	drawBlock(img, x, y, size, angle, shape27[3])
}
