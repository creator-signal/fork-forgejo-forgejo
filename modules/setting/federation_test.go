// Copyright 2022 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var algs = []Algorithm{
	AlgorithmEd25519,
	AlgorithmHMACSHA256,
	AlgorithmP256CAVAGE,
	AlgorithmP256RFC9421,
	AlgorithmP384CAVAGE,
	AlgorithmP384RFC9421,
	AlgorithmRSASHA512CAVAGE,
	AlgorithmRSAPSSRFC9421,
	AlgorithmRSASHA256CAVAGE,
	AlgorithmRSARFC9421,
}

func TestSettingAlgorithm(t *testing.T) {
	for _, alg := range algs {
		ret, err := AlgorithmFromString(string(alg))
		require.NoError(t, err)
		assert.Equal(t, ret, alg)
	}
}

// TODO: this would probably be better as a fuzzing/mutation test
func TestBadRandomSettingAlgorithm(t *testing.T) {
	// do a number of random permutations to strings
	for range 1024 {
		firstLen := 3
		secondLen := 3
		thirdLen := 3

		// get random lengths for algorithm parts
		r1, _ := rand.Int(rand.Reader, big.NewInt(5))
		if r1.Int64() > int64(firstLen) {
			firstLen = int(r1.Int64())
		}

		r2, _ := rand.Int(rand.Reader, big.NewInt(6))
		if r2.Int64() > int64(secondLen) {
			secondLen = int(r2.Int64())
		}

		r3, _ := rand.Int(rand.Reader, big.NewInt(6))
		if r3.Int64() > int64(thirdLen) {
			thirdLen = int(r3.Int64())
		}

		firstPart := make([]byte, firstLen)
		secondPart := make([]byte, secondLen)
		thirdPart := make([]byte, thirdLen)

		// get random algorithm string parts
		_, err := rand.Read(firstPart)
		require.NoError(t, err)

		_, err = rand.Read(secondPart)
		require.NoError(t, err)

		_, err = rand.Read(thirdPart)
		require.NoError(t, err)

		// construct 2 and 3 part algorithm strings
		randAlgs := []string{
			fmt.Sprintf("%v-%v", string(firstPart), string(secondPart)),
			fmt.Sprintf("%v-%v-%v", string(firstPart), string(secondPart), string(thirdPart)),
		}

		for _, randAlgStr := range randAlgs {
			if !slices.Contains(algs, Algorithm(randAlgStr)) {
				ret, err := AlgorithmFromString(randAlgStr)
				require.Error(t, err)
				assert.Equal(t, AlgorithmNone, ret)
			}
		}
	}
}
