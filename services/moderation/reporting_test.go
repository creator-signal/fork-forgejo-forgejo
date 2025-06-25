package moderation

import (
	"testing"

	"forgejo.org/models/db"
	report_model "forgejo.org/models/moderation"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}

func TestRemoveResolvedReportsWhenNoTimeout(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	// Add a resolved report
	resolvedReport := &report_model.AbuseReport{
		Status:     report_model.ReportStatusTypeHandled,
		ReporterID: 1, ContentType: report_model.ReportedContentTypeRepository,
		ContentID: 2, Category: report_model.AbuseCategoryTypeOther,
		CreatedUnix: timeutil.TimeStampNow(),
	}
	_, err := db.GetEngine(db.DefaultContext).NoAutoTime().Insert(resolvedReport)
	require.NoError(t, err)

	err = RemoveResolvedReports(db.DefaultContext, 0)
	require.NoError(t, err)

	// Resolved reports older than a minute should be deleted.
	unittest.AssertExistsIf(t, true, resolvedReport)
}
