// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package identicon

// b0 - empty tile
func svgB0(x, y, size, angle int) string {
	return ""
}

// b1 and b2 could be formatted as a rect but it will not be a measurable improvement

// b1 - full rectangle tile
func svgB1(x, y, size, angle int) string {
	return formatPolygon([]int{
		0, 0,
		4, 0,
		4, 4,
		0, 4,
	}, x, y, size, angle)
}

// b2 - small rectangle tile
func svgB2(x, y, size, angle int) string {
	return formatPolygon([]int{
		1, 1,
		3, 1,
		3, 3,
		1, 3,
	}, x, y, size, angle)
}

func svgB3(x, y, size, angle int) string {
	return formatPolygon(b3p1, x, y, size, angle)
}

func svgB4(x, y, size, angle int) string {
	return formatPolygon(block4poly1, x, y, size, angle)
}

func svgB5(x, y, size, angle int) string {
	return formatPolygon(block5poly1, x, y, size, angle)
}

func svgB6(x, y, size, angle int) string {
	return formatPolygon(block6poly1, x, y, size, angle)
}

func svgB7(x, y, size, angle int) string {
	return formatPolygon(block7poly1, x, y, size, angle)
}

func svgB8(x, y, size, angle int) string {
	return formatPolygon(block8poly1, x, y, size, angle) + formatPolygon(block8poly2, x, y, size, angle) +
		formatPolygon(block8poly3, x, y, size, angle)
}

func svgB9(x, y, size, angle int) string {
	return formatPolygon(block9poly1, x, y, size, angle)
}

func svgB10(x, y, size, angle int) string {
	return formatPolygon(block10poly1, x, y, size, angle) + formatPolygon(block10poly2, x, y, size, angle)
}

func svgB11(x, y, size, angle int) string {
	return formatPolygon(block11poly1, x, y, size, angle)
}

func svgB12(x, y, size, angle int) string {
	return formatPolygon(block12poly1, x, y, size, angle)
}

func svgB13(x, y, size, angle int) string {
	return formatPolygon(block13poly1, x, y, size, angle)
}

func svgB14(x, y, size, angle int) string {
	return formatPolygon(block14poly1, x, y, size, angle)
}

func svgB15(x, y, size, angle int) string {
	return formatPolygon(block15poly1, x, y, size, angle)
}

func svgB16(x, y, size, angle int) string {
	return formatPolygon(block16poly1, x, y, size, angle) + formatPolygon(block16poly2, x, y, size, angle)
}

func svgB17(x, y, size, angle int) string {
	return formatPolygon(block17poly1, x, y, size, angle) + formatPolygon(block17poly2, x, y, size, angle)
}

func svgB18(x, y, size, angle int) string {
	return formatPolygon(block18poly1, x, y, size, angle)
}

func svgB19(x, y, size, angle int) string {
	return formatPolygon(block19poly1, x, y, size, angle) + formatPolygon(block19poly2, x, y, size, angle) +
		formatPolygon(block19poly3, x, y, size, angle) + formatPolygon(block19poly4, x, y, size, angle)
}

func svgB20(x, y, size, angle int) string {
	return formatPolygon(block20poly1, x, y, size, angle)
}

func svgB21(x, y, size, angle int) string {
	return formatPolygon(block21poly1, x, y, size, angle) + formatPolygon(block21poly2, x, y, size, angle)
}

func svgB22(x, y, size, angle int) string {
	return formatPolygon(block22poly1, x, y, size, angle) + formatPolygon(block22poly2, x, y, size, angle)
}

func svgB23(x, y, size, angle int) string {
	return formatPolygon(block23poly1, x, y, size, angle) + formatPolygon(block23poly2, x, y, size, angle)
}

func svgB24(x, y, size, angle int) string {
	return formatPolygon(block24poly1, x, y, size, angle) + formatPolygon(block24poly2, x, y, size, angle)
}

func svgB25(x, y, size, angle int) string {
	return formatPolygon(block25poly1, x, y, size, angle) + formatPolygon(block25poly2, x, y, size, angle)
}

func svgB26(x, y, size, angle int) string {
	return formatPolygon(block26poly1, x, y, size, angle) + formatPolygon(block26poly2, x, y, size, angle) +
		formatPolygon(block26poly3, x, y, size, angle) + formatPolygon(block26poly4, x, y, size, angle)
}

func svgB27(x, y, size, angle int) string {
	return formatPolygon(block27poly1, x, y, size, angle) + formatPolygon(block27poly2, x, y, size, angle) +
		formatPolygon(block27poly3, x, y, size, angle) + formatPolygon(block27poly4, x, y, size, angle)
}
