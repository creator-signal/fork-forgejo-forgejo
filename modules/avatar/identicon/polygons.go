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

func b3p1() []int {
	return []int{
		2, 0,
		4, 2,
		2, 4,
		0, 2,
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
func block4poly1() []int {
	return []int{
		0, 0,
		4, 0,
		0, 4,
	}
}

// b5
//
//	---------
//	|   #   |
//	|  ###  |
//	| ##### |
//	|#######|
func block5poly1() []int {
	return []int{
		2, 0,
		4, 4,
		0, 4,
	}
}

// b6
//
//	--------
//	|###   |
//	|###   |
//	|###   |
//	--------
func block6poly1() []int {
	return []int{
		0, 0,
		2, 0,
		2, 4,
		0, 4,
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
func block7poly1() []int {
	return []int{
		0, 0,
		4, 2,
		4, 4,
		2, 4,
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
func block8poly1() []int {
	return []int{
		2, 0,
		3, 2,
		1, 2,
	}
}

// Bottom left
func block8poly2() []int {
	return []int{
		1, 2,
		2, 4,
		0, 4,
	}
}

// Bottom right
func block8poly3() []int {
	return []int{
		3, 2,
		4, 4,
		2, 4,
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
func block9poly1() []int {

	return []int{
		0, 0,
		4, 2,
		2, 4,
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
func block10poly1() []int {

	return []int{
		2, 0,
		4, 0,
		2, 2,
	}
}
func block10poly2() []int {

	return []int{
		0, 2,
		2, 2,
		0, 4,
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
func block11poly1() []int {

	return []int{
		0, 0,
		2, 0,
		2, 2,
		0, 2,
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
func block12poly1() []int {

	return []int{
		0, 2,
		4, 2,
		2, 4,
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
func block13poly1() []int {

	return []int{
		2, 2,
		4, 4,
		0, 4,
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
func blockb14poly1() []int {

	return []int{
		2, 0,
		2, 2,
		0, 2,
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
func blockb15poly1() []int {

	return []int{
		0, 0,
		2, 0,
		0, 2,
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
func blockb16poly1() []int {
	return []int{
		2, 0,
		4, 2,
		0, 2,
	}
}

func blockb16poly2() []int {
	return []int{
		2, 2,
		4, 4,
		0, 4,
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
func blockb17poly1() []int {
	return []int{
		0, 0,
		2, 0,
		0, 2,
	}
}
func blockb17poly2() []int {
	return []int{
		3, 3,
		4, 3,
		4, 4,
		3, 4,
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
func blockb18poly1() []int {

	return []int{
		0, 0,
		2, 0,
		0, 4,
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
func blockb19poly1() []int {

	return []int{
		0, 0,
		2, 0,
		0, 2,
	}
}
func blockb19poly2() []int {

	return []int{
		2, 0,
		4, 0,
		4, 2,
	}
}
func blockb19poly3() []int {

	return []int{
		4, 2,
		4, 4,
		2, 4,
	}
}
func blockb19poly4() []int {

	return []int{
		0, 2,
		2, 4,
		0, 4,
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
func blockb20poly1() []int {

	return []int{
		1, 0,
		0, 4,
		0, 2,
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
func blockb21poly1() []int {

	return []int{
		1, 0,
		0, 4,
		0, 2,
	}
}

func blockb21poly2() []int {

	return []int{
		1, 0,
		4, 1,
		4, 2,
	}
}

// b22
//
//	----------
//	| ####   |
//	|##  ### |
//	|##    ##|
//	|##    ##|
//	|#      #|
//	----------
func blockb22poly1() []int {

	return []int{
		1, 0,
		0, 4,
		0, 2,
	}
}

func blockb22poly2() []int {

	return []int{
		1, 0,
		4, 1,
		4, 4,
	}
}

// b23
//
//	----------
//	| #######|
//	|###    #|
//	|##      |
//	|##      |
//	|#       |
//	----------
func blockb23poly1() []int {

	return []int{
		1, 0,
		0, 4,
		0, 2,
	}
}

func blockb23poly2() []int {

	return []int{
		1, 0,
		4, 0,
		4, 1,
	}
}

// b24
//
//	----------
//	| ##  ###|
//	|###  ###|
//	|##  ##  |
//	|##  ##  |
//	|#   #   |
//	----------
func blockb24poly1() []int {

	return []int{
		1, 0,
		0, 4,
		0, 2,
	}
}

func blockb24poly2() []int {

	return []int{
		2, 0,
		4, 0,
		2, 4,
	}
}

// b25
//
//	----------
//	|#      #|
//	|##   ###|
//	|##  ##  |
//	|######  |
//	|####    |
//	----------
func blockb25poly1() []int {
	return []int{
		0, 0,
		0, 4,
		1, 4,
	}
}

func blockb25poly2() []int {
	return []int{
		0, 2,
		4, 0,
		1, 4,
	}
}

// b26
//
//	----------
//	|#      #|
//	|###  ###|
//	|  ####  |
//	|###  ###|
//	|#      #|
//	----------
func blockb26poly1() []int {
	return []int{
		0, 0,
		2, 1,
		1, 2,
	}
}

func blockb26poly2() []int {
	return []int{
		4, 0,
		3, 2,
		2, 1,
	}
}

func blockb26poly3() []int {
	return []int{
		4, 4,
		2, 3,
		3, 2,
	}
}

func blockb26poly4() []int {
	return []int{
		0, 4,
		1, 2,
		2, 3,
	}
}

// b27
//
//	----------
//	|########|
//	|##   ###|
//	|#      #|
//	|###   ##|
//	|########|
//	----------
func blockb27poly1() []int {
	return []int{
		0, 0,
		4, 0,
		0, 1,
	}
}

func blockb27poly2() []int {
	return []int{
		3, 0,
		4, 0,
		4, 4,
	}
}

func blockb27poly3() []int {
	return []int{
		4, 3,
		4, 4,
		0, 4,
	}
}

func blockb27poly4() []int {
	return []int{
		0, 4,
		0, 0,
		1, 4,
	}
}
