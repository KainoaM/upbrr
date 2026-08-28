// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package dvl

import "github.com/autobrr/upbrr/internal/trackers"

func duplicatePolicy() *trackers.DupePolicy {
	return &trackers.DupePolicy{
		ID:                  "dvl/duplicate/v1",
		EvidenceID:          "dvl-owner-guidance-exact-duplicates-only",
		LiteralIdentityOnly: true,
	}
}
