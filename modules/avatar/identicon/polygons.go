// Copyright 2015 caixw. All rights reserved.
// Copyright 2021 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// Polygons for shapes in block.go

package identicon

// b3: diamond
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

func b3p1(size int) []int {
	m := size / 2
	return []int{
		m, 0,
		size, m,
		m, size,
		0, m,
	}
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
func block4poly1(size int) []int {
	return []int{
		0, 0,
		size, 0,
		0, size,
	}
}

// b5
//
//	---------
//	|   #   |
//	|  ###  |
//	| ##### |
//	|#######|
func block5poly1(size int) []int {
	m := size / 2
	return []int{
		m, 0,
		size, size,
		0, size,
	}
}

// b6
//
//	--------
//	|###   |
//	|###   |
//	|###   |
//	--------
func block6poly1(size int) []int {
	m := size / 2
	return []int{
		0, 0,
		m, 0,
		m, size,
		0, size,
	}
}

// b7 italic cone
//
//	---------
//	| #     |
//	|  ##   |
//	|  #####|
//	|   ####|
//	|--------
func block7poly1(size int) []int {
	m := size / 2
	return []int{
		0, 0,
		size, m,
		size, size,
		m, size,
	}
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
//
// Top
func block8poly1(size int) []int {
	m := size / 2
	mm := m / 2
	return []int{
		m, 0,
		3 * mm, m,
		mm, m,
	}
}

// Bottom left
func block8poly2(size int) []int {
	m := size / 2
	mm := m / 2
	return []int{
		mm, m,
		m, size,
		0, size,
	}
}

// Bottom right
func block8poly3(size int) []int {
	m := size / 2
	mm := m / 2
	return []int{
		3 * mm, m,
		size, size,
		m, size,
	}
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
func block9poly1(size int) []int {
	m := size / 2
	return []int{
		0, 0,
		size, m,
		m, size,
	}
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
func block10poly1(size int) []int {
	m := size / 2
	return []int{
		m, 0,
		size, 0,
		m, m,
	}
}
func block10poly2(size int) []int {
	m := size / 2
	return []int{
		0, m,
		m, m,
		0, size,
	}
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
func block11poly1(size int) []int {
	m := size / 2
	return []int{
		0, 0,
		m, 0,
		m, m,
		0, m,
	}
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
func block12poly1(size int) []int {
	m := size / 2
	return []int{
		0, m,
		size, m,
		m, size,
	}
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
func block13poly1(size int) []int {
	m := size / 2
	return []int{
		m, m,
		size, size,
		0, size,
	}
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
func blockb14poly1(size int) []int {
	m := size / 2
	return []int{
		m, 0,
		m, m,
		0, m,
	}
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
func blockb15poly1(size int) []int {
	m := size / 2
	return []int{
		0, 0,
		m, 0,
		0, m,
	}
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
func blockb16poly1(size int) []int {
	m := size / 2
	return []int{
		m, 0,
		size, m,
		0, m,
	}
}

func blockb16poly2(size int) []int {
	m := size / 2
	return []int{
		m, m,
		size, size,
		0, size,
	}
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
func blockb17poly1(size int) []int {
	m := size / 2
	return []int{
		0, 0,
		m, 0,
		0, m,
	}
}
func blockb17poly2(size int) []int {
	quarter := size / 4
	return []int{
		size - quarter, size - quarter,
		size, size - quarter,
		size, size,
		size - quarter, size,
	}
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
func blockb18poly1(size int) []int {
	m := size / 2
	return []int{
		0, 0,
		m, 0,
		0, size,
	}
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
func blockb19poly1(size int) []int {
	m := size / 2
	return []int{
		0, 0,
		m, 0,
		0, m,
	}
}
func blockb19poly2(size int) []int {
	m := size / 2
	return []int{
		m, 0,
		size, 0,
		size, m,
	}
}
func blockb19poly3(size int) []int {
	m := size / 2
	return []int{
		size, m,
		size, size,
		m, size,
	}
}
func blockb19poly4(size int) []int {
	m := size / 2
	return []int{
		0, m,
		m, size,
		0, size,
	}
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
func blockb20poly1(size int) []int {
	m := size / 2
	q := size / 4
	return []int{
		q, 0,
		0, size,
		0, m,
	}
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

// b22
//
//	----------
//	| ####   |
//	|##  ### |
//	|##    ##|
//	|##    ##|
//	|#      #|
//	----------

// b23
//
//	----------
//	| #######|
//	|###    #|
//	|##      |
//	|##      |
//	|#       |
//	----------

// b24
//
//	----------
//	| ##  ###|
//	|###  ###|
//	|##  ##  |
//	|##  ##  |
//	|#   #   |
//	----------

// b25
//
//	----------
//	|#      #|
//	|##   ###|
//	|##  ##  |
//	|######  |
//	|####    |
//	----------

// b26
//
//	----------
//	|#      #|
//	|###  ###|
//	|  ####  |
//	|###  ###|
//	|#      #|
//	----------

// b27
//
//	----------
//	|########|
//	|##   ###|
//	|#      #|
//	|###   ##|
//	|########|
//	----------
