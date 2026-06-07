package v2

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"forgejo.org/models/auth"
	"forgejo.org/models/db"
	"forgejo.org/models/user"
	"forgejo.org/modules/log"
	forgejo_optional "forgejo.org/modules/optional"
	user_service "forgejo.org/services/user"

	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/errors"
	"github.com/elimity-com/scim/optional"
	"github.com/elimity-com/scim/schema"
)

type userResourceHandler struct {
	sourceID int64
}

// resolveUser parses the SCIM resource ID and looks up the Forgejo user,
// verifying it belongs to the given SCIM source.
func resolveUser(r *http.Request, id string, sourceID int64) (*user.User, error) {
	userID, err := strconv.ParseInt(id, 10, 64)
	if err != nil || userID == 0 {
		return nil, errors.ScimErrorResourceNotFound(id)
	}
	u, err := user.GetUserByID(r.Context(), userID)
	if err != nil {
		if user.IsErrUserNotExist(err) {
			return nil, errors.ScimErrorResourceNotFound(id)
		}
		return nil, err
	}
	// Exclude orgs, remote users and bots
	if u.Type != user.UserTypeIndividual || u.LoginSource != sourceID {
		return nil, errors.ScimErrorResourceNotFound(id)
	}
	return u, nil
}

// toResource converts a Forgejo user to a SCIM Resource.
func toResource(u *user.User) scim.Resource {
	attrs := scim.ResourceAttributes{
		"userName": u.Name,
		"emails": []map[string]any{
			{"value": u.Email, "primary": true},
		},
		"active": u.IsActive,
	}
	if u.FullName != "" {
		attrs["displayName"] = u.FullName
	}
	res := scim.Resource{
		ID:         strconv.FormatInt(u.ID, 10),
		Attributes: attrs,
	}
	if u.LoginSource != 0 {
		res.ExternalID = optional.NewString(u.LoginName)
	}
	return res
}

// extractPrimaryEmail returns the primary email from the SCIM emails array.
func extractPrimaryEmail(attrs scim.ResourceAttributes) string {
	raw, ok := attrs["emails"]
	if !ok {
		return ""
	}
	arr, ok := raw.([]any)
	if !ok {
		return ""
	}
	for _, entry := range arr {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if primary, _ := m["primary"].(bool); primary {
			if val, ok := m["value"].(string); ok {
				return val
			}
		}
	}
	if len(arr) > 0 {
		if m, ok := arr[0].(map[string]any); ok {
			if val, ok := m["value"].(string); ok {
				return val
			}
		}
	}
	return ""
}

// syncUserAttributes updates an existing user's mutable fields to match the
// values provided by a SCIM request.
func syncUserAttributes(ctx context.Context, u *user.User, userName, fullName, email string, isActive bool) error {
	if userName != u.Name {
		if err := user_service.AdminRenameUser(ctx, u, userName); err != nil {
			return fmt.Errorf("rename: %w", err)
		}
	}

	if fullName != u.FullName {
		u.FullName = fullName
		if err := user.UpdateUserCols(ctx, u, "full_name"); err != nil {
			return fmt.Errorf("full_name: %w", err)
		}
	}

	if email != u.Email {
		if err := user_service.AdminAddOrSetPrimaryEmailAddress(ctx, u, email); err != nil {
			return fmt.Errorf("email: %w", err)
		}
	}

	if u.IsActive != isActive || u.ProhibitLogin == isActive {
		u.IsActive = isActive
		u.ProhibitLogin = !isActive
		if err := user.UpdateUserCols(ctx, u, "is_active", "prohibit_login"); err != nil {
			return fmt.Errorf("active: %w", err)
		}
	}

	return nil
}

func (h userResourceHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	userName, _ := attributes["userName"].(string)
	if userName == "" {
		return scim.Resource{}, errors.ScimErrorBadRequest("attr `userName` missing")
	}

	fullName, _ := attributes["displayName"].(string)

	email := extractPrimaryEmail(attributes)
	if email == "" {
		return scim.Resource{}, errors.ScimErrorBadRequest("primary email missing")
	}

	externalID, _ := attributes[schema.CommonAttributeExternalID].(string)
	if externalID == "" {
		return scim.Resource{}, errors.ScimErrorBadRequest(schema.CommonAttributeExternalID + " is required to link the SCIM user to an OAuth2 login")
	}

	active := true
	if v, ok := attributes["active"].(bool); ok {
		active = v
	}

	// If a user with this externalID already exists, do a sync instead.
	existing := &user.User{}
	has, err := db.GetEngine(r.Context()).
		Where("login_type = ? AND login_source = ? AND login_name = ?", auth.OAuth2, h.sourceID, externalID).
		Get(existing)
	if err != nil {
		log.Error("SCIM: failed to lookup user by externalID: %v", err)
		return scim.Resource{}, errors.ScimErrorInternal
	}
	if has {
		if fullName == "" {
			fullName = existing.FullName
		}
		if err := syncUserAttributes(r.Context(), existing, userName, fullName, email, active); err != nil {
			log.Error("SCIM: failed to sync existing user attributes: %v", err)
			return scim.Resource{}, errors.ScimErrorInternal
		}
		// return http 200 instead of 201
		if flag, ok := r.Context().Value(upsertMarkerKey{}).(*bool); ok {
			*flag = true
		}
		return toResource(existing), nil
	}

	u := &user.User{
		Name:          userName,
		FullName:      fullName,
		Email:         email,
		LoginType:     auth.OAuth2,
		LoginSource:   h.sourceID,
		LoginName:     externalID,
		IsActive:      active,
		ProhibitLogin: !active,
	}

	if err := user.CreateUser(r.Context(), u); err != nil {
		log.Error("SCIM: failed to create user: %v", err)
		return scim.Resource{}, errors.ScimErrorInternal
	}

	return toResource(u), nil
}

func (h userResourceHandler) Get(r *http.Request, id string) (scim.Resource, error) {
	u, err := resolveUser(r, id, h.sourceID)
	if err != nil {
		return scim.Resource{}, err
	}
	return toResource(u), nil
}

func (h userResourceHandler) GetAll(r *http.Request, params scim.ListRequestParams) (scim.Page, error) {
	ctx := r.Context()

	opts := &user.SearchUserOptions{
		SourceID: forgejo_optional.Some[int64](h.sourceID),
		Actor:    &user.User{IsAdmin: true},
	}

	// count=0 means "return totalResults only, no resources" (RFC 7644 §3.4.2.4).
	if params.Count == 0 {
		opts.ListOptions = db.ListOptions{PageSize: 1, Page: 1}
	} else {
		opts.ListOptions = db.ListOptions{
			Page:     (params.StartIndex-1)/params.Count + 1,
			PageSize: params.Count,
		}
	}

	users, total, err := user.SearchUsers(ctx, opts)
	if err != nil {
		log.Error("SCIM: failed to list users: %v", err)
		return scim.Page{}, errors.ScimErrorInternal
	}

	// count=0 means "return totalResults only, no resources" (RFC 7644 §3.4.2.4).
	if params.Count == 0 {
		return scim.Page{
			Resources:    []scim.Resource{},
			TotalResults: int(total),
		}, nil
	}

	resources := make([]scim.Resource, len(users))
	for i, u := range users {
		resources[i] = toResource(u)
	}

	return scim.Page{
		Resources:    resources,
		TotalResults: int(total),
	}, nil
}

func (h userResourceHandler) Replace(r *http.Request, id string, attributes scim.ResourceAttributes) (scim.Resource, error) {
	u, err := resolveUser(r, id, h.sourceID)
	if err != nil {
		return scim.Resource{}, err
	}

	userName, _ := attributes["userName"].(string)
	if userName == "" {
		return scim.Resource{}, errors.ScimErrorBadRequest("attr `userName` missing")
	}

	email := extractPrimaryEmail(attributes)
	if email == "" {
		return scim.Resource{}, errors.ScimErrorBadRequest("primary email missing")
	}

	fullName, _ := attributes["displayName"].(string)

	active := true
	if v, ok := attributes["active"].(bool); ok {
		active = v
	}

	if err := syncUserAttributes(r.Context(), u, userName, fullName, email, active); err != nil {
		log.Error("SCIM: failed to update user: %v", err)
		return scim.Resource{}, errors.ScimErrorInternal
	}

	return toResource(u), nil
}

func (h userResourceHandler) Delete(r *http.Request, id string) error {
	u, err := resolveUser(r, id, h.sourceID)
	if err != nil {
		return err
	}
	return user_service.DeleteUser(r.Context(), u, true)
}

func (h userResourceHandler) Patch(_ *http.Request, _ string, _ []scim.PatchOperation) (scim.Resource, error) {
	return scim.Resource{}, errors.ScimErrorBadRequest("SCIM PATCH is not implemented.")
}
