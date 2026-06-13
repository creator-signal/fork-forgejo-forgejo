// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"fmt"

	auth_model "forgejo.org/models/auth"

	"code.forgejo.org/forgejo/runner/v12/act/jobparser"
	"go.yaml.in/yaml/v3"
)

type perm string

const (
	permNone  perm = "none"
	permRead  perm = "read"
	permWrite perm = "write"
)

type tokenPermissions struct {
	Code     perm `yaml:"code"`
	Releases perm `yaml:"releases"`
	Issues   perm `yaml:"issues"`
	Pr       perm `yaml:"pull-requests"`
	Actions  perm `yaml:"actions"`
	Wiki     perm `yaml:"wiki"`
	Projects perm `yaml:"projects"`
	Packages perm `yaml:"packages"`
}

func (p *tokenPermissions) combineWith(other *tokenPermissions) {
	if other.Code > p.Code {
		p.Code = other.Code
	}
	if other.Releases > p.Releases {
		p.Releases = other.Releases
	}
	if other.Issues > p.Issues {
		p.Issues = other.Issues
	}
	if other.Pr > p.Pr {
		p.Pr = other.Pr
	}
	if other.Actions > p.Actions {
		p.Actions = other.Actions
	}
	if other.Wiki > p.Wiki {
		p.Wiki = other.Wiki
	}
	if other.Projects > p.Projects {
		p.Projects = other.Projects
	}
	if other.Packages > p.Packages {
		p.Packages = other.Packages
	}
}

var (
	readAllPerms = &tokenPermissions{
		Code:     permRead,
		Releases: permRead,
		Issues:   permRead,
		Pr:       permRead,
		Actions:  permRead,
		Wiki:     permRead,
		Projects: permRead,
		Packages: permRead,
	}
	writeAllPerms = &tokenPermissions{
		Code:     permWrite,
		Releases: permWrite,
		Issues:   permWrite,
		Pr:       permWrite,
		Actions:  permWrite,
		Wiki:     permWrite,
		Projects: permWrite,
		Packages: permWrite,
	}
)

func parsePermissions(perms *yaml.Node) (*tokenPermissions, error) {
	if perms.Kind == yaml.ScalarNode {
		var str string
		if err := perms.Decode(&str); err != nil {
			return nil, fmt.Errorf("failed to decode scalar workflow permissions to string: %v", err)
		}
		switch str {
		case "read-all":
			return readAllPerms, nil
		case "write-all":
			return writeAllPerms, nil
		default:
			return nil, fmt.Errorf("failed to decode workflow permissions string: unknown value %v", str)
		}
	}

	var p tokenPermissions
	if err := perms.Decode(&p); err != nil {
		return nil, fmt.Errorf("failed to decode non-scalar workflow permissions: %v", err)
	}
	return &p, nil
}

func createTokenScope(workflow *jobparser.SingleWorkflow) (auth_model.AccessTokenScope, error) {
	workflowPerms, err := parsePermissions(&workflow.Permissions)
	if err != nil {
		return "", err
	}

	_, job := workflow.Job()
	jobPerms, err := parsePermissions(&job.Permissions)
	if err != nil {
		return "", err
	}

	workflowPerms.combineWith(jobPerms)

	var scope auth_model.AccessTokenScope
	switch workflowPerms.Packages {
	case permRead:
		scope += "," + auth_model.AccessTokenScopeReadPackage
	case permWrite:
		scope += "," + auth_model.AccessTokenScopeWritePackage
	}

	return scope.Normalize()
}
