// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package secret

import (
	"context"
	"crypto/subtle"
	"errors"
	"regexp"
	"strings"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unit"
	"forgejo.org/modules/keying"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/timeutil"

	"xorm.io/builder"
)

const (
	PairUsernameName = "ZOT_VM_ARTIFACT_USERNAME"
	PairPasswordName = "ZOT_VM_ARTIFACT_PASSWORD"
	PairRequestSchema = "creator-signal.actions-secret-pair-operation/v1"
	PairResultSchema = "creator-signal.actions-secret-pair-operation-result/v1"
	PairPurpose = "vm-artifact-publisher"
	PairActive = "Active"
	PairRevoked = "Revoked"
)

var (
	ErrManagedSecret = errors.New("managed Actions secret mutation denied")
	ErrPairConflict = errors.New("Actions secret pair operation conflict")
	ErrPairInvalid = errors.New("invalid Actions secret pair operation")
	ErrPairDisabled = errors.New("Actions secret pair unavailable")
	pairIdentityPattern = regexp.MustCompile(`\A[0-9a-f]{64}\z`)
	pairPasswordPattern = regexp.MustCompile(`\Acs-vm-artifact-[A-Za-z0-9_-]{43}\z`)
)

// ActionSecretPairOperation is private provider state, not Source qualification.
// No foreign key cascades: repository deletion retains original RepoID tombstones.
// Only ApplySecretPairOperation may create it; only repository deletion may revoke.
type ActionSecretPairOperation struct {
	ID             int64
	OperationID    string `xorm:"UNIQUE NOT NULL VARCHAR(64)"`
	RepoID         int64  `xorm:"UNIQUE(repo_purpose) NOT NULL"`
	Purpose        string `xorm:"UNIQUE(repo_purpose) NOT NULL VARCHAR(32)"`
	OwnershipID    string `xorm:"NOT NULL VARCHAR(64)"`
	TransactionID  string `xorm:"NOT NULL VARCHAR(64)"`
	Nonce          string `xorm:"NOT NULL VARCHAR(64)"`
	UsernameID     int64
	PasswordID     int64
	UsernameData   []byte `xorm:"BLOB"`
	PasswordData   []byte `xorm:"BLOB"`
	State          string `xorm:"NOT NULL VARCHAR(16)"`
	CreatedUnix    timeutil.TimeStamp `xorm:"created NOT NULL"`
}

func init() {
	db.RegisterModel(new(ActionSecretPairOperation))
}

func IsManagedSecretName(name string) bool {
	name = strings.ToUpper(name)
	return name == PairUsernameName || name == PairPasswordName
}

func ValidateSecretPairRequest(request *api.ActionSecretPairRequest) error {
	if request == nil || request.Schema != PairRequestSchema || request.Purpose != PairPurpose ||
		!pairIdentityPattern.MatchString(request.OperationID) ||
		!pairIdentityPattern.MatchString(request.Binding.OwnershipID) ||
		!pairIdentityPattern.MatchString(request.Binding.TransactionID) ||
		!pairIdentityPattern.MatchString(request.Binding.Nonce) ||
		request.Secrets.Username != "vm-image-publisher" || !pairPasswordPattern.MatchString(request.Secrets.Password) {
		return ErrPairInvalid
	}
	return nil
}

// lockPairRepository serializes pair creation, projection and repository deletion.
// SQLite uses its transactional read snapshot/write-upgrade conflict rather than
// a no-op UPDATE, so identical replay never writes. A conflicting write denies.
func lockPairRepository(ctx context.Context, repoID int64) (*repo_model.Repository, error) {
	if !db.InTransaction(ctx) || repoID <= 0 {
		return nil, ErrPairConflict
	}
	repository := new(repo_model.Repository)
	query := db.GetEngine(ctx).ID(repoID)
	if !setting.Database.Type.IsSQLite3() {
		query = query.ForUpdate()
	}
	found, err := query.Get(repository)
	if err != nil || !found {
		return nil, ErrPairConflict
	}
	return repository, nil
}

func pairActionsAvailable(ctx context.Context, repository *repo_model.Repository) bool {
	if !setting.Actions.Enabled {
		return false
	}
	_, err := repository.GetUnit(ctx, unit.TypeActions)
	return err == nil
}

func equalPairMaterial(left, right string) bool {
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func (operation *ActionSecretPairOperation) matchesRequest(request *api.ActionSecretPairRequest) bool {
	if operation.OperationID != request.OperationID || operation.Purpose != request.Purpose ||
		operation.OwnershipID != request.Binding.OwnershipID || operation.TransactionID != request.Binding.TransactionID ||
		operation.Nonce != request.Binding.Nonce || operation.State != PairActive {
		return false
	}
	username, usernameErr := keying.ActionSecret.Decrypt(operation.UsernameData, keying.ColumnAndID("username_data", operation.ID))
	password, passwordErr := keying.ActionSecret.Decrypt(operation.PasswordData, keying.ColumnAndID("password_data", operation.ID))
	return usernameErr == nil && passwordErr == nil && equalPairMaterial(string(username), request.Secrets.Username) && equalPairMaterial(string(password), request.Secrets.Password)
}

// verifyPairRows verifies exact current row ownership and decrypted intended bytes.
func verifyPairRows(operation *ActionSecretPairOperation, rows []*Secret) error {
	if operation.State != PairActive || operation.Purpose != PairPurpose || operation.RepoID <= 0 ||
		!pairIdentityPattern.MatchString(operation.OperationID) || !pairIdentityPattern.MatchString(operation.OwnershipID) ||
		!pairIdentityPattern.MatchString(operation.TransactionID) || !pairIdentityPattern.MatchString(operation.Nonce) ||
		operation.UsernameID <= 0 || operation.PasswordID <= 0 || operation.UsernameID == operation.PasswordID || len(rows) != 2 {
		return ErrPairConflict
	}
	seen := map[string]bool{}
	for _, row := range rows {
		if seen[row.Name] {
			return ErrPairConflict
		}
		seen[row.Name] = true
		column, encrypted, expectedID := "", []byte(nil), int64(0)
		switch row.Name {
		case PairUsernameName:
			column, encrypted, expectedID = "username_data", operation.UsernameData, operation.UsernameID
		case PairPasswordName:
			column, encrypted, expectedID = "password_data", operation.PasswordData, operation.PasswordID
		default:
			return ErrPairConflict
		}
		if row.RepoID != operation.RepoID || row.OwnerID != 0 || row.ID != expectedID {
			return ErrPairConflict
		}
		intended, err := keying.ActionSecret.Decrypt(encrypted, keying.ColumnAndID(column, operation.ID))
		if err != nil {
			return ErrPairConflict
		}
		if (row.Name == PairUsernameName && string(intended) != "vm-image-publisher") ||
			(row.Name == PairPasswordName && !pairPasswordPattern.Match(intended)) { return ErrPairConflict }
		actual, err := row.GetDecryptedData()
		if err != nil || !equalPairMaterial(actual, string(intended)) {
			return ErrPairConflict
		}
	}
	return nil
}

// ValidateManagedSecretProjection closes workflow-call remapping of reserved names.
// Omitting both names is permitted; presenting either requires the complete live
// authenticated pair with exact names and material in the current transaction.
func ValidateManagedSecretProjection(ctx context.Context, ownerID, repoID int64, projected map[string]string) error {
	count := 0
	for name := range projected {
		if IsManagedSecretName(name) {
			if name != PairUsernameName && name != PairPasswordName {
				return ErrPairConflict
			}
			count++
		}
	}
	if count == 0 {
		return nil
	}
	if count != 2 {
		return ErrPairConflict
	}
	actual, err := FetchActionSecrets(ctx, ownerID, repoID)
	if err != nil {
		return ErrPairConflict
	}
	username, hasUsername := actual[PairUsernameName]
	password, hasPassword := actual[PairPasswordName]
	if !hasUsername || !hasPassword || !equalPairMaterial(username, projected[PairUsernameName]) || !equalPairMaterial(password, projected[PairPasswordName]) {
		return ErrPairConflict
	}
	return nil
}

func managedRows(rows []*Secret) []*Secret {
	pair := make([]*Secret, 0, 2)
	for _, row := range rows {
		if IsManagedSecretName(row.Name) {
			pair = append(pair, row)
		}
	}
	return pair
}

// ApplySecretPairOperation atomically creates the immutable operation and pair.
// Identical replay authenticates current rows and performs no database writes.
func ApplySecretPairOperation(ctx context.Context, repoID int64, request *api.ActionSecretPairRequest) (*api.ActionSecretPairResult, error) {
	if !setting.Actions.Enabled {
		return nil, ErrPairDisabled
	}
	if err := ValidateSecretPairRequest(request); err != nil {
		return nil, err
	}
	// Snapshot caller data before entering database operations.
	input := *request
	result := &api.ActionSecretPairResult{Schema: PairResultSchema, OperationID: input.OperationID, Binding: input.Binding, State: "Applied"}
	err := db.WithTx(ctx, func(ctx context.Context) error {
		repository, err := lockPairRepository(ctx, repoID)
		if err != nil {
			return ErrPairConflict
		}
		if !pairActionsAvailable(ctx, repository) {
			return ErrPairDisabled
		}
		operation, exists, err := db.Get[ActionSecretPairOperation](ctx, builder.Eq{"operation_id": input.OperationID})
		if err != nil {
			return ErrPairConflict
		}
		var rows []*Secret
		if err := db.GetEngine(ctx).Where(builder.Or(
			builder.Eq{"owner_id": 0, "repo_id": repoID},
			builder.Eq{"owner_id": repository.OwnerID, "repo_id": 0})).Find(&rows); err != nil {
			return ErrPairConflict
		}
		pair := managedRows(rows)
		if exists {
			if operation.RepoID != repoID || !operation.matchesRequest(&input) || verifyPairRows(operation, pair) != nil {
				return ErrPairConflict
			}
			result.Replayed = true
			return nil
		}
		if len(pair) != 0 {
			return ErrPairConflict
		}
		prior, err := db.Exist[ActionSecretPairOperation](ctx, builder.Eq{"repo_id": repoID, "purpose": PairPurpose})
		if err != nil || prior {
			return ErrPairConflict
		}
		operation = &ActionSecretPairOperation{OperationID: input.OperationID, RepoID: repoID, Purpose: PairPurpose,
			OwnershipID: input.Binding.OwnershipID, TransactionID: input.Binding.TransactionID, Nonce: input.Binding.Nonce, State: PairActive}
		if err := db.Insert(ctx, operation); err != nil {
			return ErrPairConflict
		}
		operation.UsernameData = keying.ActionSecret.Encrypt([]byte(input.Secrets.Username), keying.ColumnAndID("username_data", operation.ID))
		operation.PasswordData = keying.ActionSecret.Encrypt([]byte(input.Secrets.Password), keying.ColumnAndID("password_data", operation.ID))
		for _, material := range []struct { name, data string }{{PairUsernameName, input.Secrets.Username}, {PairPasswordName, input.Secrets.Password}} {
			row := &Secret{RepoID: repoID, Name: material.name}
			if err := db.Insert(ctx, row); err != nil {
				return ErrPairConflict
			}
			row.SetData(material.data)
			if _, err := db.GetEngine(ctx).ID(row.ID).Cols("data").Update(row); err != nil {
				return ErrPairConflict
			}
			if material.name == PairUsernameName { operation.UsernameID = row.ID } else {
				operation.PasswordID = row.ID
			}
		}
		_, err = db.GetEngine(ctx).ID(operation.ID).Cols("username_id", "password_id", "username_data", "password_data").Update(operation)
		if err != nil {
			return ErrPairConflict
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrPairDisabled) {
			return nil, ErrPairDisabled
		}
		return nil, ErrPairConflict
	}
	return result, nil
}

// RevokeSecretPairForRepositoryDeletion must run inside the original delete Tx.
// It never deletes or resets the retained operation and does not expose a route.
func RevokeSecretPairForRepositoryDeletion(ctx context.Context, repoID int64) error {
	if _, err := lockPairRepository(ctx, repoID); err != nil {
		return err
	}
	_, err := db.GetEngine(ctx).Where("repo_id = ? AND state = ?", repoID, PairActive).
		Cols("state").Update(&ActionSecretPairOperation{State: PairRevoked})
	return err
}
