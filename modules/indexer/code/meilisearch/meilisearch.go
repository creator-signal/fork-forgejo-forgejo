// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package meilisearch

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/analyze"
	"forgejo.org/modules/git"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/modules/indexer/code/internal"
	indexer_internal "forgejo.org/modules/indexer/internal"
	inner_meilisearch "forgejo.org/modules/indexer/internal/meilisearch"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/typesniffer"

	"github.com/go-enry/go-enry/v2"
	"github.com/meilisearch/meilisearch-go"
)

const (
	repoIndexerLatestVersion = 1
	maxTotalHits             = 10000
)

var _ internal.Indexer = &Indexer{}

// Indexer implements Indexer interface
type Indexer struct {
	inner *inner_meilisearch.Indexer
	indexer_internal.Indexer
}

func NewIndexer(url, apiKey, indexerName string) *Indexer {
	settings := &meilisearch.Settings{
		RankingRules: []string{
			"sort",
			"exactness",
			"attribute",
			"words",
			"proximity",
			"typo",
		},
		SearchableAttributes: []string{
			"content",
			"filename",
			"language",
			"commit_id",
		},
		DisplayedAttributes: []string{
			"id",
			"repo_id",
			"content",
			"commit_id",
			"filename",
			"language",
			"updated_at",
		},
		FilterableAttributes: []string{
			"repo_id", "language",
		},
		SortableAttributes: []string{
			"updated_at",
		},
		Pagination: &meilisearch.Pagination{
			MaxTotalHits: maxTotalHits,
		},
		TypoTolerance: &meilisearch.TypoTolerance{
			Enabled: false,
		},
	}

	inner := inner_meilisearch.NewIndexer(url, apiKey, indexerName, repoIndexerLatestVersion, settings)

	return &Indexer{
		inner:   inner,
		Indexer: inner,
	}
}

func (b *Indexer) addUpdate(ctx context.Context, batchWriter git.WriteCloserError, batchReader *bufio.Reader, sha string, update internal.FileUpdate, repo *repo_model.Repository) (meiliItem, error) {
	// Ignore vendored files in code search
	if setting.Indexer.ExcludeVendored && analyze.IsVendor(update.Filename) {
		return meiliItem{}, nil
	}

	size := update.Size
	var err error
	if !update.Sized {
		var stdout string
		stdout, _, err = git.NewCommand(ctx, "cat-file", "-s").AddDynamicArguments(update.BlobSha).RunStdString(&git.RunOpts{Dir: repo.RepoPath()})
		if err != nil {
			return meiliItem{}, err
		}
		if size, err = strconv.ParseInt(strings.TrimSpace(stdout), 10, 64); err != nil {
			return meiliItem{}, fmt.Errorf("misformatted git cat-file output: %w", err)
		}
	}

	id := filenameIndexerID(repo.ID, update.Filename)

	if size > setting.Indexer.MaxIndexerFileSize {
		// file too big, delete it
		return meiliItem{
			ID:     id,
			Action: mActionDelete,
		}, nil
	}

	if _, err := batchWriter.Write([]byte(update.BlobSha + "\n")); err != nil {
		return meiliItem{}, err
	}

	_, _, size, err = git.ReadBatchLine(batchReader)
	if err != nil {
		return meiliItem{}, err
	}

	fileContents, err := io.ReadAll(io.LimitReader(batchReader, size))

	if err != nil {
		return meiliItem{}, err
	} else if !typesniffer.DetectContentType(fileContents).IsText() {
		return meiliItem{}, nil
	}

	if _, err = batchReader.Discard(1); err != nil {
		return meiliItem{}, err
	}

	return meiliItem{
		ID:     id,
		Action: mActionCreate,
		Doc: map[string]any{
			"id":         id,
			"repo_id":    repo.ID,
			"content":    string(fileContents),
			"filename":   update.Filename,
			"commit_id":  sha,
			"language":   analyze.GetCodeLanguage(update.Filename, fileContents),
			"updated_at": timeutil.TimeStampNow(),
		},
	}, nil
}

func (b *Indexer) addDelete(filename string, repo *repo_model.Repository) meiliItem {
	id := filenameIndexerID(repo.ID, filename)
	return meiliItem{
		ID:     id,
		Action: mActionDelete,
	}
}

func filenameIndexerID(repoID int64, filename string) string {
	repoPart := strconv.FormatInt(repoID, 36)
	filePart := base64.RawURLEncoding.EncodeToString([]byte(filename))
	return repoPart + "_" + filePart
}

type mAction int

const (
	mActionCreate mAction = iota
	mActionDelete
)

type meiliItem struct {
	Action mAction
	ID     string
	Doc    map[string]any
}

// Index will save the index data
func (b *Indexer) Index(ctx context.Context, repo *repo_model.Repository, sha string, changes *internal.RepoChanges) error {
	reqs := make([]meiliItem, 0)

	if len(changes.Updates) > 0 {
		r, err := gitrepo.OpenRepository(ctx, repo)
		if err != nil {
			log.Error("Unable to open git repo: %s for %-v: %v", repo.RepoPath(), repo, err)
			return err
		}
		defer r.Close()

		batch, err := r.NewBatch(ctx)
		if err != nil {
			return fmt.Errorf("failed to create git batch: %w", err)
		}
		defer batch.Close()

		for _, update := range changes.Updates {
			updateReq, err := b.addUpdate(ctx, batch.Writer, batch.Reader, sha, update, repo)
			if err != nil {
				return fmt.Errorf("addUpdate failed for file %s: %w", update.Filename, err)
			}
			if updateReq.ID != "" && updateReq.Action == mActionCreate {
				reqs = append(reqs, updateReq)
			}
		}
	}

	for _, filename := range changes.RemovedFilenames {
		reqs = append(reqs, b.addDelete(filename, repo))
	}

	for _, req := range reqs {
		switch req.Action {
		case mActionCreate:
			_, err := b.inner.Client.Index(b.inner.VersionedIndexName()).AddDocuments(req.Doc)
			if err != nil {
				return b.checkError(fmt.Errorf("failed to add document: %w", err))
			}
		case mActionDelete:
			_, err := b.inner.Client.Index(b.inner.VersionedIndexName()).DeleteDocument(req.ID)
			if err != nil {
				return b.checkError(fmt.Errorf("failed to delete document ID %s: %w", req.ID, err))
			}
		}
	}

	return nil
}

// Delete deletes indexes by ids
func (b *Indexer) Delete(ctx context.Context, repoID int64) error {
	_, err := b.inner.Client.Index(b.inner.VersionedIndexName()).DeleteDocumentsByFilter(fmt.Sprintf("repo_id = %d", repoID))
	if err != nil {
		return b.checkError(fmt.Errorf("failed to delete documents for repo %d: %w", repoID, err))
	}
	return nil
}

func convertResult(searchResult *meilisearch.SearchResponse, pageSize int) (int64, []*internal.SearchResult, []*internal.SearchResultLanguages, error) {
	if searchResult == nil {
		return 0, nil, nil, fmt.Errorf("search result is nil")
	}

	fileGroups := make(map[string]*FileMatches)

	for _, hit := range searchResult.Hits {
		doc, ok := hit.(map[string]any)
		if !ok {
			continue
		}

		content := safeString(doc["content"])

		var occurrences []internal.Match
		if matchesMap, ok := doc["_matchesPosition"].(map[string]any); ok {
			if contentMatches, ok := matchesMap["content"].([]any); ok {
				for _, match := range contentMatches {
					if m, ok := match.(map[string]any); ok {
						start := int(m["start"].(float64))
						length := int(m["length"].(float64))
						occurrences = append(occurrences, internal.Match{
							Start: start,
							End:   start + length,
						})
					}
				}
			}
		}

		language, _ := doc["language"].(string)
		repoID, _ := doc["repo_id"].(float64)
		updatedAt, _ := doc["updated_at"].(float64)
		filename := safeString(doc["filename"])
		commitID := safeString(doc["commit_id"])

		fileKey := fmt.Sprintf("%d:%s", int64(repoID), filename)

		fileGroup, exists := fileGroups[fileKey]
		if !exists {
			fileGroup = &FileMatches{
				RepoID:      int64(repoID),
				Filename:    filename,
				CommitID:    commitID,
				Content:     content,
				UpdatedUnix: timeutil.TimeStamp(updatedAt),
				Language:    language,
				Color:       enry.GetColor(language),
			}
			fileGroups[fileKey] = fileGroup
		}

		fileGroup.Matches = append(fileGroup.Matches, occurrences...)
	}

	var hits []*internal.SearchResult
	for _, fileGroup := range fileGroups {
		sort.Slice(fileGroup.Matches, func(i, j int) bool {
			return fileGroup.Matches[i].Start < fileGroup.Matches[j].Start
		})

		hits = append(hits, &internal.SearchResult{
			RepoID:      fileGroup.RepoID,
			Filename:    fileGroup.Filename,
			CommitID:    fileGroup.CommitID,
			Content:     fileGroup.Content,
			UpdatedUnix: fileGroup.UpdatedUnix,
			Language:    fileGroup.Language,
			Color:       fileGroup.Color,
			Matches:     fileGroup.Matches,
		})
	}

	if len(hits) > pageSize {
		hits = hits[:pageSize]
	}

	return int64(len(hits)), hits, extractAggs(searchResult), nil
}

type FileMatches struct {
	RepoID      int64
	Filename    string
	CommitID    string
	Content     string
	UpdatedUnix timeutil.TimeStamp
	Language    string
	Color       string
	Matches     []internal.Match
}

func safeString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func extractAggs(searchResult *meilisearch.SearchResponse) []*internal.SearchResultLanguages {
	var searchResultLanguages []*internal.SearchResultLanguages

	if searchResult.FacetDistribution == nil {
		return searchResultLanguages
	}

	facetsMap, ok := searchResult.FacetDistribution.(map[string]any)
	if !ok {
		return searchResultLanguages
	}

	langFacetRaw, ok := facetsMap["language"]
	if !ok {
		return searchResultLanguages
	}

	langFacet, ok := langFacetRaw.(map[string]any)
	if !ok {
		return searchResultLanguages
	}

	searchResultLanguages = make([]*internal.SearchResultLanguages, 0, len(langFacet))

	for lang, countRaw := range langFacet {
		countFloat, ok := countRaw.(float64)
		if !ok {
			continue
		}

		searchResultLanguages = append(searchResultLanguages, &internal.SearchResultLanguages{
			Language: lang,
			Color:    enry.GetColor(lang),
			Count:    int(countFloat),
		})
	}

	return searchResultLanguages
}

func (b *Indexer) Search(ctx context.Context, opts *internal.SearchOptions) (int64, []*internal.SearchResult, []*internal.SearchResultLanguages, error) {
	start, pageSize := opts.GetSkipTake()

	var filters []string

	if len(opts.RepoIDs) > 0 {
		f := inner_meilisearch.NewFilterIn("repo_id", opts.RepoIDs...)
		filters = append(filters, string(f))
	}

	if opts.Filename != "" {
		filters = append(filters, fmt.Sprintf("filename.tree = \"%s\"", opts.Filename))
	}

	if opts.Language != "" {
		filters = append(filters, fmt.Sprintf("language = \"%s\"", opts.Language))
	}

	searchRequest := &meilisearch.SearchRequest{
		Limit:                 int64(pageSize),
		Offset:                int64(start),
		Filter:                filters,
		ShowMatchesPosition:   true,
		AttributesToHighlight: []string{"content"},
		Facets:                []string{"language"},
	}

	searchResult, err := b.inner.Client.Index(b.inner.VersionedIndexName()).Search(opts.Keyword, searchRequest)
	if err != nil {
		return 0, nil, nil, err
	}

	total, hits, langs, err := convertResult(searchResult, pageSize)
	if err != nil {
		return 0, nil, nil, err
	}

	return total, hits, langs, nil
}

func (b *Indexer) checkError(err error) error {
	log.Error("Meilisearch error: %v", err)
	return err
}
