// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package migrations

import (
	"bytes"
	"context"
	"fmt"
	"forgejo.org/modules/proxy"
	"forgejo.org/modules/util"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	user_model "forgejo.org/models/user"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	base "forgejo.org/modules/migration"
	"forgejo.org/modules/structs"

	"github.com/PuerkitoBio/goquery"
)

var (
	_ base.Downloader        = &PagureDownloader{}
	_ base.DownloaderFactory = &PagureDownloaderFactory{}
)

func init() {
	RegisterDownloaderFactory(&PagureDownloaderFactory{})
}

// PagureDownloaderFactory defines a downloader factory
type PagureDownloaderFactory struct{}

// PagureUser defines a user on Pagure to be migrated over to Forgejo
type PagureUser struct {
	FullURL  string `json:"full_url"`
	Fullname string `json:"fullname"`
	Name     string `json:"name"`
	URLPath  string `json:"url_path"`
}

// PagureRepoInfo describes the repository with preliminary information
type PagureRepoInfo struct {
	ID            int64             `json:"id"`
	Name          string            `json:"name"`
	FullName      string            `json:"fullname"`
	Description   string            `json:"description"`
	Topics        []string          `json:"tags"`
	CloseStatuses []string          `json:"close_status"`
	Priorities    map[string]string `json:"priorities"`
	Milestones    map[string]struct {
		Active bool    `json:"active"`
		Date   *string `json:"date"`
	} `json:"milestones"`
}

// PagureDownloader implements a Downloader interface to get repository information from Pagure
type PagureDownloader struct {
	base.NullDownloader
	ctx                   context.Context
	client                *http.Client
	baseURL               *url.URL
	meta                  PagureRepoInfo
	repoName              string
	token                 string
	privateIssuesOnlyRepo bool
	repoID                int64
	maxIssueIndex         int64
	userMap               map[string]*PagureUser
	milestoneMap          map[int64]string
	priorities            map[string]string
}

// PagureLabelsList defines a list of labels under an issue tracker
type PagureLabelsList struct {
	Labels []string `json:"tags"`
}

// PagureLabel defines a label under the issue tracker labels list
type PagureLabel struct {
	Label            string `json:"tag"`
	LabelColor       string `json:"tag_color"`
	LabelDescription string `json:"tag_description"`
}

// PagureIssueContext confirms if a said unit is an issue ticket or a pull request
type PagureIssueContext struct {
	IsPullRequest bool
}

// PagureIssuesResponse describes a list of issue tickets under an issue tracker
type PagureIssuesResponse struct {
	Issues []struct {
		Assignee     interface{}   `json:"assignee"`
		Blocks       []string      `json:"blocks"`
		CloseStatus  string        `json:"close_status"`
		ClosedAt     string        `json:"closed_at"`
		ClosedBy     interface{}   `json:"closed_by"`
		Comments     []interface{} `json:"comments"`
		Content      string        `json:"content"`
		CustomFields []interface{} `json:"custom_fields"`
		DateCreated  string        `json:"date_created"`
		Depends      []interface{} `json:"depends"`
		ID           int64         `json:"id"`
		LastUpdated  string        `json:"last_updated"`
		Milestone    string        `json:"milestone"`
		Priority     int64         `json:"priority"`
		Private      bool          `json:"private"`
		Status       string        `json:"status"`
		Tags         []string      `json:"tags"`
		Title        string        `json:"title"`
		User         PagureUser    `json:"user"`
	} `json:"issues"`
	Pagination struct {
		First   string  `json:"first"`
		Last    string  `json:"last"`
		Next    *string `json:"next"`
		Page    int     `json:"page"`
		Pages   int     `json:"pages"`
		PerPage int     `json:"per_page"`
		Prev    string  `json:"prev"`
	}
}

// PagureIssueDetail describes a list of issue comments under an issue ticket
type PagureIssueDetail struct {
	Comment      string     `json:"comment"`
	DateCreated  string     `json:"date_created"`
	ID           int64      `json:"id"`
	Notification bool       `json:"notification"`
	User         PagureUser `json:"user"`
}

// PagurePRRresponse describes a list of pull requests under an issue tracker
type PagurePRResponse struct {
	Pagination struct {
		Next *string `json:"next"`
	} `json:"pagination"`
	Requests []struct {
		Branch         string     `json:"branch"`
		BranchFrom     string     `json:"branch_from"`
		CommitStop     string     `json:"commit_stop"`
		DateCreated    string     `json:"date_created"`
		FullURL        string     `json:"full_url"`
		ID             int        `json:"id"`
		InitialComment string     `json:"initial_comment"`
		LastUpdated    string     `json:"last_updated"`
		Status         string     `json:"status"`
		Tags           []string   `json:"tags"`
		Title          string     `json:"title"`
		User           PagureUser `json:"user"`
		ClosedAt       string     `json:"closed_at"`
		ClosedBy       PagureUser `json:"closed_by"`
		RepoFrom       struct {
			FullURL string `json:"full_url"`
		} `json:"repo_from"`
	} `json:"requests"`
}

// processDate converts epoch time string to Go formatted time
func processDate(dateStr *string) time.Time {
	date := time.Time{}
	if dateStr == nil {
		return date
	}

	unix, err := strconv.Atoi(*dateStr)
	if err != nil {
		fmt.Println("Error:", err)
		return date
	}

	date = time.Unix(int64(unix), 0)
	return date
}

// New returns a downloader related to this factory according MigrateOptions
func (f *PagureDownloaderFactory) New(ctx context.Context, opts base.MigrateOptions) (base.Downloader, error) {
	u, err := url.Parse(opts.CloneAddr)
	if err != nil {
		return nil, err
	}

	repoName := ""

	fields := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(fields) == 2 {
		repoName = fields[0] + "/" + strings.TrimSuffix(fields[1], ".git")
	} else if len(fields) == 1 {
		repoName = strings.TrimSuffix(fields[0], ".git")
	} else {
		return nil, fmt.Errorf("invalid path: %s", u.Path)
	}

	u.Path, u.Fragment = "", ""
	log.Info("Create Pagure downloader. BaseURL: %v RepoName: %s", u, repoName)

	return NewPagureDownloader(ctx, u, opts.AuthToken, repoName), nil
}

// GitServiceType returns the type of Git service
func (f *PagureDownloaderFactory) GitServiceType() structs.GitServiceType {
	return structs.PagureService
}

// SetContext sets context
func (d *PagureDownloader) SetContext(ctx context.Context) {
	d.ctx = ctx
}

// NewPagureDownloader creates a new downloader object
func NewPagureDownloader(ctx context.Context, baseURL *url.URL, token, repoName string) *PagureDownloader {
	privateIssuesOnlyRepo := false
	if token != "" {
		privateIssuesOnlyRepo = true
	}

	downloader := &PagureDownloader{
		ctx:      ctx,
		baseURL:  baseURL,
		repoName: repoName,
		client: &http.Client{
			Transport: &http.Transport{
				Proxy: proxy.Proxy(),
			},
		},
		token:                 token,
		privateIssuesOnlyRepo: privateIssuesOnlyRepo,
		userMap:               make(map[string]*PagureUser),
		milestoneMap:          make(map[int64]string),
		priorities:            make(map[string]string),
	}

	return downloader
}

// String sets the default text for information purposes
func (d *PagureDownloader) String() string {
	return fmt.Sprintf("migration from Pagure server %s [%d]/%s", d.baseURL, d.repoID, d.repoName)
}

// LogString sets the default text for logging purposes
func (d *PagureDownloader) LogString() string {
	if d == nil {
		return "<PagureDownloader nil>"
	}
	return fmt.Sprintf("<PagureDownloader %s [%d]/%s>", d.baseURL, d.repoID, d.repoName)
}

// callAPI handles all the requests made against Pagure
func (d *PagureDownloader) callAPI(endpoint string, parameter map[string]string, result any) error {
	u, err := d.baseURL.Parse(endpoint)
	if err != nil {
		return err
	}

	if parameter != nil {
		query := u.Query()
		for k, v := range parameter {
			query.Set(k, v)
		}
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(d.ctx, "GET", u.String(), nil)
	if err != nil {
		return err
	}
	if d.privateIssuesOnlyRepo {
		req.Header.Set("Authorization", "token "+d.token)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	return decoder.Decode(&result)
}

// GetRepoInfo returns repository information from Pagure
func (d *PagureDownloader) GetRepoInfo() (*base.Repository, error) {
	err := d.callAPI("/api/0/"+d.repoName, nil, &d.meta)
	if err != nil {
		return nil, err
	}

	d.repoID, d.priorities = d.meta.ID, d.meta.Priorities

	cloneURL, err := d.baseURL.Parse(d.meta.FullName)
	if err != nil {
		return nil, err
	}
	originalURL, err := d.baseURL.Parse(d.meta.FullName)
	if err != nil {
		return nil, err
	}

	return &base.Repository{
		Name:        d.meta.Name,
		Description: d.meta.Description,
		CloneURL:    cloneURL.String() + ".git",
		OriginalURL: originalURL.String(),
	}, nil
}

// GetMilestones returns milestones information from Pagure
func (d *PagureDownloader) GetMilestones() ([]*base.Milestone, error) {
	milestones := make([]*base.Milestone, 0, len(d.meta.Milestones))
	for name, details := range d.meta.Milestones {
		state := "closed"
		if details.Active {
			state = "open"
		}

		deadline := processDate(details.Date)
		milestones = append(milestones, &base.Milestone{
			Title:       name,
			Description: "",
			Deadline:    &deadline,
			State:       state,
		})
	}

	return milestones, nil
}

// GetLabels returns labels information from Pagure
func (d *PagureDownloader) GetLabels() ([]*base.Label, error) {
	rawLabels := PagureLabelsList{}

	err := d.callAPI("/api/0/"+d.repoName+"/tags", nil, &rawLabels)
	if err != nil {
		return nil, err
	}

	labels := make([]*base.Label, 0, len(rawLabels.Labels)+len(d.meta.CloseStatuses))

	for _, label := range rawLabels.Labels {
		rawLabel := PagureLabel{}
		err = d.callAPI("/api/0/"+d.repoName+"/tag/"+label, nil, &rawLabel)
		if err != nil {
			return nil, err
		}
		labels = append(labels, &base.Label{
			Name:        label,
			Description: rawLabel.LabelDescription,
			Color:       strings.TrimPrefix(rawLabel.LabelColor, "#"),
		})
	}

	for _, close_status := range d.meta.CloseStatuses {
		labels = append(labels, &base.Label{
			Name:        "Closed As/" + close_status,
			Description: "Closed with the reason of " + close_status,
			Color:       "FF0000",
			Exclusive:   true,
		})
	}

	for _, value := range d.priorities {
		if value != "" {
			labels = append(labels, &base.Label{
				Name:        "Priority/" + value,
				Description: "Priority of " + value,
				Color:       "FF00FF",
				Exclusive:   true,
			})
		}
	}

	return labels, nil
}

// GetIssues returns issue tickets from Pagure
func (d *PagureDownloader) GetIssues(page, perPage int) ([]*base.Issue, bool, error) {
	rawIssues := PagureIssuesResponse{}

	err := d.callAPI(
		"/api/0/"+d.repoName+"/issues",
		map[string]string{
			"page":     strconv.Itoa(page),
			"per_page": strconv.Itoa(perPage),
			"status":   "all",
		},
		&rawIssues,
	)
	if err != nil {
		return nil, false, err
	}

	issues := make([]*base.Issue, 0, len(rawIssues.Issues))
	for _, issue := range rawIssues.Issues {
		log.Debug("Processing issue %d", issue.ID)
		if d.privateIssuesOnlyRepo && !issue.Private {
			log.Info("Skipping issue %d because it is not private and we are only downloading private issues", issue.ID)
			continue
		}
		labels := []*base.Label{}
		for _, tag := range issue.Tags {
			labels = append(labels, &base.Label{Name: tag})
		}

		if issue.CloseStatus != "" {
			labels = append(labels, &base.Label{Name: "Closed As/" + issue.CloseStatus})
		}

		priorityStr := ""
		if issue.Priority != 0 {
			priorityStr = strconv.FormatInt(issue.Priority, 10)
		}

		if priorityStr != "" {
			priorityValue, ok := d.priorities[priorityStr]
			if ok {
				labels = append(labels, &base.Label{Name: "Priority/" + priorityValue})
			}
		}
		poster := d.tryGetUser(issue.User.Name)
		log.Info("Adding issue: %d", issue.ID)

		closedat := processDate(&issue.ClosedAt)

		issues = append(issues, &base.Issue{
			Title:        issue.Title,
			Number:       issue.ID,
			PosterName:   poster.Name,
			PosterID:     poster.ID,
			PosterEmail:  poster.Email,
			Content:      issue.Content,
			Milestone:    issue.Milestone,
			State:        strings.ToLower(issue.Status),
			Created:      processDate(&issue.DateCreated),
			Updated:      processDate(&issue.LastUpdated),
			Closed:       &closedat,
			Labels:       labels,
			ForeignIndex: issue.ID,
			Context:      PagureIssueContext{IsPullRequest: false},
		})

		if d.maxIssueIndex < issue.ID {
			d.maxIssueIndex = issue.ID
		}
	}
	hasNext := rawIssues.Pagination.Next == nil

	return issues, hasNext, nil
}

// GetComments returns issue comments from Pagure
func (d *PagureDownloader) GetComments(commentable base.Commentable) ([]*base.Comment, bool, error) {
	context, ok := commentable.GetContext().(PagureIssueContext)
	if !ok {
		return nil, false, fmt.Errorf("unexpected context: %+v", commentable.GetContext())
	}

	list := struct {
		Comments []PagureIssueDetail `json:"comments"`
	}{}
	endpoint := ""

	if context.IsPullRequest {
		endpoint = fmt.Sprintf("/api/0/%s/pull-request/%d", d.repoName, commentable.GetForeignIndex())
	} else {
		endpoint = fmt.Sprintf("/api/0/%s/issue/%d", d.repoName, commentable.GetForeignIndex())
	}

	err := d.callAPI(endpoint, nil, &list)
	if err != nil {
		log.Error("Error calling API: %v", err)
		return nil, false, err
	}

	comments := make([]*base.Comment, 0, len(list.Comments))
	for _, unit := range list.Comments {
		if len(unit.Comment) == 0 {
			log.Error("Empty comment")
			continue
		}
		poster := d.tryGetUser(unit.User.Name)

		log.Info("Adding comment: %d", unit.ID)
		c := &base.Comment{
			IssueIndex:  commentable.GetLocalIndex(),
			Index:       unit.ID,
			PosterID:    poster.ID,
			PosterName:  poster.Name,
			PosterEmail: poster.Email,
			Content:     unit.Comment,
			Created:     processDate(&unit.DateCreated),
		}
		comments = append(comments, c)
	}

	return comments, true, nil
}

// GetPullRequests returns pull requests from Pagure
func (d *PagureDownloader) GetPullRequests(page, perPage int) ([]*base.PullRequest, bool, error) {
	// Could not figure out how to disable this in opts, so if a private issues only repo,
	// We just return an empty list
	if d.privateIssuesOnlyRepo {
		pullRequests := make([]*base.PullRequest, 0, 0)
		return pullRequests, true, nil
	}

	rawPullRequests := PagurePRResponse{}
	err := d.callAPI(
		"/api/0/"+d.repoName+"/pull-requests",
		map[string]string{
			"page":     strconv.Itoa(page),
			"per_page": strconv.Itoa(perPage),
			"status":   "all",
		},
		&rawPullRequests,
	)
	if err != nil {
		return nil, false, err
	}

	pullRequests := make([]*base.PullRequest, 0, len(rawPullRequests.Requests))

	for _, pr := range rawPullRequests.Requests {
		poster := d.tryGetUser(pr.User.Name)
		state, merged, baseSHA := "", false, ""
		labels := []*base.Label{}

		for _, tag := range pr.Tags {
			labels = append(labels, &base.Label{Name: tag})
		}
		mergedtime := processDate(&pr.ClosedAt)

		if strings.ToLower(pr.Status) == "merged" {
			state, merged = "closed", true
			// If it is a merged PR, we need to get the earliest commit (CommitStop), and load the Pagure HTML
			// page for that commit, and get the parent commit for it. The Pagure API does not do this.
			fullURL := d.baseURL.ResolveReference(&url.URL{Path: d.repoName + "/c/" + pr.CommitStop, RawQuery: "branch=" + pr.Branch})
			req, err := http.NewRequestWithContext(d.ctx, "GET", fullURL.String(), nil)
			if err != nil {
				log.Error("Error creating request: %v", err)
				return nil, false, err
			}

			resp, err := d.client.Do(req)
			if err != nil {
				log.Error("Error making request: %v", err)
				return nil, false, err
			}
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Error("Error reading response body: %v", err)
				return nil, false, err
			}
			doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyBytes))
			if err != nil {
				log.Error("Error reading response body: %v", err)
				return nil, false, err
			}

			doc.Find(".ml-auto .btn-group a.btn.btn-outline-primary.btn-sm").Each(func(i int, s *goquery.Selection) {
				// For each item found, get the title
				if s.Text() == "parent" {
					// Get the title attribute
					title, exists := s.Attr("title")
					if exists {
						baseSHA = title
					}
				}
			})

		} else if util.ASCIIEqualFold(pr.Status, "open") {
			state, merged, baseSHA = "open", false, ""
		} else {
			state, merged, baseSHA = "closed", false, ""
		}

		pullRequests = append(pullRequests, &base.PullRequest{
			Title:       pr.Title,
			Number:      int64(pr.ID),
			PosterName:  poster.Name,
			PosterID:    poster.ID,
			PosterEmail: poster.Email,
			Content:     pr.InitialComment,
			State:       state,
			Created:     processDate(&pr.DateCreated),
			Updated:     processDate(&pr.LastUpdated),
			MergedTime:  &mergedtime,
			Closed:      &mergedtime,
			Merged:      merged,
			Labels:      labels,
			Head: base.PullRequestBranch{
				Ref:      pr.BranchFrom,
				SHA:      pr.CommitStop,
				RepoName: d.repoName,
				CloneURL: pr.RepoFrom.FullURL + ".git",
			},
			Base: base.PullRequestBranch{
				Ref:      pr.Branch,
				SHA:      baseSHA,
				RepoName: d.repoName,
			},
			ForeignIndex: int64(pr.ID),
			PatchURL:     pr.FullURL + ".patch",
			Context:      PagureIssueContext{IsPullRequest: true},
		})

		// SECURITY: Ensure that the PR is safe
		_ = CheckAndEnsureSafePR(pullRequests[len(pullRequests)-1], d.baseURL.String(), d)
	}

	hasNext := rawPullRequests.Pagination.Next == nil

	return pullRequests, hasNext, nil
}

// GetTopics return repository topics from Pagure
func (d *PagureDownloader) GetTopics() ([]string, error) {
	info := PagureRepoInfo{}
	err := d.callAPI(
		"/api/0/"+d.repoName,
		nil,
		&info,
	)
	if err != nil {
		return nil, err
	}
	return info.Topics, nil
}

func (d *PagureDownloader) tryGetUser(name string) *user_model.User {
	user, ok := d.userMap[name]

	temp := struct {
		User PagureUser `json:"user"`
	}{}

	if !ok {
		err := d.callAPI(
			fmt.Sprintf("/api/0/user/%s", name),
			nil,
			&temp,
		)
		if err != nil {
			user = &PagureUser{
				Name:     name,
				Fullname: name,
			}
		} else {
			user = &temp.User
		}
		d.userMap[name] = user
	}

	fuser, err := user_model.GetUserByName(d.ctx, user.Name)
	if err != nil {
		u := &user_model.User{
			Name:        user.Name,
			Email:       user.Name + "@fedoraproject.org",
			LoginName:   user.Name,
			LoginSource: 1, //FAS HARDCODED PROLLY BAD
			FullName:    user.Fullname,
		}
		fmt.Println("IDHAR ", *u, " HAI")
		err := user_model.CreateUser(d.ctx, u)
		if err != nil {
			log.Error("Error creating user: `%s` - %v", u.Name, err)
			return nil
		}
		newuser, err := user_model.GetUserByName(d.ctx, user.Name)
		if err != nil {
			log.Error("Error getting user: %v", err)
			return nil
		}
		return newuser
	} else {
		return fuser
	}
}
