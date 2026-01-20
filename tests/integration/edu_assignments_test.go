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
)

func TestEduAssignmentsList(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user1")

	ctx := context.Background()
	engine := db.GetEngine(ctx)

	_, err := engine.Exec("INSERT INTO edu_assignments (repo_id, title, description, deadline_unix, created_unix, updated_unix) VALUES (?, ?, ?, ?, ?, ?)",
		1, "Integration Test Assignment", "Desc", time.Now().Add(1*time.Hour).Unix(), time.Now().Unix(), time.Now().Unix())
	assert.NoError(t, err)

	req := NewRequest(t, "GET", "/edu/assignments")
	resp := session.MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, resp.Body)
	assert.Contains(t, htmlDoc.doc.Find("h1").Text(), "Assignments")
	assert.Contains(t, resp.Body.String(), "Integration Test Assignment")
}

func TestEduAssignmentsDetail(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user1")

	ctx := context.Background()
	engine := db.GetEngine(ctx)

	res, err := engine.Exec("INSERT INTO edu_assignments (repo_id, title, description, deadline_unix, created_unix, updated_unix) VALUES (?, ?, ?, ?, ?, ?)",
		1, "Detail Test Assignment", "Detail Desc", time.Now().Add(1*time.Hour).Unix(), time.Now().Unix(), time.Now().Unix())
	assert.NoError(t, err)

	id, err := res.LastInsertId()
	assert.NoError(t, err)

	req := NewRequest(t, "GET", fmt.Sprintf("/edu/assignments/%d", id))
	resp := session.MakeRequest(t, req, http.StatusOK)

	assert.Contains(t, resp.Body.String(), "Detail Test Assignment")
	assert.Contains(t, resp.Body.String(), "Detail Desc")
}
