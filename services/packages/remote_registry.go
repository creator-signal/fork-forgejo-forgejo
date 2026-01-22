package packages

import (
	"fmt"

	"forgejo.org/models/packages"
	rr_model "forgejo.org/models/remote_registry"
	"forgejo.org/modules/log"
	"forgejo.org/modules/validation"
	"forgejo.org/services/context"
)

type RRCredentials struct {
	RemoteUser     string
	RemotePassword string
	RemoteToken    string
}

type RROpts struct {
	Name       string
	RemoteURL  string
	RemoteType packages.Type
	OwnerType  rr_model.RemoteRegistryOwnerType
	OwnerID    int64
	Auth       RRCredentials
}

func NewRemoteRegistry(opts RROpts) (rr_model.RemoteRegistry, error) {
	result := rr_model.RemoteRegistry{
		Name:           opts.Name,
		RemoteURL:      opts.RemoteURL,
		RemoteType:     opts.RemoteType,
		OwnerType:      opts.OwnerType,
		OwnerID:        opts.OwnerID,
		RemoteUser:     opts.Auth.RemoteUser,
		RemotePassword: opts.Auth.RemotePassword,
		RemoteToken:    opts.Auth.RemoteToken,
	}
	if valid, err := validation.IsValid(result); !valid {
		return rr_model.RemoteRegistry{}, err
	}
	return result, nil
}

func CreateRemoteRegistry(ctx *context.APIContext, rr rr_model.RemoteRegistry) error {
	return rr_model.CreateRemoteRegistry(ctx, rr)
}

func GetOwnerType(ctx *context.APIContext) (rr_model.RemoteRegistryOwnerType, error) {
	if ctx.Repo.Repository != nil {
		return rr_model.RRRepo, nil
	} else if ctx.ContextUser.IsOrganization() {
		return rr_model.RROrg, nil
	} else if ctx.ContextUser.IsUser() {
		return rr_model.RRUser, nil
	}
	return "", fmt.Errorf("invalid owner type")
}

func GetRemoteRegistryByName(ctx *context.APIContext, ownerType rr_model.RemoteRegistryOwnerType, ownerID int64, name string) (*rr_model.RemoteRegistry, error) {
	rr, err := rr_model.GetRemoteRegistryByName(ctx, ownerType, ownerID, name)
	if err != nil {
		return &rr_model.RemoteRegistry{}, err
	}
	log.Trace("Found remote registry %q for ownerID: %d", name, ownerID)
	return rr, nil
}
