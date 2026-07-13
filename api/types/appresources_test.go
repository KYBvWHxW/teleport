/*
Copyright 2026 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package types

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoleAppResourcesValidation(t *testing.T) {
	manyRules := make([]AppResource, maxAppRulesPerRole+1)
	for i := range manyRules {
		manyRules[i] = AppResource{Paths: []string{"/health"}}
	}

	tests := []struct {
		name      string
		version   string
		allow     RoleConditions
		deny      RoleConditions
		assertErr require.ErrorAssertionFunc
	}{
		{
			name:      "v9 paths only",
			version:   V9,
			allow:     RoleConditions{AppResources: []AppResource{{Paths: []string{"/health"}}}},
			assertErr: require.NoError,
		},
		{
			name:      "v9 unsafe_allow_all alone",
			version:   V9,
			allow:     RoleConditions{AppResources: []AppResource{{UnsafeAllowAll: true}}},
			assertErr: require.NoError,
		},
		{
			name:    "v9 full rule with codes and hints",
			version: V9,
			allow: RoleConditions{AppResources: []AppResource{{
				Paths:          []string{"/api/v4/projects/{project}/**"},
				Methods:        []string{"GET", "HEAD"},
				Where:          `contains(user.traits["allowed_projects"], vars.project)`,
				AllowCode:      "repo_read",
				AllowReason:    "Read access to the repository API",
				DenyCodeHint:   "project_not_allowed",
				DenyReasonHint: "Project is not in the caller's allowlist",
			}}},
			assertErr: require.NoError,
		},
		{
			name:      "v9 expressions only",
			version:   V9,
			allow:     RoleConditions{AppResourcesExpressions: []string{`path.match(literal("health"))`}},
			assertErr: require.NoError,
		},
		{
			name:      "rule missing paths and unsafe_allow_all",
			version:   V9,
			allow:     RoleConditions{AppResources: []AppResource{{Methods: []string{"GET"}}}},
			assertErr: require.Error,
		},
		{
			name:      "unsafe_allow_all combined with paths",
			version:   V9,
			allow:     RoleConditions{AppResources: []AppResource{{UnsafeAllowAll: true, Paths: []string{"/health"}}}},
			assertErr: require.Error,
		},
		{
			name:      "allow_reason without allow_code",
			version:   V9,
			allow:     RoleConditions{AppResources: []AppResource{{Paths: []string{"/health"}, AllowReason: "why"}}},
			assertErr: require.Error,
		},
		{
			name:      "deny_reason_hint without deny_code_hint",
			version:   V9,
			allow:     RoleConditions{AppResources: []AppResource{{Paths: []string{"/health"}, Where: "user.name == \"a\"", DenyReasonHint: "why"}}},
			assertErr: require.Error,
		},
		{
			name:      "deny_code_hint without where",
			version:   V9,
			allow:     RoleConditions{AppResources: []AppResource{{Paths: []string{"/health"}, DenyCodeHint: "nope"}}},
			assertErr: require.Error,
		},
		{
			name:      "allow_code with reserved prefix",
			version:   V9,
			allow:     RoleConditions{AppResources: []AppResource{{Paths: []string{"/health"}, AllowCode: "teleport_read"}}},
			assertErr: require.Error,
		},
		{
			name:      "deny_code_hint with reserved prefix",
			version:   V9,
			allow:     RoleConditions{AppResources: []AppResource{{Paths: []string{"/health"}, Where: "user.name == \"a\"", DenyCodeHint: "teleport_nope"}}},
			assertErr: require.Error,
		},
		{
			name:      "where over the byte cap",
			version:   V9,
			allow:     RoleConditions{AppResources: []AppResource{{Paths: []string{"/health"}, Where: strings.Repeat("a", maxAppWhereBytes+1)}}},
			assertErr: require.Error,
		},
		{
			name:      "expression over the byte cap",
			version:   V9,
			allow:     RoleConditions{AppResourcesExpressions: []string{strings.Repeat("a", maxAppExpressionBytes+1)}},
			assertErr: require.Error,
		},
		{
			name:      "over the per-role rule cap",
			version:   V9,
			allow:     RoleConditions{AppResources: manyRules},
			assertErr: require.Error,
		},
		{
			name:      "app_resources under deny",
			version:   V9,
			deny:      RoleConditions{AppResources: []AppResource{{Paths: []string{"/health"}}}},
			assertErr: require.Error,
		},
		{
			name:      "app_resources_expressions under deny",
			version:   V9,
			deny:      RoleConditions{AppResourcesExpressions: []string{`path.match(literal("health"))`}},
			assertErr: require.Error,
		},
		{
			name:      "app_resources on v8 role",
			version:   V8,
			allow:     RoleConditions{AppResources: []AppResource{{Paths: []string{"/health"}}}},
			assertErr: require.Error,
		},
		{
			name:      "app_resources_expressions on v8 role",
			version:   V8,
			allow:     RoleConditions{AppResourcesExpressions: []string{`path.match(literal("health"))`}},
			assertErr: require.Error,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			role := &RoleV6{
				Metadata: Metadata{Name: "test"},
				Version:  test.version,
				Spec:     RoleSpecV6{Allow: test.allow, Deny: test.deny},
			}
			test.assertErr(t, role.CheckAndSetDefaults())
		})
	}
}
