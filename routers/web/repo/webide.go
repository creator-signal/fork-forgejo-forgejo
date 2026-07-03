// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"path"
	"strings"
	"unicode/utf8"

	"forgejo.org/models"
	git_model "forgejo.org/models/git"
	"forgejo.org/models/unit"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/base"
	"forgejo.org/modules/git"
	"forgejo.org/modules/public"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
	files_service "forgejo.org/services/repository/files"
)

const tplWebIDE base.TplName = "repo/ide/index"

// MustEnableWebIDE 404s unless the Web IDE is enabled instance-wide.
func MustEnableWebIDE(ctx *context.Context) {
	if !setting.Repository.WebIDE.Enabled {
		ctx.NotFound("MustEnableWebIDE", nil)
		return
	}
}

type webIDEEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"` // file | dir | symlink | submodule
	Size int64  `json:"size"`
}

// webIDEBlob carries file content for the editor; empty + Editable=false when oversized/binary.
type webIDEBlob struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Encoding string `json:"encoding,omitempty"`
	Content  string `json:"content,omitempty"`
	Editable bool   `json:"editable"`
}

// webIDERef resolves ?ref= to a commit, defaulting to the repo default branch.
func webIDERef(ctx *context.Context) (*git.Commit, string, bool) {
	ref := ctx.FormString("ref")
	if ref == "" {
		ref = ctx.Repo.Repository.DefaultBranch
	}
	commit, err := ctx.Repo.GitRepo.GetCommit(ref)
	if err != nil {
		if git.IsErrNotExist(err) {
			ctx.NotFound("GetCommit", err)
		} else {
			ctx.ServerError("GetCommit", err)
		}
		return nil, ref, false
	}
	return commit, ref, true
}

// IDE renders the full-screen Web IDE shell.
func IDE(ctx *context.Context) {
	// Relaxed CSP for vscode-web workers/wasm; scoped to this route only.
	ctx.Resp.Header().Set("Content-Security-Policy",
		"default-src 'self'; "+
			"script-src 'self' 'wasm-unsafe-eval' blob:; "+
			"worker-src 'self' blob:; "+
			"style-src 'self' 'unsafe-inline'; "+
			"img-src 'self' data: blob:; "+
			"font-src 'self' data:; "+
			"connect-src 'self'")

	ctx.Data["PageIsWebIDE"] = true
	ctx.Data["Title"] = ctx.Repo.Repository.FullName() + " - Web IDE"

	ref := ctx.FormString("ref")
	if ref == "" {
		ref = ctx.Repo.Repository.DefaultBranch
	}
	openPath := cleanIDEPath(ctx.FormString("path"))

	// Base commit of ref: sent back on commit as last_commit_id so the service
	// can conflict-detect updates/deletes (blob SHAs aren't exposed to the client).
	baseCommit := ""
	if c, err := ctx.Repo.GitRepo.GetCommit(ref); err == nil {
		baseCommit = c.ID.String()
	}

	ctx.Data["WebIDERef"] = ref
	ctx.Data["WebIDEOpenPath"] = openPath
	ctx.Data["WebIDEBaseCommit"] = baseCommit
	ctx.Data["CanWriteCode"] = ctx.Repo.CanWrite(unit.TypeCode)
	ctx.PageData["webide"] = map[string]any{
		"repoLink":        ctx.Repo.RepoLink,
		"owner":           ctx.Repo.Repository.OwnerName,
		"repo":            ctx.Repo.Repository.Name,
		"ref":             ref,
		"path":            openPath,
		"baseCommit":      baseCommit,
		"defaultBranch":   ctx.Repo.Repository.DefaultBranch,
		"canWrite":        ctx.Repo.CanWrite(unit.TypeCode),
		"maxEditableSize": setting.Repository.WebIDE.MaxEditableSize,
	}

	ctx.HTML(http.StatusOK, tplWebIDE)
}

// IDEApp serves the vendored workbench index.html with no-cache, so a rebuilt
// frontend is never served stale. The hashed JS/CSS it references keep their
// normal long cache (they are immutable).
func IDEApp(ctx *context.Context) {
	data, err := public.AssetFS().ReadFile("assets", "webide", "index.html")
	if err != nil {
		ctx.NotFound("IDEApp", err)
		return
	}
	ctx.Resp.Header().Set("Content-Type", "text/html; charset=utf-8")
	ctx.Resp.Header().Set("Cache-Control", "no-cache, private, max-age=0, must-revalidate")
	ctx.Resp.WriteHeader(http.StatusOK)
	_, _ = ctx.Resp.Write(data)
}

// IDETree lists one directory level (metadata only) for the FileSystemProvider.
func IDETree(ctx *context.Context) {
	commit, _, ok := webIDERef(ctx)
	if !ok {
		return
	}

	treePath := cleanIDEPath(ctx.FormString("path"))

	tree := &commit.Tree
	if treePath != "" {
		var err error
		tree, err = commit.SubTree(treePath)
		if err != nil {
			if git.IsErrNotExist(err) {
				ctx.NotFound("SubTree", err)
			} else {
				ctx.ServerError("SubTree", err)
			}
			return
		}
	}

	entries, err := tree.ListEntries()
	if err != nil {
		ctx.ServerError("ListEntries", err)
		return
	}

	out := make([]webIDEEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, webIDEEntry{
			Name: e.Name(),
			Path: path.Join(treePath, e.Name()),
			Type: entryType(e),
			Size: e.Size(),
		})
	}
	ctx.JSON(http.StatusOK, out)
}

// IDEBlob returns file content for the editor, or streams raw bytes (LFS-aware) when download/oversized/binary.
func IDEBlob(ctx *context.Context) {
	commit, _, ok := webIDERef(ctx)
	if !ok {
		return
	}

	treePath := cleanIDEPath(ctx.FormString("path"))
	if treePath == "" {
		ctx.NotFound("IDEBlob", nil)
		return
	}
	ctx.Repo.TreePath = treePath // ServeBlobOrLFS reads it

	entry, err := commit.GetTreeEntryByPath(treePath)
	if err != nil {
		if git.IsErrNotExist(err) {
			ctx.NotFound("GetTreeEntryByPath", err)
		} else {
			ctx.ServerError("GetTreeEntryByPath", err)
		}
		return
	}
	if entry.IsDir() || entry.IsSubmodule() {
		ctx.NotFound("IDEBlob", nil)
		return
	}

	blob := entry.Blob()
	maxSize := setting.Repository.WebIDE.MaxEditableSize

	if ctx.FormBool("download") || blob.Size() > maxSize {
		if err := ServeBlobOrLFS(ctx, blob, nil); err != nil {
			ctx.ServerError("ServeBlobOrLFS", err)
		}
		return
	}

	rc, err := blob.DataAsync()
	if err != nil {
		ctx.ServerError("DataAsync", err)
		return
	}
	defer rc.Close()

	buf, err := io.ReadAll(io.LimitReader(rc, maxSize+1))
	if err != nil {
		ctx.ServerError("ReadAll", err)
		return
	}
	if int64(len(buf)) > maxSize { // grew past the cap between stat and read
		if err := ServeBlobOrLFS(ctx, blob, nil); err != nil {
			ctx.ServerError("ServeBlobOrLFS", err)
		}
		return
	}

	editable := isEditableText(buf)
	resp := webIDEBlob{Path: treePath, Size: blob.Size(), Editable: editable}
	if editable {
		resp.Encoding = "base64"
		resp.Content = base64.StdEncoding.EncodeToString(buf)
	}
	ctx.JSON(http.StatusOK, resp)
}

func entryType(e *git.TreeEntry) string {
	switch {
	case e.IsSubmodule():
		return "submodule"
	case e.IsDir():
		return "dir"
	case e.IsLink():
		return "symlink"
	default:
		return "file"
	}
}

// isEditableText treats valid UTF-8 without NUL bytes as text (conservative binary heuristic).
func isEditableText(buf []byte) bool {
	if !utf8.Valid(buf) {
		return false
	}
	for _, b := range buf {
		if b == 0 {
			return false
		}
	}
	return true
}

// cleanIDEPath normalizes a request path and rejects traversal and .git components.
func cleanIDEPath(p string) string {
	p = strings.TrimPrefix(path.Clean("/"+strings.TrimSpace(p)), "/")
	if p == "." {
		return ""
	}
	for _, seg := range strings.Split(p, "/") {
		if strings.EqualFold(seg, ".git") {
			return ""
		}
	}
	return p
}

type webIDECommitFile struct {
	Operation string `json:"operation"` // create | update | delete
	Path      string `json:"path"`
	FromPath  string `json:"from_path"`
	Content   string `json:"content"` // base64, for create/update
	SHA       string `json:"sha"`     // expected blob SHA (conflict detection)
}

type webIDECommitRequest struct {
	Branch       string             `json:"branch"`
	NewBranch    string             `json:"new_branch"`
	Message      string             `json:"message"`
	Signoff      bool               `json:"signoff"`
	LastCommitID string             `json:"last_commit_id"` // commit-level conflict detection (same-branch only)
	CommitMailID int64              `json:"commit_mail_id"` // 0=doer, -1=placeholder, >0=activated email
	Files        []webIDECommitFile `json:"files"`
}

// IDECommit applies a multi-file commit via the existing ChangeRepoFiles service.
func IDECommit(ctx *context.Context) {
	var req webIDECommitRequest
	if err := json.NewDecoder(ctx.Req.Body).Decode(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if len(req.Files) == 0 {
		ctx.JSON(http.StatusUnprocessableEntity, map[string]string{"error": "no files to commit"})
		return
	}

	baseBranch := req.Branch
	if baseBranch == "" {
		baseBranch = ctx.Repo.Repository.DefaultBranch
	}
	newBranch := baseBranch
	if req.NewBranch != "" {
		if !git.IsValidRefPattern(git.BranchPrefix + req.NewBranch) {
			ctx.JSON(http.StatusUnprocessableEntity, map[string]string{"error": "invalid new branch name"})
			return
		}
		newBranch = req.NewBranch
	}

	var identity *files_service.IdentityOptions
	if req.CommitMailID != 0 {
		var ok bool
		if identity, ok = webIDEIdentity(ctx, req.CommitMailID); !ok {
			return
		}
	}

	changeFiles := make([]*files_service.ChangeRepoFile, 0, len(req.Files))
	for _, f := range req.Files {
		crf := &files_service.ChangeRepoFile{
			Operation:    f.Operation,
			TreePath:     cleanIDEPath(f.Path),
			FromTreePath: cleanIDEPath(f.FromPath),
			SHA:          f.SHA,
		}
		if f.Operation != "delete" {
			content, err := base64.StdEncoding.DecodeString(f.Content)
			if err != nil {
				ctx.JSON(http.StatusUnprocessableEntity, map[string]string{"error": "invalid base64 content for " + f.Path})
				return
			}
			crf.ContentReader = bytes.NewReader(content)
		}
		changeFiles = append(changeFiles, crf)
	}

	message := strings.TrimSpace(req.Message)
	if message == "" {
		message = "Update files via Web IDE"
	}

	resp, err := files_service.ChangeRepoFiles(ctx, ctx.Repo.Repository, ctx.Doer, &files_service.ChangeRepoFilesOptions{
		LastCommitID: req.LastCommitID,
		OldBranch:    baseBranch,
		NewBranch:    newBranch,
		Message:      message,
		Files:        changeFiles,
		Signoff:      req.Signoff,
		Author:       identity,
		Committer:    identity,
	})
	if err != nil {
		webIDECommitError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, resp)
}

// webIDEIdentity resolves an explicit commit-email selection, writing a JSON error on failure.
func webIDEIdentity(ctx *context.Context, commitMailID int64) (*files_service.IdentityOptions, bool) {
	identity := &files_service.IdentityOptions{Name: ctx.Doer.Name}
	if commitMailID == -1 {
		identity.Email = ctx.Doer.GetPlaceholderEmail()
		return identity, true
	}
	email, err := user_model.GetEmailAddressByID(ctx, ctx.Doer.ID, commitMailID)
	if err != nil {
		ctx.ServerError("GetEmailAddressByID", err)
		return nil, false
	}
	if email == nil || !email.IsActivated {
		ctx.JSON(http.StatusUnprocessableEntity, map[string]string{"error": "invalid commit mail"})
		return nil, false
	}
	identity.Email = email.Email
	return identity, true
}

// webIDECommitError maps ChangeRepoFiles errors to JSON responses like the existing editor/API paths.
func webIDECommitError(ctx *context.Context, err error) {
	switch {
	case git.IsErrNotExist(err), models.IsErrRepoFileDoesNotExist(err):
		ctx.JSON(http.StatusNotFound, map[string]string{"error": "file no longer exists"})
	case git_model.IsErrLFSFileLocked(err):
		ctx.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
	case models.IsErrFilenameInvalid(err), models.IsErrFilePathInvalid(err), models.IsErrSHAOrCommitIDNotProvided(err):
		ctx.JSON(http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
	case models.IsErrRepoFileAlreadyExists(err):
		ctx.JSON(http.StatusConflict, map[string]string{"error": "file already exists"})
	case git.IsErrBranchNotExist(err):
		ctx.JSON(http.StatusNotFound, map[string]string{"error": "branch does not exist"})
	case git_model.IsErrBranchAlreadyExists(err):
		ctx.JSON(http.StatusConflict, map[string]string{"error": "branch already exists"})
	case models.IsErrCommitIDDoesNotMatch(err), models.IsErrSHADoesNotMatch(err), git.IsErrPushOutOfDate(err):
		ctx.JSON(http.StatusConflict, map[string]string{"error": "the file or base branch has moved; reload and retry"})
	case git.IsErrPushRejected(err), models.IsErrUserCannotCommit(err), models.IsErrFilePathProtected(err):
		ctx.JSON(http.StatusForbidden, map[string]string{"error": "commit rejected by branch protection"})
	default:
		ctx.ServerError("ChangeRepoFiles", err)
	}
}
