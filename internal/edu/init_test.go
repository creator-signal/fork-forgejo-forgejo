package edu

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"github.com/stretchr/testify/assert"
)

func TestInit_SchemaSyncs(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	e := db.GetEngine(db.DefaultContext)
	assert.NoError(t, e.Sync(
		new(Course), new(CourseEnrollment), new(Assignment), new(Submission),
		new(TestResult), new(UserRole), new(ImportDraft), new(ImportDraftRow),
		new(InitForksTask), new(DistributeTask), new(CourseSyncTask), new(CourseSyncPR),
	))
}
