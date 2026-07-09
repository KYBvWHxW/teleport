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

package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gravitational/trace"

	"github.com/gravitational/teleport"
	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/teleport/lib/cloud/gcp"
	"github.com/gravitational/teleport/lib/srv/discovery/common"
	"github.com/gravitational/teleport/lib/utils/set"
)

// gcpDatabaseGetter lists and converts databases for one GCP database service.
type gcpDatabaseGetter func(ctx context.Context, cfg *gcpFetcherConfig) (types.Databases, error)

// gcpFetcherConfig is the configuration for a GCP database fetcher.
type gcpFetcherConfig struct {
	// Type is the type of DB matcher, for example "cloudsql".
	Type string
	// GCPClients are the GCP API clients.
	GCPClients gcp.Clients
	// ProjectID is the GCP project to search for databases.
	ProjectID string
	// Locations are the GCP location selectors to match cloud databases.
	// Empty matches all locations.
	Locations []string
	// Labels is a selector to match cloud databases.
	Labels types.Labels
	// DiscoveryConfigName is the name of the discovery config which originated the resource.
	// Might be empty when the fetcher is using static matchers:
	// ie teleport.yaml/discovery_service.<cloud>.<matcher>
	DiscoveryConfigName string
	// Logger is the slog.Logger.
	Logger *slog.Logger
}

// CheckAndSetDefaults validates the config and sets defaults.
func (c *gcpFetcherConfig) CheckAndSetDefaults() error {
	if c.Type == "" {
		return trace.BadParameter("missing parameter Type")
	}
	if c.GCPClients == nil {
		return trace.BadParameter("missing parameter GCPClients")
	}
	if c.ProjectID == "" {
		return trace.BadParameter("missing parameter ProjectID")
	}
	if len(c.Labels) == 0 {
		return trace.BadParameter("missing parameter Labels")
	}
	if c.Logger == nil {
		c.Logger = slog.With(
			teleport.ComponentKey, "watch:gcp",
			"labels", c.Labels,
			"locations", c.Locations,
			"project_id", c.ProjectID,
			"type", c.Type,
		)
	}
	return nil
}

func (c *gcpFetcherConfig) locationMatcher() func(region string) bool {
	s := set.New(c.Locations...)
	matchAll := s.Len() == 0 || s.Contains(types.Wildcard)

	return func(region string) bool {
		if matchAll {
			return true
		}
		return s.Contains(region)
	}
}

type gcpFetcher struct {
	cfg    gcpFetcherConfig
	getter gcpDatabaseGetter
}

func newGCPFetcher(cfg gcpFetcherConfig, getter gcpDatabaseGetter) (common.Fetcher, error) {
	if err := cfg.CheckAndSetDefaults(); err != nil {
		return nil, trace.Wrap(err)
	}
	return &gcpFetcher{
		cfg:    cfg,
		getter: getter,
	}, nil
}

// Cloud returns the cloud the fetcher is operating.
func (f *gcpFetcher) Cloud() string {
	return types.CloudGCP
}

// ResourceType identifies the resource type the fetcher is returning.
func (f *gcpFetcher) ResourceType() string {
	return types.KindDatabase
}

// FetcherType returns the matcher type (`discovery_service.gcp.[].types`).
func (f *gcpFetcher) FetcherType() string {
	return f.cfg.Type
}

// IntegrationName returns the integration name. GCP database discovery only
// supports ambient credentials, so this is always empty.
func (f *gcpFetcher) IntegrationName() string {
	return ""
}

// GetDiscoveryConfigName is the name of the discovery config which originated
// the resource. Might be empty when using static matchers.
func (f *gcpFetcher) GetDiscoveryConfigName() string {
	return f.cfg.DiscoveryConfigName
}

// Get returns GCP databases matching the fetcher's selectors.
func (f *gcpFetcher) Get(ctx context.Context) (types.ResourcesWithLabels, error) {
	databases, err := f.getter(ctx, &f.cfg)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	databases = filterDatabasesByLabels(ctx, databases, f.cfg.Labels, f.cfg.Logger)

	for _, db := range databases {
		common.ApplyGCPDatabaseNameSuffix(db, f.cfg.Type)
	}
	return databases.AsResources(), nil
}

// String returns the fetcher's string description.
func (f *gcpFetcher) String() string {
	return fmt.Sprintf("gcpFetcher(Type=%v, ProjectID=%v, Locations=%v, Labels=%v)",
		f.cfg.Type, f.cfg.ProjectID, f.cfg.Locations, f.cfg.Labels)
}
