// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package identicon

func svgB0(x, y, size, angle int) string {
	return ""
}

// b1 and b2 could be formatted as a rect but it will not be a measurable improvement

func svgB1(x, y, size, angle int) string {
	return formatPolygon(shape1[0], x, y, size, angle)
}

func svgB2(x, y, size, angle int) string {
	return formatPolygon(shape2[0], x, y, size, angle)
}

func svgB3(x, y, size, angle int) string {
	return formatPolygon(shape3[0], x, y, size, angle)
}

func svgB4(x, y, size, angle int) string {
	return formatPolygon(shape4[0], x, y, size, angle)
}

func svgB5(x, y, size, angle int) string {
	return formatPolygon(shape5[0], x, y, size, angle)
}

func svgB6(x, y, size, angle int) string {
	return formatPolygon(shape6[0], x, y, size, angle)
}

func svgB7(x, y, size, angle int) string {
	return formatPolygon(shape7[0], x, y, size, angle)
}

func svgB8(x, y, size, angle int) string {
	return formatPolygon(shape8[0], x, y, size, angle) + formatPolygon(shape8[1], x, y, size, angle) +
		formatPolygon(shape8[2], x, y, size, angle)
}

func svgB9(x, y, size, angle int) string {
	return formatPolygon(shape9[0], x, y, size, angle)
}

func svgB10(x, y, size, angle int) string {
	return formatPolygon(shape10[0], x, y, size, angle) + formatPolygon(shape10[1], x, y, size, angle)
}

func svgB11(x, y, size, angle int) string {
	return formatPolygon(shape11[0], x, y, size, angle)
}

func svgB12(x, y, size, angle int) string {
	return formatPolygon(shape12[0], x, y, size, angle)
}

func svgB13(x, y, size, angle int) string {
	return formatPolygon(shape13[0], x, y, size, angle)
}

func svgB14(x, y, size, angle int) string {
	return formatPolygon(shape14[0], x, y, size, angle)
}

func svgB15(x, y, size, angle int) string {
	return formatPolygon(shape15[0], x, y, size, angle)
}

func svgB16(x, y, size, angle int) string {
	return formatPolygon(shape16[0], x, y, size, angle) + formatPolygon(shape16[1], x, y, size, angle)
}

func svgB17(x, y, size, angle int) string {
	return formatPolygon(shape17[0], x, y, size, angle) + formatPolygon(shape17[1], x, y, size, angle)
}

func svgB18(x, y, size, angle int) string {
	return formatPolygon(shape18[0], x, y, size, angle)
}

func svgB19(x, y, size, angle int) string {
	return formatPolygon(shape19[0], x, y, size, angle) + formatPolygon(shape19[1], x, y, size, angle) +
		formatPolygon(shape19[2], x, y, size, angle) + formatPolygon(shape19[3], x, y, size, angle)
}

func svgB20(x, y, size, angle int) string {
	return formatPolygon(shape20[0], x, y, size, angle)
}

func svgB21(x, y, size, angle int) string {
	return formatPolygon(shape21[0], x, y, size, angle) + formatPolygon(shape21[1], x, y, size, angle)
}

func svgB22(x, y, size, angle int) string {
	return formatPolygon(shape22[0], x, y, size, angle) + formatPolygon(shape22[1], x, y, size, angle)
}

func svgB23(x, y, size, angle int) string {
	return formatPolygon(shape23[0], x, y, size, angle) + formatPolygon(shape23[1], x, y, size, angle)
}

func svgB24(x, y, size, angle int) string {
	return formatPolygon(shape24[0], x, y, size, angle) + formatPolygon(shape24[1], x, y, size, angle)
}

func svgB25(x, y, size, angle int) string {
	return formatPolygon(shape25[0], x, y, size, angle) + formatPolygon(shape25[1], x, y, size, angle)
}

func svgB26(x, y, size, angle int) string {
	return formatPolygon(shape26[0], x, y, size, angle) + formatPolygon(shape26[1], x, y, size, angle) +
		formatPolygon(shape26[2], x, y, size, angle) + formatPolygon(shape26[3], x, y, size, angle)
}

func svgB27(x, y, size, angle int) string {
	return formatPolygon(shape27[0], x, y, size, angle) + formatPolygon(shape27[1], x, y, size, angle) +
		formatPolygon(shape27[2], x, y, size, angle) + formatPolygon(shape27[3], x, y, size, angle)
}
