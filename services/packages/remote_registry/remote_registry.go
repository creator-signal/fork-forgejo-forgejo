package remote_registry

import (
	"fmt"

	rr_model "forgejo.org/models/remote_registry"
	"forgejo.org/services/context"
)

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
