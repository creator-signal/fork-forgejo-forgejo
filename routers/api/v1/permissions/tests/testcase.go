// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

// See README.md for a documentation of the test logic

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"forgejo.org/modules/web/routing"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
	apiv1_permissions_testhelpers "forgejo.org/routers/api/v1/permissions/testhelpers"
)

type testData struct {
	own    map[string]string
	shared map[string]string
}

func verifySharedKey(key string) {
	if !slices.Contains([]string{
		"disable-units",
		"doer",
		"doer.actions.task.IsForkPullRequest",
		"doer.actions.task.RepoID",
		"doer.authentication",
		"doer.scope",
		"repository",
		"repository.init",
		"token.level",
	}, key) {
		panic(fmt.Sprintf("%s is not a shared key", key))
	}
}

func (o *testData) SetShared(key, value string) {
	verifySharedKey(key)
	o.shared[key] = value
}

func (o *testData) SetSharedDefault(key, value string) {
	if !o.HasShared(key) {
		o.SetShared(key, value)
	}
}

func (o *testData) GetShared(key string) string {
	verifySharedKey(key)
	return o.shared[key]
}

func (o *testData) HasShared(key string) bool {
	verifySharedKey(key)
	_, has := o.shared[key]
	return has
}

func (o *testData) SetOwn(key, value string) {
	o.own[key] = value
}

func (o *testData) SetOwnDefault(key, value string) {
	if !o.HasOwn(key) {
		o.SetOwn(key, value)
	}
}

func (o *testData) GetOwn(key string) string {
	return o.own[key]
}

func (o *testData) HasOwn(key string) bool {
	_, has := o.own[key]
	return has
}

func (o *testData) String() string {
	var s []string
	for k, e := range o.shared {
		s = append(s, fmt.Sprintf("%s:%s", k, e))
	}
	for k, e := range o.own {
		s = append(s, fmt.Sprintf("%s:%s", k, e))
	}
	slices.Sort(s)
	return strings.Join(s, ",")
}

func newTestData(own, shared map[string]string) *testData {
	testData := &testData{
		own:    make(map[string]string, 10),
		shared: make(map[string]string, 10),
	}
	for key, value := range own {
		testData.SetOwn(key, value)
	}
	for key, value := range shared {
		testData.SetShared(key, value)
	}
	return testData
}

func (o *testData) Clone() *testData {
	return &testData{
		own:    maps.Clone(o.own),
		shared: maps.Clone(o.shared),
	}
}

type testCase struct {
	data  *testData
	error string

	used bool
}

func (o *testCase) Clone() *testCase {
	f := *o
	f.data = o.data.Clone()
	return &f
}

// See README.md for a documentation of the test logic that uses
// this test description.
type functionTest struct {
	// The testCase will be constructed, when this function is the last
	// one of the chain.  It will go through the fulfillNeeds and
	// interpret of the previous functions in the chain, as well as its
	// own interpret.
	testCases []*testCase

	// List the settings which might be updated while interpreting the testData
	// so that they are restored upon test completion.
	protectSettingsBool []*bool

	// number of static arguments to pass to call's last argument
	staticArgs int
	// call the middleware (set automatically by [registerFunctionTest])
	call func(t *testing.T, ctx apiv1_permissions.Context, data *testData, staticArgs []any)

	sequenceFilter []string
	fulfillNeeds   func(t *testing.T, data *testData)
	interpret      func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData)
}

func buildSignatureStringToFunctionTest(t *testing.T) {
	for signatureString, signature := range apiv1_permissions_testhelpers.GetSignatureStringToSignature() {
		for prefix, builder := range prefixToFunctionTestBuilder {
			if strings.HasPrefix(signatureString, prefix) {
				builder(t, signatureString, signature)
			}
		}
	}
}

func registerFunctionTest(fun func(apiv1_permissions.Context), test functionTest) bool {
	shortName := routing.GetFuncShortName(fun)
	test.call = func(t *testing.T, ctx apiv1_permissions.Context, _ *testData, _ []any) {
		t.Logf("calling %s(ctx)", shortName)
		fun(ctx)
	}
	return registerFunctionTestWithCall(fun, test)
}

func registerFunctionTestWithCall(fun any, test functionTest) bool {
	signatureString := apiv1_permissions_testhelpers.SignatureToString([]any{fun})
	if _, has := signatureStringToFunctionTest[signatureString]; has {
		panic(fmt.Errorf("attempt to register %s twice", signatureString))
	}
	if test.call == nil {
		panic("'call' field is required")
	}
	signatureStringToFunctionTest[signatureString] = test
	return true
}

var signatureStringToFunctionTest = map[string]functionTest{}

type functionTestBuilder func(t *testing.T, signatureString string, signature []any)

func registerFunctionTestBuilder(prefixes []string, builder functionTestBuilder) bool {
	for _, prefix := range prefixes {
		if _, has := prefixToFunctionTestBuilder[prefix]; has {
			panic(fmt.Errorf("attempt to register %s twice", prefix))
		}
		prefixToFunctionTestBuilder[prefix] = builder
	}
	return true
}

var prefixToFunctionTestBuilder = map[string]functionTestBuilder{}
