package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ensureEduTables creates edu_* tables if they don't already exist.
// Xorm Sync creates tables without the 'edu_' prefix, but the squirrel-based
// repository layer queries 'edu_*' tables. This helper bridges the gap for tests.
func ensureEduTables(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	engine := db.GetEngine(ctx)

	tables := []string{
		`CREATE TABLE IF NOT EXISTS edu_courses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			creator_id INTEGER NOT NULL,
			org_id INTEGER DEFAULT 0,
			start_unix INTEGER DEFAULT 0,
			end_unix INTEGER DEFAULT 0,
			created_unix INTEGER DEFAULT 0,
			updated_unix INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS edu_course_enrollments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			course_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			role VARCHAR(20) NOT NULL DEFAULT 'student',
			created_unix INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS edu_assignments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			course_id INTEGER DEFAULT 0,
			repo_id INTEGER NOT NULL,
			title VARCHAR(255) NOT NULL,
			description TEXT,
			deadline_unix INTEGER DEFAULT 0,
			created_unix INTEGER DEFAULT 0,
			updated_unix INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS edu_submissions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			assignment_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			student_repo_id INTEGER DEFAULT 0,
			status VARCHAR(50) NOT NULL DEFAULT 'started',
			grade INTEGER DEFAULT -1,
			comment TEXT DEFAULT '',
			graded_by_id INTEGER DEFAULT 0,
			graded_unix INTEGER DEFAULT 0,
			created_unix INTEGER DEFAULT 0,
			updated_unix INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS edu_test_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			submission_id INTEGER NOT NULL,
			commit_sha VARCHAR(64) NOT NULL,
			score INTEGER DEFAULT 0,
			details TEXT,
			created_unix INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS edu_user_role (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL UNIQUE,
			role VARCHAR(20) NOT NULL,
			created_unix INTEGER DEFAULT 0,
			updated_unix INTEGER DEFAULT 0
		)`,
	}

	for _, ddl := range tables {
		_, err := engine.Exec(ddl)
		require.NoError(t, err)
	}
}

// cleanEduTables deletes all data from edu tables.
func cleanEduTables(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	engine := db.GetEngine(ctx)

	for _, tbl := range []string{"edu_submissions", "edu_test_results", "edu_assignments", "edu_course_enrollments", "edu_courses", "edu_user_role"} {
		_, err := engine.Exec(fmt.Sprintf("DELETE FROM %s", tbl))
		require.NoError(t, err)
	}
}

// setupEduEnv is a helper that prepares test env + ensures edu tables exist and are clean.
func setupEduEnv(t *testing.T) func() {
	t.Helper()
	deferFn := tests.PrepareTestEnv(t)
	ensureEduTables(t)
	cleanEduTables(t)
	return deferFn
}

// insertEduCourse inserts a course and returns its ID.
func insertEduCourse(t *testing.T, name, description string, creatorID int64) int64 {
	t.Helper()
	ctx := context.Background()
	engine := db.GetEngine(ctx)
	now := time.Now().Unix()

	res, err := engine.Exec(
		"INSERT INTO edu_courses (name, description, creator_id, org_id, start_unix, end_unix, created_unix, updated_unix) VALUES (?, ?, ?, 0, ?, ?, ?, ?)",
		name, description, creatorID, now, now+86400*30, now, now,
	)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return id
}

// insertEduEnrollment inserts a course enrollment.
func insertEduEnrollment(t *testing.T, courseID, userID int64, role string) {
	t.Helper()
	ctx := context.Background()
	engine := db.GetEngine(ctx)
	now := time.Now().Unix()

	_, err := engine.Exec(
		"INSERT INTO edu_course_enrollments (course_id, user_id, role, created_unix) VALUES (?, ?, ?, ?)",
		courseID, userID, role, now,
	)
	require.NoError(t, err)
}

// insertEduAssignment inserts an assignment and returns its ID.
func insertEduAssignment(t *testing.T, courseID, repoID int64, title, description string, deadlineUnix int64) int64 {
	t.Helper()
	ctx := context.Background()
	engine := db.GetEngine(ctx)
	now := time.Now().Unix()

	res, err := engine.Exec(
		"INSERT INTO edu_assignments (course_id, repo_id, title, description, deadline_unix, created_unix, updated_unix) VALUES (?, ?, ?, ?, ?, ?, ?)",
		courseID, repoID, title, description, deadlineUnix, now, now,
	)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return id
}

// insertEduSubmission inserts a submission and returns its ID.
func insertEduSubmission(t *testing.T, assignmentID, userID int64, status string) int64 {
	t.Helper()
	ctx := context.Background()
	engine := db.GetEngine(ctx)
	now := time.Now().Unix()

	res, err := engine.Exec(
		"INSERT INTO edu_submissions (assignment_id, user_id, student_repo_id, status, grade, comment, graded_by_id, graded_unix, created_unix, updated_unix) VALUES (?, ?, 0, ?, -1, '', 0, 0, ?, ?)",
		assignmentID, userID, status, now, now,
	)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return id
}

// insertEduUserRole inserts a user role into the Xorm-managed 'edu_user_role' table.
// This is used by edu.GetUserRole (role.go) which queries via Xorm ORM.
func insertEduUserRole(t *testing.T, userID int64, role string) {
	t.Helper()
	ctx := context.Background()
	engine := db.GetEngine(ctx)
	now := time.Now().Unix()

	// The UserRole struct is managed by Xorm, table name = 'edu_user_role'
	_, err := engine.Exec(
		"INSERT OR REPLACE INTO edu_user_role (user_id, role, created_unix, updated_unix) VALUES (?, ?, ?, ?)",
		userID, role, now, now,
	)
	require.NoError(t, err)
}

// --------------------------------------------------------------------------
// 1. Course tests
// --------------------------------------------------------------------------

func TestEduCourseCreate(t *testing.T) {
	defer setupEduEnv(t)()

	session := loginUser(t, "user1")

	// POST create course
	req := NewRequestWithValues(t, "POST", "/edu/teacher/courses/new", map[string]string{
		"name":        "New Test Course",
		"description": "Course description",
		"start_date":  "2026-01-01T00:00",
		"end_date":    "2026-06-30T23:59",
	})
	session.MakeRequest(t, req, http.StatusSeeOther)

	// Verify course was created in DB
	ctx := context.Background()
	engine := db.GetEngine(ctx)
	var name string
	has, err := engine.SQL("SELECT name FROM edu_courses WHERE name = ?", "New Test Course").Get(&name)
	require.NoError(t, err)
	assert.True(t, has, "course should exist in database")
	assert.Equal(t, "New Test Course", name)
}

func TestEduCourseDetail(t *testing.T) {
	defer setupEduEnv(t)()

	session := loginUser(t, "user1")

	// Insert course data, enroll user1 so they can see it
	courseID := insertEduCourse(t, "Detail Course", "Detailed description", 1)
	insertEduEnrollment(t, courseID, 1, "teacher")

	// GET course detail
	req := NewRequest(t, "GET", fmt.Sprintf("/edu/teacher/courses/%d", courseID))
	resp := session.MakeRequest(t, req, http.StatusOK)

	assert.Contains(t, resp.Body.String(), "Detail Course")
	assert.Contains(t, resp.Body.String(), "Detailed description")
}

func TestEduCourseEdit(t *testing.T) {
	defer setupEduEnv(t)()

	session := loginUser(t, "user1")

	courseID := insertEduCourse(t, "Old Name", "Old desc", 1)
	insertEduEnrollment(t, courseID, 1, "teacher")

	// POST edit
	req := NewRequestWithValues(t, "POST", fmt.Sprintf("/edu/teacher/courses/%d/edit", courseID), map[string]string{
		"name":        "Updated Name",
		"description": "Updated desc",
	})
	session.MakeRequest(t, req, http.StatusSeeOther)

	// Verify updated data
	req = NewRequest(t, "GET", fmt.Sprintf("/edu/teacher/courses/%d", courseID))
	resp := session.MakeRequest(t, req, http.StatusOK)
	assert.Contains(t, resp.Body.String(), "Updated Name")
	assert.Contains(t, resp.Body.String(), "Updated desc")
}

func TestEduCourseDelete(t *testing.T) {
	defer setupEduEnv(t)()

	session := loginUser(t, "user1")

	courseID := insertEduCourse(t, "To Be Deleted", "Will be removed", 1)
	insertEduEnrollment(t, courseID, 1, "teacher")

	// POST delete
	req := NewRequestWithValues(t, "POST", fmt.Sprintf("/edu/teacher/courses/%d/delete", courseID), map[string]string{})
	session.MakeRequest(t, req, http.StatusSeeOther)

	// Verify course is gone from DB
	ctx := context.Background()
	engine := db.GetEngine(ctx)
	var name string
	has, err := engine.SQL("SELECT name FROM edu_courses WHERE id = ?", courseID).Get(&name)
	require.NoError(t, err)
	assert.False(t, has, "course should be deleted from database")
}

// --------------------------------------------------------------------------
// 2. Enrollment tests
// --------------------------------------------------------------------------

func TestEduEnrollUser(t *testing.T) {
	defer setupEduEnv(t)()

	session := loginUser(t, "user1")

	courseID := insertEduCourse(t, "Enrollment Course", "Test enrollment", 1)
	insertEduEnrollment(t, courseID, 1, "teacher")

	// POST enroll user2
	req := NewRequestWithValues(t, "POST", fmt.Sprintf("/edu/teacher/courses/%d/enroll", courseID), map[string]string{
		"username": "user2",
		"role":     "student",
	})
	session.MakeRequest(t, req, http.StatusSeeOther)

	// Verify user2 appears in course detail
	req = NewRequest(t, "GET", fmt.Sprintf("/edu/teacher/courses/%d", courseID))
	resp := session.MakeRequest(t, req, http.StatusOK)
	assert.Contains(t, resp.Body.String(), "user2")
}

func TestEduRemoveEnrollment(t *testing.T) {
	defer setupEduEnv(t)()

	session := loginUser(t, "user1")

	courseID := insertEduCourse(t, "Unenroll Course", "Test unenroll", 1)
	insertEduEnrollment(t, courseID, 1, "teacher")
	insertEduEnrollment(t, courseID, 2, "student")

	// Verify user2 is enrolled
	req := NewRequest(t, "GET", fmt.Sprintf("/edu/teacher/courses/%d", courseID))
	resp := session.MakeRequest(t, req, http.StatusOK)
	assert.Contains(t, resp.Body.String(), "user2")

	// POST unenroll
	req = NewRequestWithValues(t, "POST", fmt.Sprintf("/edu/teacher/courses/%d/unenroll", courseID), map[string]string{
		"user_id": "2",
	})
	session.MakeRequest(t, req, http.StatusSeeOther)

	// Verify user2 is gone from DB
	ctx := context.Background()
	engine := db.GetEngine(ctx)
	var count int64
	_, err := engine.SQL("SELECT COUNT(*) FROM edu_course_enrollments WHERE course_id = ? AND user_id = 2", courseID).Get(&count)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count, "user2 should not be enrolled anymore")
}

// --------------------------------------------------------------------------
// 3. Assignment tests
// --------------------------------------------------------------------------

func TestEduStudentAssignmentList(t *testing.T) {
	defer setupEduEnv(t)()

	// Setup: course + enrollment for user2 + assignment
	courseID := insertEduCourse(t, "Student Course", "desc", 1)
	insertEduEnrollment(t, courseID, 1, "teacher")
	insertEduEnrollment(t, courseID, 2, "student")
	insertEduAssignment(t, courseID, 1, "Visible Assignment", "Should be visible", time.Now().Add(24*time.Hour).Unix())

	// user2 (enrolled student) sees the assignment
	session2 := loginUser(t, "user2")
	req := NewRequest(t, "GET", "/edu/student/assignments")
	resp := session2.MakeRequest(t, req, http.StatusOK)
	assert.Contains(t, resp.Body.String(), "Visible Assignment")

	// user4 (not enrolled) does not see the assignment
	session4 := loginUser(t, "user4")
	req = NewRequest(t, "GET", "/edu/student/assignments")
	resp = session4.MakeRequest(t, req, http.StatusOK)
	assert.NotContains(t, resp.Body.String(), "Visible Assignment")
}

func TestEduAssignmentDetail(t *testing.T) {
	defer setupEduEnv(t)()

	courseID := insertEduCourse(t, "Detail Course", "desc", 1)
	insertEduEnrollment(t, courseID, 2, "student")
	assignmentID := insertEduAssignment(t, courseID, 1, "Detail Assignment", "Homework details", time.Now().Add(24*time.Hour).Unix())

	session := loginUser(t, "user2")
	req := NewRequest(t, "GET", fmt.Sprintf("/edu/student/assignments/%d", assignmentID))
	resp := session.MakeRequest(t, req, http.StatusOK)

	assert.Contains(t, resp.Body.String(), "Detail Assignment")
	assert.Contains(t, resp.Body.String(), "Homework details")
}

func TestEduAssignmentJoinDeadlinePassed(t *testing.T) {
	defer setupEduEnv(t)()

	courseID := insertEduCourse(t, "Expired Course", "desc", 1)
	insertEduEnrollment(t, courseID, 2, "student")
	// deadline in the past
	assignmentID := insertEduAssignment(t, courseID, 1, "Expired Assignment", "Too late", time.Now().Add(-24*time.Hour).Unix())

	session := loginUser(t, "user2")
	req := NewRequestWithValues(t, "POST", fmt.Sprintf("/edu/student/assignments/%d/join", assignmentID), map[string]string{})
	// Should redirect with flash error (deadline passed)
	resp := session.MakeRequest(t, req, http.StatusSeeOther)
	assert.Contains(t, resp.Header().Get("Location"), fmt.Sprintf("/edu/student/assignments/%d", assignmentID))
}

// --------------------------------------------------------------------------
// 4. Instructor Dashboard tests
// --------------------------------------------------------------------------

func TestEduInstructorSubmissions(t *testing.T) {
	defer setupEduEnv(t)()

	// user1 is admin, has access to repo 1
	courseID := insertEduCourse(t, "Instructor Course", "desc", 1)
	insertEduEnrollment(t, courseID, 1, "teacher")
	insertEduEnrollment(t, courseID, 2, "student")
	assignmentID := insertEduAssignment(t, courseID, 1, "Graded Assignment", "desc", time.Now().Add(24*time.Hour).Unix())
	insertEduSubmission(t, assignmentID, 2, "started")

	session := loginUser(t, "user1")
	req := NewRequest(t, "GET", fmt.Sprintf("/edu/teacher/assignments/%d/submissions", assignmentID))
	resp := session.MakeRequest(t, req, http.StatusOK)

	// Should show submissions table
	assert.Contains(t, resp.Body.String(), "user2")
}

func TestEduInstructorSubmissionsAccessDenied(t *testing.T) {
	defer setupEduEnv(t)()

	courseID := insertEduCourse(t, "Restricted Course", "desc", 1)
	insertEduEnrollment(t, courseID, 2, "student")
	assignmentID := insertEduAssignment(t, courseID, 1, "Restricted Assignment", "desc", time.Now().Add(24*time.Hour).Unix())

	// user4 is not the repo owner and not admin
	session := loginUser(t, "user4")
	req := NewRequest(t, "GET", fmt.Sprintf("/edu/teacher/assignments/%d/submissions", assignmentID))
	session.MakeRequest(t, req, http.StatusForbidden)
}

// --------------------------------------------------------------------------
// 5. Grading tests
// --------------------------------------------------------------------------

func TestEduGradeSubmission(t *testing.T) {
	defer setupEduEnv(t)()

	courseID := insertEduCourse(t, "Grading Course", "desc", 1)
	insertEduEnrollment(t, courseID, 1, "teacher")
	insertEduEnrollment(t, courseID, 2, "student")
	assignmentID := insertEduAssignment(t, courseID, 1, "Grade Assignment", "desc", time.Now().Add(24*time.Hour).Unix())
	subID := insertEduSubmission(t, assignmentID, 2, "submitted")

	session := loginUser(t, "user1")

	// POST grade
	req := NewRequestWithValues(t, "POST", fmt.Sprintf("/edu/teacher/assignments/%d/submissions/%d/grade", assignmentID, subID), map[string]string{
		"grade":   "85",
		"comment": "Good work",
	})
	session.MakeRequest(t, req, http.StatusSeeOther)

	// Verify grade in submission detail
	req = NewRequest(t, "GET", fmt.Sprintf("/edu/teacher/assignments/%d/submissions/%d", assignmentID, subID))
	resp := session.MakeRequest(t, req, http.StatusOK)
	assert.Contains(t, resp.Body.String(), "85")
	assert.Contains(t, resp.Body.String(), "Good work")
}

func TestEduGradeValidation(t *testing.T) {
	defer setupEduEnv(t)()

	courseID := insertEduCourse(t, "Validation Course", "desc", 1)
	insertEduEnrollment(t, courseID, 1, "teacher")
	insertEduEnrollment(t, courseID, 2, "student")
	assignmentID := insertEduAssignment(t, courseID, 1, "Validate Assignment", "desc", time.Now().Add(24*time.Hour).Unix())
	subID := insertEduSubmission(t, assignmentID, 2, "submitted")

	session := loginUser(t, "user1")

	// Grade > 100 should be rejected (redirect back with error flash)
	req := NewRequestWithValues(t, "POST", fmt.Sprintf("/edu/teacher/assignments/%d/submissions/%d/grade", assignmentID, subID), map[string]string{
		"grade":   "150",
		"comment": "Too high",
	})
	session.MakeRequest(t, req, http.StatusSeeOther)

	// Grade < 0 should be rejected
	req = NewRequestWithValues(t, "POST", fmt.Sprintf("/edu/teacher/assignments/%d/submissions/%d/grade", assignmentID, subID), map[string]string{
		"grade":   "-1",
		"comment": "Negative",
	})
	session.MakeRequest(t, req, http.StatusSeeOther)

	// Verify the grade was NOT saved (still -1)
	ctx := context.Background()
	engine := db.GetEngine(ctx)
	var grade int
	_, err := engine.SQL("SELECT grade FROM edu_submissions WHERE id = ?", subID).Get(&grade)
	require.NoError(t, err)
	assert.Equal(t, -1, grade)
}

// --------------------------------------------------------------------------
// 6. Dashboard redirect test
// --------------------------------------------------------------------------

func TestEduDashboardRedirect(t *testing.T) {
	defer setupEduEnv(t)()

	// Use user4 (non-admin) as teacher to avoid site-admin redirect to /edu/admin
	insertEduUserRole(t, 4, "teacher")

	session := loginUser(t, "user4")
	req := NewRequest(t, "GET", "/edu/dashboard")
	resp := session.MakeRequest(t, req, http.StatusSeeOther)

	// Should redirect teacher to teacher dashboard
	location := resp.Header().Get("Location")
	assert.Contains(t, location, "/edu/teacher")
}

func TestEduDashboardStudentRedirect(t *testing.T) {
	defer setupEduEnv(t)()

	// Set user2 as student
	insertEduUserRole(t, 2, "student")

	session := loginUser(t, "user2")
	req := NewRequest(t, "GET", "/edu/dashboard")
	resp := session.MakeRequest(t, req, http.StatusSeeOther)

	location := resp.Header().Get("Location")
	assert.Contains(t, location, "/edu/student/assignments")
}
