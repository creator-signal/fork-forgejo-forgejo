package packages

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"forgejo.org/models/packages"
	rr_model "forgejo.org/models/remote_registry"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/modules/validation"
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
	remoteHost, err := url.Parse(opts.RemoteURL)
	if err != nil {
		return rr_model.RemoteRegistry{}, err
	}

	remotePort := 0
	if remoteHost.Port() != "" {
		remotePort, err = strconv.Atoi(remoteHost.Port())
		if err != nil {
			return rr_model.RemoteRegistry{}, err
		}
	}

	result := rr_model.RemoteRegistry{
		Name:           opts.Name,
		RemoteURL:      opts.RemoteURL,
		RemoteHost:     remoteHost.Host,
		RemotePort:     uint16(remotePort),
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

func CreateRemoteRegistry(ctx context.Context, rr rr_model.RemoteRegistry) error {
	return rr_model.CreateRemoteRegistry(ctx, rr)
}

func UpdateRemoteRegistry(ctx context.Context, rr rr_model.RemoteRegistry, name string) error {
	return rr_model.UpdateRemoteRegistry(ctx, rr, name)
}

func DeleteRemoteRegistry(ctx context.Context, ownerType rr_model.RemoteRegistryOwnerType, ownerID int64, registryName string) error {
	return rr_model.DeleteRemoteRegistry(ctx, ownerType, ownerID, registryName)
}

func GetOwnerType(ctx context.Context, isOrg, isUser bool) (rr_model.RemoteRegistryOwnerType, error) {
	if isOrg {
		return rr_model.RROrg, nil
	} else if isUser {
		return rr_model.RRUser, nil
	}
	return "", fmt.Errorf("invalid owner type")
}

func GetRemoteRegistry(ctx context.Context, isOrg, isUser bool, userName, registryName string) (*rr_model.RemoteRegistry, error) {
	ownerType, err := GetOwnerType(ctx, isOrg, isUser)
	if err != nil {
		return &rr_model.RemoteRegistry{}, err
	}

	// Get correct registry from params
	owner, err := user_model.GetUserByName(ctx, userName)
	if err != nil {
		return &rr_model.RemoteRegistry{}, err
	}

	rr, err := rr_model.GetRemoteRegistryByName(ctx, ownerType, owner.ID, registryName)
	if err != nil {
		return &rr_model.RemoteRegistry{}, err
	}
	log.Trace("Found remote registry %q for ownerID: %d", registryName, owner.ID)
	return rr, nil
}

func GetRemoteRegistries(ctx context.Context, isOrg, isUser bool, userName string) ([]*rr_model.RemoteRegistry, error) {
	ownerType, err := GetOwnerType(ctx, isOrg, isUser)
	if err != nil {
		return []*rr_model.RemoteRegistry{}, err
	}

	// Get correct registry from params
	owner, err := user_model.GetUserByName(ctx, userName)
	if err != nil {
		return []*rr_model.RemoteRegistry{}, err
	}

	rrs, err := rr_model.GetRemoteRegistriesByOwnerType(ctx, ownerType, owner.ID)
	if err != nil {
		return []*rr_model.RemoteRegistry{}, err
	}
	log.Trace("Found remote registries for ownerID: %d", owner.ID)
	return rrs, nil
}
