/*
Copyright 2025 Gravitational, Inc.

Licensed under the Apache License, Config 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package testlib

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	foov1 "github.com/gravitational/teleport/api/gen/proto/go/teleport/foo/v1"
	headerv1 "github.com/gravitational/teleport/api/gen/proto/go/teleport/header/v1"
)

func (s *TerraformSuiteOSS) TestFoo() {
	name := "teleport_foo.test"

	resource.Test(s.T(), resource.TestCase{
		ProtoV6ProviderFactories:  s.terraformProviders,
		PreventPostDestroyRefresh: true,
		IsUnitTest:                true,
		Steps: []resource.TestStep{
			{
				Config: s.getFixture("foo_0_create.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "scope", "/example/basic"),
					resource.TestCheckResourceAttr(name, "spec.value", "value0"),
				),
			},
			{
				Config:   s.getFixture("foo_0_create.tf"),
				PlanOnly: true,
			},
			{
				Config: s.getFixture("foo_1_update.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "scope", "/example/basic"),
					resource.TestCheckResourceAttr(name, "spec.value", "value1"),
				),
			},
			{
				Config:   s.getFixture("foo_1_update.tf"),
				PlanOnly: true,
			},
		},
	})
}

func (s *TerraformSuiteOSS) TestImportFoo() {
	ctx, cancel := context.WithCancel(context.Background())
	s.T().Cleanup(cancel)

	r := "teleport_foo"
	scope := "/example/basic"
	teleportName := "test_foo"
	id := scope + "::" + teleportName
	name := r + "." + id

	foo := foov1.Foo_builder{
		Kind:    "foo",
		Version: "v1",
		Metadata: headerv1.Metadata_builder{
			Name: teleportName,
		}.Build(),
		Scope: scope,
		Spec: foov1.FooSpec_builder{
			Value: "value0",
		}.Build(),
	}.Build()

	createResp, err := s.client.FooClient().CreateFoo(ctx, foov1.CreateFooRequest_builder{
		Foo: foo,
	}.Build())
	require.NoError(s.T(), err)

	require.EventuallyWithT(s.T(), func(t *assert.CollectT) {
		getResp, err := s.client.FooClient().GetFoo(ctx, foov1.GetFooRequest_builder{
			Name:  id,
			Scope: scope,
		}.Build())
		require.NoError(s.T(), err)

		require.Equal(t, getResp.GetFoo().GetMetadata().GetRevision(), createResp.GetFoo().GetMetadata().GetRevision())
	}, 5*time.Second, 200*time.Millisecond)

	resource.Test(s.T(), resource.TestCase{
		ProtoV6ProviderFactories: s.terraformProviders,
		IsUnitTest:               true,
		Steps: []resource.TestStep{
			{
				Config:        s.terraformConfig + "\n" + `resource "` + r + `" "` + teleportName + `" { }`,
				ResourceName:  name,
				ImportState:   true,
				ImportStateId: id,
				ImportStateCheck: func(state []*terraform.InstanceState) error {
					require.Equal(s.T(), "foo", state[0].Attributes["kind"])
					require.Equal(s.T(), scope, state[0].Attributes["scope"])
					require.Equal(s.T(), "value0", state[0].Attributes["spec.value"])

					return nil
				},
			},
		},
	})
}
