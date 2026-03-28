package edu

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"regexp"
	"time"

	"forgejo.org/modules/log"
)

var validUsernameRe = regexp.MustCompile(`^[\da-zA-Z][-.\w]*$`)

func generatePassword() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate password: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *service) UploadCSV(ctx context.Context, courseID, creatorID int64, data []byte, mapping CSVColumnMapping) (*ImportDraft, error) {
	rows, err := ParseCSV(data, mapping)
	if err != nil {
		return nil, fmt.Errorf("parse CSV: %w", err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("no valid rows found in CSV")
	}

	now := time.Now().Unix()
	draft := &ImportDraft{
		CourseID:    courseID,
		CreatorID:   creatorID,
		Status:      StatusDraft,
		RawCSV:      string(data),
		CreatedUnix: now,
	}

	if err := s.repo.CreateImportDraft(ctx, draft); err != nil {
		return nil, fmt.Errorf("create import draft: %w", err)
	}

	draftRows := make([]*ImportDraftRow, 0, len(rows))
	for _, r := range rows {
		draftRows = append(draftRows, &ImportDraftRow{
			DraftID:     draft.ID,
			FullName:    r.FullName,
			Email:       r.Email,
			Group:       r.Group,
			Username:    r.Username,
			Role:        "student",
			Status:      StatusPending,
			CreatedUnix: now,
		})
	}

	if err := s.repo.CreateImportDraftRows(ctx, draftRows); err != nil {
		return nil, fmt.Errorf("create import draft rows: %w", err)
	}

	return draft, nil
}

func (s *service) GetImportDraft(ctx context.Context, id int64) (*ImportDraft, []*ImportDraftRow, error) {
	draft, err := s.repo.GetImportDraft(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("get import draft: %w", err)
	}
	if draft == nil {
		return nil, nil, nil
	}

	rows, err := s.repo.GetImportDraftRows(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("get import draft rows: %w", err)
	}

	return draft, rows, nil
}

func (s *service) UpdateDraftRow(ctx context.Context, rowID int64, username, email string) error {
	row := &ImportDraftRow{
		ID:       rowID,
		Username: username,
		Email:    email,
		Status:   StatusPending,
	}
	return s.repo.UpdateImportDraftRow(ctx, row)
}

func (s *service) ExecuteImport(ctx context.Context, draftID int64, doerID int64, defaultRole RoleType) (*ImportResult, error) {
	if s.users == nil {
		return nil, fmt.Errorf("user creator not configured")
	}

	draft, err := s.repo.GetImportDraft(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("get draft: %w", err)
	}
	if draft == nil {
		return nil, fmt.Errorf("draft not found")
	}

	rows, err := s.repo.GetImportDraftRows(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("get draft rows: %w", err)
	}

	result := &ImportResult{
		TotalRows: len(rows),
	}

	for _, row := range rows {
		if row.Status != StatusPending {
			continue
		}

		if !validUsernameRe.MatchString(row.Username) {
			row.Status = StatusError
			row.ErrorMsg = "invalid username format"
			if errUpd := s.repo.UpdateImportDraftRow(ctx, row); errUpd != nil {
				log.Error("Failed to update import draft row: %v", errUpd)
			}
			result.Errors++
			continue
		}

		// Check if user already exists
		existingUser, err := s.users.GetUserByName(ctx, row.Username)
		if err == nil && existingUser != nil {
			// User exists — just enroll
			enrollment := &CourseEnrollment{
				CourseID:    draft.CourseID,
				UserID:      existingUser.ID,
				Role:        defaultRole,
				CreatedUnix: time.Now().Unix(),
			}
			if err := s.repo.EnrollUser(ctx, enrollment); err != nil {
				row.Status = StatusError
				row.ErrorMsg = fmt.Sprintf("enroll existing user: %v", err)
				if errUpd := s.repo.UpdateImportDraftRow(ctx, row); errUpd != nil {
					log.Error("Failed to update import draft row: %v", errUpd)
				}
				result.Errors++
				continue
			}
			row.Status = "exists"
			if errUpd := s.repo.UpdateImportDraftRow(ctx, row); errUpd != nil {
				log.Error("Failed to update import draft row: %v", errUpd)
			}
			result.AlreadyExist++
			continue
		}

		// Generate email if empty
		email := row.Email
		if email == "" {
			email = row.Username + "@edu.local"
		}

		// Generate password
		password, err := generatePassword()
		if err != nil {
			row.Status = StatusError
			row.ErrorMsg = fmt.Sprintf("generate password: %v", err)
			if errUpd := s.repo.UpdateImportDraftRow(ctx, row); errUpd != nil {
				log.Error("Failed to update import draft row: %v", errUpd)
			}
			result.Errors++
			continue
		}

		// Create user
		if err := s.users.CreateUser(ctx, row.Username, email, password, row.FullName); err != nil {
			row.Status = StatusError
			row.ErrorMsg = fmt.Sprintf("create user: %v", err)
			if errUpd := s.repo.UpdateImportDraftRow(ctx, row); errUpd != nil {
				log.Error("Failed to update import draft row: %v", errUpd)
			}
			result.Errors++
			continue
		}

		// Get the created user for enrollment
		newUser, err := s.users.GetUserByName(ctx, row.Username)
		if err != nil {
			row.Status = StatusError
			row.ErrorMsg = fmt.Sprintf("get created user: %v", err)
			if errUpd := s.repo.UpdateImportDraftRow(ctx, row); errUpd != nil {
				log.Error("Failed to update import draft row: %v", errUpd)
			}
			result.Errors++
			continue
		}

		// Enroll in course
		enrollment := &CourseEnrollment{
			CourseID:    draft.CourseID,
			UserID:      newUser.ID,
			Role:        defaultRole,
			CreatedUnix: time.Now().Unix(),
		}
		if err := s.repo.EnrollUser(ctx, enrollment); err != nil {
			row.Status = StatusError
			row.ErrorMsg = fmt.Sprintf("enroll new user: %v", err)
			if errUpd := s.repo.UpdateImportDraftRow(ctx, row); errUpd != nil {
				log.Error("Failed to update import draft row: %v", errUpd)
			}
			result.Errors++
			continue
		}

		row.Status = "created"
		if errUpd := s.repo.UpdateImportDraftRow(ctx, row); errUpd != nil {
			log.Error("Failed to update import draft row: %v", errUpd)
		}
		result.Created++
		result.Credentials = append(result.Credentials, UserCredential{
			Username: row.Username,
			Password: password,
			FullName: row.FullName,
			Email:    email,
		})
	}

	draft.Status = StatusDone
	if err := s.repo.UpdateImportDraft(ctx, draft); err != nil {
		log.Error("Failed to update import draft: %v", err)
	}

	return result, nil
}

func (s *service) DeleteImportDraft(ctx context.Context, id int64) error {
	return s.repo.DeleteImportDraft(ctx, id)
}
