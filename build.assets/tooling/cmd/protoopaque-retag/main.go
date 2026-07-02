// Teleport
// Copyright (C) 2026 Gravitational, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"os"
	"strings"

	"github.com/gravitational/trace"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func main() {
	protogen.Options{}.Run(run)
}
func run(gen *protogen.Plugin) error {
	gen.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL | pluginpb.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS)
	gen.SupportedEditionsMinimum = descriptorpb.Edition_EDITION_PROTO2
	gen.SupportedEditionsMaximum = descriptorpb.Edition_EDITION_2024

	for _, f := range gen.Files {
		if !f.Generate {
			continue
		}

		prefix, ok := strings.CutPrefix(f.GeneratedFilenamePrefix, "github.com/gravitational/teleport/")
		if !ok {
			return trace.BadParameter("filename prefix %+q is not part of the Teleport package tree", f.GeneratedFilenamePrefix)
		}
		prefix = "./" + prefix

		isDefaultHybrid := defaultHybrid(f.Proto.GetName())

		{
			inPath := prefix + ".pb.go"
			source, err := os.ReadFile(inPath)
			if err != nil {
				return err
			}
			if err := os.Remove(inPath); err != nil {
				return err
			}

			pre, post, found := bytes.Cut(source, []byte("//go:build !protoopaque\n"))
			if !found {
				return trace.Errorf("missing build tag line in file %+q", inPath)
			}
			var replacement []byte
			if isDefaultHybrid {
				replacement = []byte("//go:build !teleport_protoopaque\n")
			} else {
				replacement = []byte("//go:build teleport_protohybrid\n")
			}

			if err := os.WriteFile(prefix+"_protohybrid.pb.go", bytes.Join([][]byte{pre, replacement, post}, nil), 0o644); err != nil {
				return err
			}
		}
		{
			inPath := prefix + "_protoopaque.pb.go"
			source, err := os.ReadFile(inPath)
			if err != nil {
				return err
			}
			if err := os.Remove(inPath); err != nil {
				return err
			}

			pre, post, found := bytes.Cut(source, []byte("//go:build protoopaque\n"))
			if !found {
				return trace.Errorf("missing build tag line in file %+q", inPath)
			}
			var replacement []byte
			if isDefaultHybrid {
				replacement = []byte("//go:build teleport_protoopaque\n")
			} else {
				replacement = []byte("//go:build !teleport_protohybrid\n")
			}

			if err := os.WriteFile(inPath, bytes.Join([][]byte{pre, replacement, post}, nil), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

func defaultHybrid(protoPath string) bool {
	// these files should default to the hybrid codegen because code referencing
	// them hasn't been updated yet
	switch protoPath {
	case
		"accessgraph/v1/session_search.proto",
		"accessgraph/v1alpha/access_graph_service.proto",
		"accessgraph/v1alpha/aws.proto",
		"accessgraph/v1alpha/azure.proto",
		"accessgraph/v1alpha/entra.proto",
		"accessgraph/v1alpha/events.proto",
		"accessgraph/v1alpha/github.proto",
		"accessgraph/v1alpha/gitlab.proto",
		"accessgraph/v1alpha/graph.proto",
		"accessgraph/v1alpha/netiq.proto",
		"accessgraph/v1alpha/okta.proto",
		"accessgraph/v1alpha/resources.proto",
		"teleport/access_graph/v1/authorized_key.proto",
		"teleport/access_graph/v1/private_key.proto",
		"teleport/access_graph/v1/secrets_service.proto",
		"teleport/accesslist/v1/accesslist_service.proto",
		"teleport/accesslist/v1/accesslist.proto",
		"teleport/accessmonitoringrules/v1/access_monitoring_rules_service.proto",
		"teleport/accessmonitoringrules/v1/access_monitoring_rules.proto",
		"teleport/appauthconfig/v1/appauthconfig_service.proto",
		"teleport/appauthconfig/v1/appauthconfig_sessions_service.proto",
		"teleport/appauthconfig/v1/appauthconfig.proto",
		"teleport/auditlog/v1/auditlog.proto",
		"teleport/autoupdate/v1/autoupdate_service.proto",
		"teleport/autoupdate/v1/autoupdate.proto",
		"teleport/backendinfo/v1/backendinfo.proto",
		"teleport/beams/v1/beam_service.proto",
		"teleport/beams/v1/beam.proto",
		"teleport/clientiprestriction/v1/clientiprestriction_service.proto",
		"teleport/clientiprestriction/v1/clientiprestriction.proto",
		"teleport/clusterconfig/v1/access_graph_settings.proto",
		"teleport/clusterconfig/v1/access_graph.proto",
		"teleport/clusterconfig/v1/clusterconfig_service.proto",
		"teleport/crownjewel/v1/crownjewel_service.proto",
		"teleport/crownjewel/v1/crownjewel.proto",
		"teleport/dbobject/v1/dbobject_service.proto",
		"teleport/dbobject/v1/dbobject.proto",
		"teleport/dbobjectimportrule/v1/dbobjectimportrule_service.proto",
		"teleport/dbobjectimportrule/v1/dbobjectimportrule.proto",
		"teleport/decision/v1alpha1/database_access.proto",
		"teleport/decision/v1alpha1/decision_service.proto",
		"teleport/decision/v1alpha1/denial_metadata.proto",
		"teleport/decision/v1alpha1/enforcement_feature.proto",
		"teleport/decision/v1alpha1/permit_metadata.proto",
		"teleport/decision/v1alpha1/request_metadata.proto",
		"teleport/decision/v1alpha1/resource.proto",
		"teleport/decision/v1alpha1/ssh_access.proto",
		"teleport/decision/v1alpha1/ssh_identity.proto",
		"teleport/decision/v1alpha1/ssh_join.proto",
		"teleport/decision/v1alpha1/tls_identity.proto",
		"teleport/delegation/v1/delegation_session_resource.proto",
		"teleport/delegation/v1/delegation_session_service.proto",
		"teleport/desktop/v1/tdpb.proto",
		"teleport/devicetrust/v1/assert.proto",
		"teleport/devicetrust/v1/authenticate_challenge.proto",
		"teleport/devicetrust/v1/device_collected_data.proto",
		"teleport/devicetrust/v1/device_confirmation_token.proto",
		"teleport/devicetrust/v1/device_enroll_token.proto",
		"teleport/devicetrust/v1/device_profile.proto",
		"teleport/devicetrust/v1/device_source.proto",
		"teleport/devicetrust/v1/device_web_token.proto",
		"teleport/devicetrust/v1/device.proto",
		"teleport/devicetrust/v1/devicetrust_service.proto",
		"teleport/devicetrust/v1/os_type.proto",
		"teleport/devicetrust/v1/tpm.proto",
		"teleport/devicetrust/v1/user_certificates.proto",
		"teleport/discoveryconfig/v1/discoveryconfig_service.proto",
		"teleport/discoveryconfig/v1/discoveryconfig.proto",
		"teleport/dynamicwindows/v1/dynamicwindows_service.proto",
		"teleport/embedding/v1/embedding.proto",
		"teleport/externalauditstorage/v1/externalauditstorage_service.proto",
		"teleport/externalauditstorage/v1/externalauditstorage.proto",
		"teleport/gitserver/v1/git_server_service.proto",
		"teleport/grpcclientconfig/v1/grpcclientconfig.proto",
		"teleport/grpcclientconfig/v1/grpcclientconfigservice.proto",
		"teleport/hardwarekeyagent/v1/hardwarekeyagent_service.proto",
		"teleport/header/v1/metadata.proto",
		"teleport/header/v1/resourceheader.proto",
		"teleport/healthcheckconfig/v1/health_check_config_service.proto",
		"teleport/healthcheckconfig/v1/health_check_config.proto",
		"teleport/identitycenter/v1/identitycenter.proto",
		"teleport/identitycenter/v1/service.proto",
		"teleport/integration/v1/awsoidc_service.proto",
		"teleport/integration/v1/awsra_service.proto",
		"teleport/integration/v1/integration_service.proto",
		"teleport/inventory/v1/inventory_service.proto",
		"teleport/issuance/v1/service.proto",
		"teleport/join/v1/joinservice.proto",
		"teleport/kube/v1/kube_service.proto",
		"teleport/kubewaitingcontainer/v1/kubewaitingcontainer_service.proto",
		"teleport/kubewaitingcontainer/v1/kubewaitingcontainer.proto",
		"teleport/label/v1/label.proto",
		"teleport/legacy/client/proto/event.proto",
		"teleport/legacy/client/proto/inventory.proto",
		"teleport/legacy/client/proto/requestable_roles.proto",
		"teleport/lib/multiplexer/test/ping.proto",
		"teleport/lib/teleterm/auto_update/v1/auto_update_service.proto",
		"teleport/lib/teleterm/v1/access_request.proto",
		"teleport/lib/teleterm/v1/app.proto",
		"teleport/lib/teleterm/v1/auth_settings.proto",
		"teleport/lib/teleterm/v1/cluster.proto",
		"teleport/lib/teleterm/v1/database.proto",
		"teleport/lib/teleterm/v1/gateway.proto",
		"teleport/lib/teleterm/v1/kube.proto",
		"teleport/lib/teleterm/v1/label.proto",
		"teleport/lib/teleterm/v1/server.proto",
		"teleport/lib/teleterm/v1/service.proto",
		"teleport/lib/teleterm/v1/target_health.proto",
		"teleport/lib/teleterm/v1/tshd_events_service.proto",
		"teleport/lib/teleterm/v1/usage_events.proto",
		"teleport/lib/teleterm/v1/windows_desktop.proto",
		"teleport/lib/teleterm/vnet/v1/vnet_service.proto",
		"teleport/lib/vnet/diag/v1/diag.proto",
		"teleport/lib/vnet/v1/client_application_service.proto",
		"teleport/linuxdesktop/v1/linux_desktop_service.proto",
		"teleport/linuxdesktop/v1/linux_desktop.proto",
		"teleport/loginrule/v1/loginrule_service.proto",
		"teleport/loginrule/v1/loginrule.proto",
		"teleport/machineid/v1/bot_instance_service.proto",
		"teleport/machineid/v1/bot_instance.proto",
		"teleport/machineid/v1/bot_service.proto",
		"teleport/machineid/v1/bot.proto",
		"teleport/machineid/v1/federation_service.proto",
		"teleport/machineid/v1/federation.proto",
		"teleport/machineid/v1/workload_identity_service.proto",
		"teleport/notifications/v1/notifications_service.proto",
		"teleport/notifications/v1/notifications.proto",
		"teleport/okta/v1/okta_service.proto",
		"teleport/plugins/v1/plugin_service.proto",
		"teleport/presence/v1/relay_server.proto",
		"teleport/presence/v1/service.proto",
		"teleport/provisioning/v1/provisioning.proto",
		"teleport/quicpeering/v1alpha/dial.proto",
		"teleport/recordingencryption/v1/recording_encryption_service.proto",
		"teleport/recordingencryption/v1/recording_encryption.proto",
		"teleport/recordingmetadata/v1/recordingmetadata_service.proto",
		"teleport/recordingmetadata/v1/recordingmetadata.proto",
		"teleport/relaypeering/v1alpha/dial.proto",
		"teleport/relaytunnel/v1alpha/control.proto",
		"teleport/relaytunnel/v1alpha/dial.proto",
		"teleport/relaytunnel/v1alpha/discovery_service.proto",
		"teleport/resourceusage/v1/access_requests.proto",
		"teleport/resourceusage/v1/account_usage_type.proto",
		"teleport/resourceusage/v1/resourceusage_service.proto",
		"teleport/samlidp/v1/samlidp.proto",
		"teleport/scim/v1/scim_service.proto",
		"teleport/scopes/access/v1/assignment.proto",
		"teleport/scopes/access/v1/role.proto",
		"teleport/scopes/access/v1/service.proto",
		"teleport/scopes/joining/v1/service.proto",
		"teleport/scopes/joining/v1/token.proto",
		"teleport/scopes/v1/scopes.proto",
		"teleport/secreports/v1/secreports_service.proto",
		"teleport/secreports/v1/secreports.proto",
		"teleport/sessionsearch/v1/session_search.proto",
		"teleport/ssh/v1/ssh.proto",
		"teleport/stableunixusers/v1/stableunixusers.proto",
		"teleport/storage/local/stableunixusers/v1/stableunixusers.proto",
		"teleport/subca/v1/cert_authority_override_id.proto",
		"teleport/subca/v1/cert_authority_override.proto",
		"teleport/subca/v1/certificate_override_id.proto",
		"teleport/subca/v1/certificate_override.proto",
		"teleport/subca/v1/crl.proto",
		"teleport/subca/v1/csr.proto",
		"teleport/subca/v1/distinguished_name.proto",
		"teleport/subca/v1/public_key_hash.proto",
		"teleport/subca/v1/subca_service.proto",
		"teleport/summarizer/v1/access_request.proto",
		"teleport/summarizer/v1/summarizer_service.proto",
		"teleport/summarizer/v1/summarizer.proto",
		"teleport/trait/v1/trait.proto",
		"teleport/transport/v1/transport_service.proto",
		"teleport/trust/v1/trust_service.proto",
		"teleport/userloginstate/v1/userloginstate_service.proto",
		"teleport/userloginstate/v1/userloginstate.proto",
		"teleport/userpreferences/v1/access_graph.proto",
		"teleport/userpreferences/v1/assist.proto",
		"teleport/userpreferences/v1/cluster_preferences.proto",
		"teleport/userpreferences/v1/discover_resource_preferences.proto",
		"teleport/userpreferences/v1/onboard.proto",
		"teleport/userpreferences/v1/sidenav_preferences.proto",
		"teleport/userpreferences/v1/theme.proto",
		"teleport/userpreferences/v1/unified_resource_preferences.proto",
		"teleport/userpreferences/v1/userpreferences.proto",
		"teleport/userprovisioning/v2/statichostuser_service.proto",
		"teleport/userprovisioning/v2/statichostuser.proto",
		"teleport/users/v1/users_service.proto",
		"teleport/usertasks/v1/user_tasks_service.proto",
		"teleport/usertasks/v1/user_tasks.proto",
		"teleport/vnet/v1/vnet_config_service.proto",
		"teleport/vnet/v1/vnet_config.proto",
		"teleport/workloadcluster/v1/workloadcluster_service.proto",
		"teleport/workloadcluster/v1/workloadcluster.proto",
		"teleport/workloadidentity/v1/attrs.proto",
		"teleport/workloadidentity/v1/issuance_service.proto",
		"teleport/workloadidentity/v1/join_attrs.proto",
		"teleport/workloadidentity/v1/resource_service.proto",
		"teleport/workloadidentity/v1/resource.proto",
		"teleport/workloadidentity/v1/revocation_resource.proto",
		"teleport/workloadidentity/v1/revocation_service.proto",
		"teleport/workloadidentity/v1/sigstore_policy_resource.proto",
		"teleport/workloadidentity/v1/sigstore_policy_service.proto",
		"teleport/workloadidentity/v1/sigstore.proto",
		"teleport/workloadidentity/v1/x509_overrides_service.proto",
		"teleport/workloadidentity/v1/x509_overrides.proto":
		return true
	default:
		return false
	}
}
