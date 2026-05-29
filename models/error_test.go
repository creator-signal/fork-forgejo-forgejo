package models

import (
	"fmt"
	"testing"
)

func Test_IsErrRepoFileDoesNotExist(t *testing.T) {
	t.Run("direct_error_returns_true", func(t *testing.T) {
		input := ErrRepoFileDoesNotExist{
			Path: "/some/arbitrary/path",
			Name: "repo-file-does-exist",
		}

		out := IsErrRepoFileDoesNotExist(input)
		if !out {
			t.Errorf("got %v, wanted true", out)
		}
	})

	t.Run("wrapped_error_returns_true", func(t *testing.T) {
		input := fmt.Errorf(
			"some wrapping of the base error: %w", 
			ErrRepoFileDoesNotExist{
			Path: "/path/for/wrapped/error",
			Name: "wrapped-error-repo",
		},
		)

		out := IsErrRepoFileDoesNotExist(input)
		if !out {
			t.Errorf("got %v, wanted true", out)
		}
	})
}
