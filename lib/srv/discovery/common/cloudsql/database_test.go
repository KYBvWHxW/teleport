/*
 * Teleport
 * Copyright (C) 2026  Gravitational, Inc.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package cloudsql

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"

	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/teleport/lib/defaults"
)

func ipMapping(ipType, addr string) *sqladmin.IpMapping {
	return &sqladmin.IpMapping{Type: ipType, IpAddress: addr}
}

func dnsMapping(name, connType, scope string) *sqladmin.DnsNameMapping {
	return &sqladmin.DnsNameMapping{Name: name, ConnectionType: connType, DnsScope: scope}
}

func TestIsInstanceAvailable(t *testing.T) {
	tests := []struct {
		name  string
		state string
		want  bool
	}{
		{name: "runnable", state: "RUNNABLE", want: true},
		{name: "suspended", state: "SUSPENDED", want: false},
		{name: "empty", state: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isInstanceAvailable(&sqladmin.DatabaseInstance{State: tt.state}))
		})
	}
}

func TestProtocolAndPort(t *testing.T) {
	tests := []struct {
		name         string
		version      string
		wantProtocol string
		wantPort     string
		wantOK       bool
	}{
		{name: "postgres", version: "POSTGRES_14", wantProtocol: defaults.ProtocolPostgres, wantPort: "5432", wantOK: true},
		{name: "mysql", version: "MYSQL_8_0", wantProtocol: defaults.ProtocolMySQL, wantPort: "3306", wantOK: true},
		{name: "sqlserver unsupported", version: "SQLSERVER_2019_STANDARD", wantOK: false},
		{name: "prefix without underscore", version: "POSTGRES", wantOK: false},
		{name: "case sensitive", version: "mysql_8_0", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protocol, port, ok := protocolAndPort(tt.version)
			require.Equal(t, tt.wantOK, ok)
			require.Equal(t, tt.wantProtocol, protocol)
			require.Equal(t, tt.wantPort, port)
		})
	}
}

func TestPSCEnabled(t *testing.T) {
	enabled := func(b bool) *sqladmin.Settings {
		return &sqladmin.Settings{
			IpConfiguration: &sqladmin.IpConfiguration{PscConfig: &sqladmin.PscConfig{PscEnabled: b}},
		}
	}
	tests := []struct {
		name     string
		settings *sqladmin.Settings
		want     bool
	}{
		{name: "nil settings", settings: nil, want: false},
		{name: "nil ip configuration", settings: &sqladmin.Settings{}, want: false},
		{name: "nil psc config", settings: &sqladmin.Settings{IpConfiguration: &sqladmin.IpConfiguration{}}, want: false},
		{name: "psc disabled", settings: enabled(false), want: false},
		{name: "psc enabled", settings: enabled(true), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, pscEnabled(&sqladmin.DatabaseInstance{Settings: tt.settings}))
		})
	}
}

func TestFindInstanceEndpoints(t *testing.T) {
	tests := []struct {
		name     string
		instance *sqladmin.DatabaseInstance
		want     endpointTypeMap
	}{
		{
			name:     "public DNS name",
			instance: &sqladmin.DatabaseInstance{DnsNames: []*sqladmin.DnsNameMapping{dnsMapping("pub.example", "PUBLIC", "INSTANCE")}},
			want:     endpointTypeMap{public: "pub.example"},
		},
		{
			name:     "PSA DNS name maps to private",
			instance: &sqladmin.DatabaseInstance{DnsNames: []*sqladmin.DnsNameMapping{dnsMapping("priv.example", "PRIVATE_SERVICES_ACCESS", "INSTANCE")}},
			want:     endpointTypeMap{private: "priv.example"},
		},
		{
			name:     "PSC DNS name ignored when PSC disabled",
			instance: &sqladmin.DatabaseInstance{DnsNames: []*sqladmin.DnsNameMapping{dnsMapping("psc.example", "PRIVATE_SERVICE_CONNECT", "INSTANCE")}},
			want:     endpointTypeMap{},
		},
		{
			name: "PSC DNS name used when PSC enabled",
			instance: &sqladmin.DatabaseInstance{
				DnsNames: []*sqladmin.DnsNameMapping{dnsMapping("psc.example", "PRIVATE_SERVICE_CONNECT", "INSTANCE")},
				Settings: &sqladmin.Settings{IpConfiguration: &sqladmin.IpConfiguration{PscConfig: &sqladmin.PscConfig{PscEnabled: true}}},
			},
			want: endpointTypeMap{psc: "psc.example"},
		},
		{
			name:     "non-INSTANCE scope ignored",
			instance: &sqladmin.DatabaseInstance{DnsNames: []*sqladmin.DnsNameMapping{dnsMapping("pub.example", "PUBLIC", "REGIONAL")}},
			want:     endpointTypeMap{},
		},
		{
			name:     "empty DNS name ignored",
			instance: &sqladmin.DatabaseInstance{DnsNames: []*sqladmin.DnsNameMapping{dnsMapping("", "PUBLIC", "INSTANCE")}},
			want:     endpointTypeMap{},
		},
		{
			name: "IP fallback for primary and private",
			instance: &sqladmin.DatabaseInstance{IpAddresses: []*sqladmin.IpMapping{
				ipMapping("PRIMARY", "1.2.3.4"),
				ipMapping("PRIVATE", "10.0.0.1"),
			}},
			want: endpointTypeMap{public: "1.2.3.4", private: "10.0.0.1"},
		},
		{
			name: "DNS public not overridden by primary IP",
			instance: &sqladmin.DatabaseInstance{
				DnsNames:    []*sqladmin.DnsNameMapping{dnsMapping("pub.example", "PUBLIC", "INSTANCE")},
				IpAddresses: []*sqladmin.IpMapping{ipMapping("PRIMARY", "1.2.3.4")},
			},
			want: endpointTypeMap{public: "pub.example"},
		},
		{
			name:     "empty IP address ignored",
			instance: &sqladmin.DatabaseInstance{IpAddresses: []*sqladmin.IpMapping{ipMapping("PRIMARY", "")}},
			want:     endpointTypeMap{},
		},
		{
			name: "private IP does not override private DNS name",
			instance: &sqladmin.DatabaseInstance{
				DnsNames:    []*sqladmin.DnsNameMapping{dnsMapping("priv.example", "PRIVATE_SERVICES_ACCESS", "INSTANCE")},
				IpAddresses: []*sqladmin.IpMapping{ipMapping("PRIVATE", "10.0.0.1")},
			},
			want: endpointTypeMap{private: "priv.example"},
		},
		{
			name:     "unknown connection type ignored",
			instance: &sqladmin.DatabaseInstance{DnsNames: []*sqladmin.DnsNameMapping{dnsMapping("weird.example", "SOMETHING_ELSE", "INSTANCE")}},
			want:     endpointTypeMap{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, findInstanceEndpoints(tt.instance))
		})
	}
}

func TestInstanceUserLabel(t *testing.T) {
	require.Empty(t, instanceUserLabel(&sqladmin.DatabaseInstance{}, "key"),
		"nil Settings yields empty string")
	require.Empty(t, instanceUserLabel(&sqladmin.DatabaseInstance{Settings: &sqladmin.Settings{}}, "key"),
		"nil UserLabels yields empty string")

	instance := &sqladmin.DatabaseInstance{
		Settings: &sqladmin.Settings{UserLabels: map[string]string{"env": "prod"}},
	}
	require.Equal(t, "prod", instanceUserLabel(instance, "env"))
	require.Empty(t, instanceUserLabel(instance, "missing"))
}

func TestChooseEndpoint(t *testing.T) {
	overrideLabels := func(surface string) *sqladmin.Settings {
		return &sqladmin.Settings{UserLabels: map[string]string{types.GCPDatabaseEndpointTypeOverrideLabel: surface}}
	}
	tests := []struct {
		name     string
		instance *sqladmin.DatabaseInstance
		want     string
	}{
		// Default precedence (no override): public > private > psc.
		{
			name: "public preferred over private",
			instance: &sqladmin.DatabaseInstance{IpAddresses: []*sqladmin.IpMapping{
				ipMapping("PRIMARY", "1.2.3.4"),
				ipMapping("PRIVATE", "10.0.0.1"),
			}},
			want: "1.2.3.4",
		},
		{
			name:     "private when no public",
			instance: &sqladmin.DatabaseInstance{IpAddresses: []*sqladmin.IpMapping{ipMapping("PRIVATE", "10.0.0.1")}},
			want:     "10.0.0.1",
		},
		{
			name: "psc when only psc",
			instance: &sqladmin.DatabaseInstance{
				DnsNames: []*sqladmin.DnsNameMapping{dnsMapping("psc.example", "PRIVATE_SERVICE_CONNECT", "INSTANCE")},
				Settings: &sqladmin.Settings{IpConfiguration: &sqladmin.IpConfiguration{PscConfig: &sqladmin.PscConfig{PscEnabled: true}}},
			},
			want: "psc.example",
		},
		{
			name:     "no endpoint",
			instance: &sqladmin.DatabaseInstance{},
			want:     "",
		},
		{
			name: "override public",
			instance: &sqladmin.DatabaseInstance{
				IpAddresses: []*sqladmin.IpMapping{ipMapping("PRIMARY", "1.2.3.4")},
				Settings:    overrideLabels(endpointTypePublic),
			},
			want: "1.2.3.4",
		},
		{
			name: "override private",
			instance: &sqladmin.DatabaseInstance{
				IpAddresses: []*sqladmin.IpMapping{ipMapping("PRIMARY", "1.2.3.4"), ipMapping("PRIVATE", "10.0.0.1")},
				Settings:    overrideLabels(endpointTypePrivate),
			},
			want: "10.0.0.1",
		},
		{
			name: "override psc",
			instance: &sqladmin.DatabaseInstance{
				DnsNames: []*sqladmin.DnsNameMapping{dnsMapping("psc.example", "PRIVATE_SERVICE_CONNECT", "INSTANCE")},
				Settings: &sqladmin.Settings{
					IpConfiguration: &sqladmin.IpConfiguration{PscConfig: &sqladmin.PscConfig{PscEnabled: true}},
					UserLabels:      map[string]string{types.GCPDatabaseEndpointTypeOverrideLabel: endpointTypePSC},
				},
			},
			want: "psc.example",
		},
		{
			name: "override to absent surface returns empty",
			instance: &sqladmin.DatabaseInstance{
				IpAddresses: []*sqladmin.IpMapping{ipMapping("PRIMARY", "1.2.3.4")}, // only public present
				Settings:    overrideLabels(endpointTypePrivate),
			},
			want: "",
		},
		{
			name: "unrecognized override falls back to default precedence",
			instance: &sqladmin.DatabaseInstance{
				IpAddresses: []*sqladmin.IpMapping{ipMapping("PRIMARY", "1.2.3.4")},
				Settings:    overrideLabels("bogus"),
			},
			want: "1.2.3.4",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, chooseEndpoint(tt.instance))
		})
	}
}

func TestMapInstanceTypeLabel(t *testing.T) {
	tests := []struct {
		instanceType string
		want         string
	}{
		{instanceType: instanceTypePrimary, want: labelInstanceTypePrimary},
		{instanceType: instanceTypeReadReplica, want: labelInstanceTypeReadReplica},
		{instanceType: "READ_POOL_INSTANCE", want: ""},
		{instanceType: "ON_PREMISES_INSTANCE", want: ""},
		{instanceType: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.instanceType, func(t *testing.T) {
			require.Equal(t, tt.want, mapInstanceTypeLabel(tt.instanceType))
		})
	}
}

func TestResolveRouting(t *testing.T) {
	tests := []struct {
		name         string
		instance     *sqladmin.DatabaseInstance
		wantNil      bool
		wantProtocol string
		wantURI      string
	}{
		{
			name: "supported engine with reachable endpoint",
			instance: &sqladmin.DatabaseInstance{
				DatabaseVersion: "POSTGRES_14",
				IpAddresses:     []*sqladmin.IpMapping{ipMapping("PRIMARY", "1.2.3.4")},
			},
			wantProtocol: defaults.ProtocolPostgres,
			wantURI:      "1.2.3.4:5432",
		},
		{
			name: "hostname endpoint joined without brackets",
			instance: &sqladmin.DatabaseInstance{
				DatabaseVersion: "POSTGRES_14",
				DnsNames:        []*sqladmin.DnsNameMapping{dnsMapping("inst.psc.example", "PRIVATE_SERVICE_CONNECT", "INSTANCE")},
				Settings:        &sqladmin.Settings{IpConfiguration: &sqladmin.IpConfiguration{PscConfig: &sqladmin.PscConfig{PscEnabled: true}}},
			},
			wantProtocol: defaults.ProtocolPostgres,
			wantURI:      "inst.psc.example:5432",
		},
		{
			name: "unsupported engine",
			instance: &sqladmin.DatabaseInstance{
				DatabaseVersion: "SQLSERVER_2019_STANDARD",
				IpAddresses:     []*sqladmin.IpMapping{ipMapping("PRIMARY", "1.2.3.4")},
			},
			wantNil: true,
		},
		{
			name:     "no reachable endpoint",
			instance: &sqladmin.DatabaseInstance{DatabaseVersion: "POSTGRES_14"},
			wantNil:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, skipReason := resolveRouting(tt.instance)
			if tt.wantNil {
				require.Nil(t, r)
				require.NotEmpty(t, skipReason)
				return
			}
			require.NotNil(t, r)
			require.Empty(t, skipReason)
			require.Equal(t, tt.wantProtocol, r.protocol)
			require.Equal(t, tt.wantURI, r.uri)
		})
	}
}

func TestLabelsFromInstance(t *testing.T) {
	tests := []struct {
		name     string
		instance *sqladmin.DatabaseInstance
		routing  *routing
		want     map[string]string
	}{
		{
			name: "user labels merged with discovery labels",
			instance: &sqladmin.DatabaseInstance{
				Project:         "proj-1",
				Region:          "us-central1",
				State:           "RUNNABLE",
				DatabaseVersion: "POSTGRES_14",
				InstanceType:    instanceTypePrimary,
				Settings:        &sqladmin.Settings{UserLabels: map[string]string{"env": "prod", "team": "data"}},
			},
			routing: &routing{protocol: defaults.ProtocolPostgres},
			want: map[string]string{
				"env":                             "prod",
				"team":                            "data",
				types.CloudLabel:                  types.CloudGCP,
				types.DiscoveryLabelGCPProjectID:  "proj-1",
				types.DiscoveryLabelRegion:        "us-central1",
				types.DiscoveryLabelEngine:        defaults.ProtocolPostgres,
				types.DiscoveryLabelEngineVersion: "POSTGRES_14",
				types.DiscoveryLabelStatus:        "RUNNABLE",
				types.DiscoveryLabelInstanceType:  labelInstanceTypePrimary,
			},
		},
		{
			name: "nil settings yields discovery labels only",
			instance: &sqladmin.DatabaseInstance{
				Project:         "proj-1",
				Region:          "us-central1",
				State:           "RUNNABLE",
				DatabaseVersion: "POSTGRES_14",
				InstanceType:    instanceTypePrimary,
			},
			routing: &routing{protocol: defaults.ProtocolPostgres},
			want: map[string]string{
				types.CloudLabel:                  types.CloudGCP,
				types.DiscoveryLabelGCPProjectID:  "proj-1",
				types.DiscoveryLabelRegion:        "us-central1",
				types.DiscoveryLabelEngine:        defaults.ProtocolPostgres,
				types.DiscoveryLabelEngineVersion: "POSTGRES_14",
				types.DiscoveryLabelStatus:        "RUNNABLE",
				types.DiscoveryLabelInstanceType:  labelInstanceTypePrimary,
			},
		},
		{
			// raw engine-version enum; empty fields yield empty labels.
			name:     "mysql raw engine version, sparse instance",
			instance: &sqladmin.DatabaseInstance{DatabaseVersion: "MYSQL_8_0"},
			routing:  &routing{protocol: defaults.ProtocolMySQL},
			want: map[string]string{
				types.CloudLabel:                  types.CloudGCP,
				types.DiscoveryLabelGCPProjectID:  "",
				types.DiscoveryLabelRegion:        "",
				types.DiscoveryLabelEngine:        defaults.ProtocolMySQL,
				types.DiscoveryLabelEngineVersion: "MYSQL_8_0",
				types.DiscoveryLabelStatus:        "",
				types.DiscoveryLabelInstanceType:  "",
			},
		},
		{
			// computed keys override colliding user labels ("us-spoof"/"backdoor" lose).
			name: "computed discovery labels win over colliding user labels",
			instance: &sqladmin.DatabaseInstance{
				Project:         "proj-1",
				Region:          "us-central1",
				State:           "RUNNABLE",
				DatabaseVersion: "POSTGRES_14",
				InstanceType:    instanceTypePrimary,
				Settings: &sqladmin.Settings{UserLabels: map[string]string{
					types.DiscoveryLabelRegion:       "us-spoof", // collides with computed region
					types.DiscoveryLabelInstanceType: "backdoor", // collides with computed instance-type
					"env":                            "prod",     // non-colliding control
				}},
			},
			routing: &routing{protocol: defaults.ProtocolPostgres},
			want: map[string]string{
				"env":                             "prod",
				types.CloudLabel:                  types.CloudGCP,
				types.DiscoveryLabelGCPProjectID:  "proj-1",
				types.DiscoveryLabelRegion:        "us-central1",
				types.DiscoveryLabelEngine:        defaults.ProtocolPostgres,
				types.DiscoveryLabelEngineVersion: "POSTGRES_14",
				types.DiscoveryLabelStatus:        "RUNNABLE",
				types.DiscoveryLabelInstanceType:  labelInstanceTypePrimary,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, labelsFromInstance(tt.instance, tt.routing))
		})
	}
}

func TestNewDatabaseFromInstance(t *testing.T) {
	instance := &sqladmin.DatabaseInstance{
		Name:            "pg-instance",
		Project:         "proj-1",
		Region:          "us-central1",
		State:           "RUNNABLE",
		DatabaseVersion: "POSTGRES_14",
		InstanceType:    instanceTypePrimary,
		IpAddresses:     []*sqladmin.IpMapping{ipMapping("PRIMARY", "1.2.3.4")},
		Settings:        &sqladmin.Settings{UserLabels: map[string]string{"env": "prod"}},
	}
	identity := func(meta types.Metadata) types.Metadata { return meta }
	got, skipReason, err := NewDatabaseFromInstance(instance, identity)
	require.NoError(t, err)
	require.Empty(t, skipReason)

	want, err := types.NewDatabaseV3(types.Metadata{
		Name:        "pg-instance",
		Description: "Cloud SQL instance in us-central1",
		Labels: map[string]string{
			"env":                             "prod",
			types.CloudLabel:                  types.CloudGCP,
			types.DiscoveryLabelGCPProjectID:  "proj-1",
			types.DiscoveryLabelRegion:        "us-central1",
			types.DiscoveryLabelEngine:        defaults.ProtocolPostgres,
			types.DiscoveryLabelEngineVersion: "POSTGRES_14",
			types.DiscoveryLabelStatus:        "RUNNABLE",
			types.DiscoveryLabelInstanceType:  labelInstanceTypePrimary,
		},
	}, types.DatabaseSpecV3{
		Protocol: defaults.ProtocolPostgres,
		URI:      "1.2.3.4:5432",
		GCP: types.GCPCloudSQL{
			ProjectID:  "proj-1",
			InstanceID: "pg-instance",
		},
	})
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(want, got))
}

