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

package accesslist

import (
	"github.com/gravitational/trace"

	"github.com/gravitational/teleport/api/types/accesslist"
	"github.com/gravitational/teleport/api/types/header"
	"github.com/gravitational/teleport/lib/scopes"
)

func newMember(listName, name scopes.QualifiedName, kind string) (*accesslist.AccessListMember, error) {
	return accesslist.NewAccessListMemberWithScope(
		header.Metadata{Name: name.String()},
		accesslist.AccessListMemberSpec{
			AccessList:     listName.String(),
			Name:           name.String(),
			MembershipKind: kind,
		},
		listName.Scope,
	)
}

func getReviewFrequency(months int) (accesslist.ReviewFrequency, error) {
	f := accesslist.ReviewFrequency(months)
	switch f {
	case accesslist.OneMonth, accesslist.ThreeMonths, accesslist.SixMonths, accesslist.OneYear:
		return f, nil
	}
	return 0, trace.BadParameter("--audit-frequency must be one of 1, 3, 6, 12 (got %d)", months)
}

func getReviewDayOfMonth(day int) (accesslist.ReviewDayOfMonth, error) {
	d := accesslist.ReviewDayOfMonth(day)
	switch d {
	case accesslist.FirstDayOfMonth, accesslist.FifteenthDayOfMonth, accesslist.LastDayOfMonth:
		return d, nil
	}
	return 0, trace.BadParameter("--audit-day must be one of 1, 15, 31 (got %d)", day)
}
