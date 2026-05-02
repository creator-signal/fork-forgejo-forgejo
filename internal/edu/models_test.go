package edu

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubmissionStatusValues(t *testing.T) {
	assert.Equal(t, SubmissionStatus("pending"), StatusSubmissionPending)
	assert.Equal(t, SubmissionStatus("running"), StatusSubmissionRunning)
	assert.Equal(t, SubmissionStatus("done"), StatusSubmissionDone)
	assert.Equal(t, SubmissionStatus("approved"), StatusSubmissionApproved)
	assert.Equal(t, SubmissionStatus("merged"), StatusSubmissionMerged)
	assert.Equal(t, SubmissionStatus("failed"), StatusSubmissionFailed)
}
