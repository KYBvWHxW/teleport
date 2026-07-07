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

package discovery

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gravitational/trace"

	discoveryconfigv1 "github.com/gravitational/teleport/api/gen/proto/go/teleport/discoveryconfig/v1"
)

const discoveryServiceSetupDocsURL = "https://goteleport.com/docs/reference/deployment/config/#discovery-service"

func (s discoverySummary) renderText(w io.Writer, now time.Time) error {
	if len(s) == 0 {
		_, err := fmt.Fprintln(w, "No AWS or Azure discovery_config resources are configured.")
		if err != nil {
			return trace.Wrap(err)
		}
		_, err = fmt.Fprintln(w, "Static discovery_service matchers from teleport.yaml do not report discovery config status.")
		return trace.Wrap(err)
	}

	for i, config := range s {
		if i > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return trace.Wrap(err)
			}
		}
		if err := config.writeText(w, now); err != nil {
			return trace.Wrap(err)
		}
	}

	return nil
}

func (c configSummary) writeText(w io.Writer, now time.Time) error {
	if err := writeSummaryLine(w, "Discovery config %s:", c.Name); err != nil {
		return trace.Wrap(err)
	}
	if err := writeSummaryLine(w, "  Discovery group: %s", c.DiscoveryGroup); err != nil {
		return trace.Wrap(err)
	}
	if err := writeSummaryLine(w, "  Status: %s", formatSummaryState(c.State)); err != nil {
		return trace.Wrap(err)
	}
	if c.LastSyncTime != nil {
		if err := writeSummaryLine(w, "  Last run: %s", formatSummaryLastRun(c.LastSyncTime, now)); err != nil {
			return trace.Wrap(err)
		}
	}
	if c.ErrorMessage != "" {
		if err := writeSummaryLine(w, "  Error: %s", c.ErrorMessage); err != nil {
			return trace.Wrap(err)
		}
	}
	if len(c.Servers) == 0 {
		return trace.Wrap(writeSummaryLine(w, "  No Discovery Services running for %s. See %s.", c.DiscoveryGroup, discoveryServiceSetupDocsURL))
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return trace.Wrap(err)
	}
	for _, server := range c.Servers {
		if err := server.writeText(w, now); err != nil {
			return trace.Wrap(err)
		}
	}
	return nil
}

func (s serverSummary) writeText(w io.Writer, now time.Time) error {
	if err := writeSummaryLine(w, "  Service (%s):", s.ServerID); err != nil {
		return trace.Wrap(err)
	}
	if s.PollInterval != "" {
		if err := writeSummaryLine(w, "    Poll interval: %s", formatPollInterval(s.PollInterval)); err != nil {
			return trace.Wrap(err)
		}
	}
	if s.LastUpdate != nil {
		if err := writeSummaryLine(w, "    Last update: %s", formatSummaryLastRun(s.LastUpdate, now)); err != nil {
			return trace.Wrap(err)
		}
	}
	for _, integration := range s.Integrations {
		if err := integration.writeText(w, now); err != nil {
			return trace.Wrap(err)
		}
	}
	return nil
}

func (i integrationSummary) writeText(w io.Writer, now time.Time) error {
	if err := writeSummaryLine(w, "    %s:", formatIntegrationName(i.Integration)); err != nil {
		return trace.Wrap(err)
	}
	for _, resource := range i.Resources {
		if err := resource.writeText(w, now); err != nil {
			return trace.Wrap(err)
		}
	}
	return nil
}

func (r resourceResult) writeText(w io.Writer, now time.Time) error {
	if err := writeSummaryLine(w, "      %s discovery:", r.Kind); err != nil {
		return trace.Wrap(err)
	}
	if err := writeSummaryLine(w, "        Previous sync: %s%s", formatSummaryLastRun(r.SyncEnd, now), formatSyncDuration(r.SyncStart, r.SyncEnd)); err != nil {
		return trace.Wrap(err)
	}
	return trace.Wrap(writeSummaryLine(w, "        Result: %s", formatResourceResult(r)))
}

func writeSummaryLine(w io.Writer, format string, args ...any) error {
	line := fmt.Sprintf(format, args...)
	_, err := fmt.Fprintln(w, strings.TrimRight(line, " "))
	return trace.Wrap(err)
}

func formatSummaryState(state string) string {
	switch state {
	case "", discoveryconfigv1.DiscoveryConfigState_DISCOVERY_CONFIG_STATE_UNSPECIFIED.String():
		return summaryStatusNotReporting
	case discoveryconfigv1.DiscoveryConfigState_DISCOVERY_CONFIG_STATE_ERROR.String():
		return "error"
	case discoveryconfigv1.DiscoveryConfigState_DISCOVERY_CONFIG_STATE_SYNCING.String():
		return "syncing"
	case discoveryconfigv1.DiscoveryConfigState_DISCOVERY_CONFIG_STATE_RUNNING.String():
		return "healthy"
	default:
		return state
	}
}

func formatIntegrationName(integration string) string {
	if integration == "" {
		return "ambient credentials"
	}
	return integration
}

func formatPollInterval(pollInterval string) string {
	duration, err := time.ParseDuration(pollInterval)
	if err != nil {
		return pollInterval
	}
	return strings.TrimSpace(humanize.RelTime(time.Time{}, time.Time{}.Add(duration), "", ""))
}

func formatSyncDuration(start, end *time.Time) string {
	if start == nil || end == nil {
		return ""
	}
	duration := end.Sub(*start).Round(time.Second)
	return " (took " + duration.String() + ")"
}

func formatResourceResult(result resourceResult) string {
	return strings.Join([]string{
		strconv.FormatUint(result.Found, 10) + " found",
		strconv.FormatUint(result.Enrolled, 10) + " enrolled",
		strconv.FormatUint(result.Failed, 10) + " failed",
	}, ", ")
}

func formatSummaryLastRun(t *time.Time, now time.Time) string {
	if t == nil || t.IsZero() {
		return "never"
	}
	return humanize.RelTime(*t, now, "ago", "from now")
}
