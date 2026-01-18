// Copyright Forgejo Authors
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	repo_model "forgejo.org/models/repo"
	f3_util "forgejo.org/services/f3/util"

	"code.forgejo.org/f3/gof3/v3/tree/generic"
)

type ConverterUUIDAndAttachment interface {
	UUIDToAttachment(context.Context, string) (string, bool)
	AttachmentToUUID(context.Context, generic.Path) (string, bool)

	Add(uuid, path string)
}

type converterUUIDAndAttachment struct{}

func NewConverterUUIDAndAttachment() ConverterUUIDAndAttachment {
	return &converterUUIDAndAttachment{}
}

func (o *converterUUIDAndAttachment) Add(uuid, path string) {
	panic("not implemented")
}

func (o converterUUIDAndAttachment) UUIDToAttachment(ctx context.Context, uuid string) (string, bool) {
	attachment, err := repo_model.GetAttachmentByUUID(ctx, uuid)
	if err != nil {
		if repo_model.IsErrAttachmentNotExist(err) {
			return "", false
		}
		panic(fmt.Errorf("GetAttachmentByUUID(%s): %w", uuid, err))
	}

	f3Path, err := f3_util.ConvertForgejoToF3Path(ctx, attachment)
	if err != nil {
		panic(fmt.Errorf("ConvertForgejoToF3Path(%+v): %w", attachment, err))
	}

	return f3Path, true
}

func (o converterUUIDAndAttachment) AttachmentToUUID(ctx context.Context, path generic.Path) (string, bool) {
	id := path.Last().GetID().Int64()
	attachment, err := repo_model.GetAttachmentByID(ctx, id)
	if err != nil {
		if repo_model.IsErrAttachmentNotExist(err) {
			return "", false
		}
		panic(fmt.Errorf("GetAttachmentByID(%d): %w", id, err))
	}
	return attachment.UUID, true
}
