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
	"cmp"
	"fmt"
	"maps"
	"net"
	"strings"

	"github.com/gravitational/trace"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"

	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/teleport/lib/defaults"
)

// The google.golang.org/api SDK types Cloud SQL enum-like fields as plain
// strings with no generated constants, so the values we depend on are named here.
const (
	// Accepted values for types.GCPDatabaseEndpointTypeOverrideLabel
	endpointTypePublic  = "public"
	endpointTypePrivate = "private"
	endpointTypePSC     = "psc"

	instanceTypePrimary     = "CLOUD_SQL_INSTANCE"
	instanceTypeReadReplica = "READ_REPLICA_INSTANCE"

	labelInstanceTypePrimary     = "primary"
	labelInstanceTypeReadReplica = "read-replica"
)

func isInstanceAvailable(instance *sqladmin.DatabaseInstance) bool {
	const stateRunnable = "RUNNABLE"
	return instance.State == stateRunnable
}

func protocolAndPort(databaseVersion string) (protocol, port string, ok bool) {
	const (
		versionPrefixMySQL    = "MYSQL_"
		versionPrefixPostgres = "POSTGRES_"

		defaultPortMySQL    = "3306"
		defaultPortPostgres = "5432"
	)

	switch {
	case strings.HasPrefix(databaseVersion, versionPrefixMySQL):
		return defaults.ProtocolMySQL, defaultPortMySQL, true
	case strings.HasPrefix(databaseVersion, versionPrefixPostgres):
		return defaults.ProtocolPostgres, defaultPortPostgres, true
	default:
		return "", "", false
	}
}

func pscEnabled(instance *sqladmin.DatabaseInstance) bool {
	if instance.Settings == nil ||
		instance.Settings.IpConfiguration == nil ||
		instance.Settings.IpConfiguration.PscConfig == nil {
		return false
	}
	return instance.Settings.IpConfiguration.PscConfig.PscEnabled
}

type endpointTypeMap struct {
	public, private, psc string
}

func findInstanceEndpoints(instance *sqladmin.DatabaseInstance) endpointTypeMap {
	const (
		scopeInstance = "INSTANCE"

		connectionTypePublic = "PUBLIC"
		connectionTypePSA    = "PRIVATE_SERVICES_ACCESS"
		connectionTypePSC    = "PRIVATE_SERVICE_CONNECT"

		ipTypePrimary = "PRIMARY"
		ipTypePrivate = "PRIVATE"
	)

	endpoints := endpointTypeMap{}

	// find usable DNS names
	for _, dns := range instance.DnsNames {
		if dns.Name == "" || dns.DnsScope != scopeInstance {
			continue
		}
		switch dns.ConnectionType {
		case connectionTypePublic:
			endpoints.public = dns.Name
		case connectionTypePSA:
			endpoints.private = dns.Name
		case connectionTypePSC:
			if pscEnabled(instance) {
				endpoints.psc = dns.Name
			}
		}
	}

	// fallback to IP addresses
	for _, ipAddr := range instance.IpAddresses {
		if ipAddr.IpAddress == "" {
			continue
		}

		switch {
		case ipAddr.Type == ipTypePrimary && endpoints.public == "":
			endpoints.public = ipAddr.IpAddress
		case ipAddr.Type == ipTypePrivate && endpoints.private == "":
			endpoints.private = ipAddr.IpAddress
		}
	}

	return endpoints
}

func instanceUserLabel(instance *sqladmin.DatabaseInstance, key string) string {
	if instance.Settings == nil {
		return ""
	}
	return instance.Settings.UserLabels[key]
}

func chooseEndpoint(instance *sqladmin.DatabaseInstance) string {
	endpoints := findInstanceEndpoints(instance)

	// respect the override, even if it makes us pick an empty endpoint.
	if override := instanceUserLabel(instance, types.GCPDatabaseEndpointTypeOverrideLabel); override != "" {
		switch override {
		case endpointTypePublic:
			return endpoints.public
		case endpointTypePrivate:
			return endpoints.private
		case endpointTypePSC:
			return endpoints.psc
		}
	}

	return cmp.Or(endpoints.public, endpoints.private, endpoints.psc)
}

func isSupportedInstanceType(instance *sqladmin.DatabaseInstance) bool {
	return mapInstanceTypeLabel(instance.InstanceType) != ""
}

func mapInstanceTypeLabel(instanceType string) string {
	switch instanceType {
	case instanceTypePrimary:
		return labelInstanceTypePrimary
	case instanceTypeReadReplica:
		return labelInstanceTypeReadReplica
	}
	// Currently we don't support:
	// - READ_POOL_INSTANCE: need changes in DB agent TLS checks.
	// - ON_PREMISES_INSTANCE: not tested.
	return ""
}

// routing is the resolved protocol and connection endpoint for an instance.
type routing struct {
	protocol string
	uri      string
}

// resolveRouting determines how Teleport would route to an instance. On
// failure it returns a nil routing and the reason the instance is unroutable.
func resolveRouting(instance *sqladmin.DatabaseInstance) (*routing, string) {
	protocol, port, ok := protocolAndPort(instance.DatabaseVersion)
	if !ok {
		return nil, fmt.Sprintf("unsupported database version %q", instance.DatabaseVersion)
	}
	host := chooseEndpoint(instance)
	if host == "" {
		return nil, "no reachable connection endpoint"
	}
	return &routing{
		protocol: protocol,
		uri:      net.JoinHostPort(host, port),
	}, ""
}

// labelsFromInstance assembles the discovery labels for a Cloud SQL
// instance, including its user labels.
func labelsFromInstance(instance *sqladmin.DatabaseInstance, routing *routing) map[string]string {
	labels := make(map[string]string)

	if instance.Settings != nil {
		maps.Copy(labels, instance.Settings.UserLabels)
	}

	labels[types.CloudLabel] = types.CloudGCP
	labels[types.DiscoveryLabelGCPProjectID] = instance.Project
	labels[types.DiscoveryLabelRegion] = instance.Region
	labels[types.DiscoveryLabelEngine] = routing.protocol
	labels[types.DiscoveryLabelEngineVersion] = instance.DatabaseVersion
	labels[types.DiscoveryLabelStatus] = instance.State
	labels[types.DiscoveryLabelInstanceType] = mapInstanceTypeLabel(instance.InstanceType)
	return labels
}

// NewDatabaseFromInstance builds a types.Database from an instance.
// The metadata is passed through modifyMeta before construction.
// Ineligible instances are skipped: a non-empty skipReason is returned
// with a nil database and a nil error.
func NewDatabaseFromInstance(instance *sqladmin.DatabaseInstance, modifyMeta func(types.Metadata) types.Metadata) (db types.Database, skipReason string, err error) {
	if !isInstanceAvailable(instance) {
		return nil, fmt.Sprintf("instance is not available (state %q)", instance.State), nil
	}
	if !isSupportedInstanceType(instance) {
		return nil, fmt.Sprintf("unsupported instance type %q", instance.InstanceType), nil
	}
	rt, skipReason := resolveRouting(instance)
	if rt == nil {
		return nil, skipReason, nil
	}

	labels := labelsFromInstance(instance, rt)
	db, err = types.NewDatabaseV3(
		modifyMeta(types.Metadata{
			Name:        instance.Name,
			Description: fmt.Sprintf("Cloud SQL instance in %v", instance.Region),
			Labels:      labels,
		}),
		types.DatabaseSpecV3{
			Protocol: rt.protocol,
			URI:      rt.uri,
			GCP: types.GCPCloudSQL{
				ProjectID:  instance.Project,
				InstanceID: instance.Name,
			},
		})
	if err != nil {
		return nil, "", trace.Wrap(err)
	}
	return db, "", nil
}
