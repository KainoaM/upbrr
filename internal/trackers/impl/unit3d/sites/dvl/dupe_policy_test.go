// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package dvl

import (
	"testing"

	"github.com/autobrr/upbrr/internal/trackers/dupe"
	"github.com/autobrr/upbrr/pkg/api"
)

func TestRelatedNonExactReleaseCoexists(t *testing.T) {
	t.Parallel()

	policy := Profile().DupePolicy
	if policy == nil {
		t.Fatal("DVL duplicate policy is missing")
	}
	target := api.TrackerDuplicateTarget{
		Names:      []string{"Example.Release.2026.1080p.WEB-DL-GRP"},
		Type:       "WEB-DL",
		Resolution: "1080p",
		SizeBytes:  1_000,
		FileNames:  []string{"Example.Release.2026.1080p.WEB-DL-GRP.mkv"},
	}
	candidate := dupe.TrackerCandidate{
		Name:       "Example.Release.2026.1080p.WEB-DL-OTHER",
		Type:       "WEB-DL",
		Resolution: "1080p",
		SizeBytes:  900,
		SizeKnown:  true,
		Files:      []string{"Example.Release.2026.1080p.WEB-DL-OTHER.mkv"},
		FileCount:  1,
	}

	evaluation := dupe.Evaluate(
		target,
		[]dupe.TrackerCandidate{candidate},
		*policy,
		dupe.SearchEvidence{Complete: true, WorkScope: dupe.WorkScopeProviderID},
	)
	if evaluation.Blocks || evaluation.RequiresAction || len(evaluation.Candidates) != 1 ||
		evaluation.Candidates[0].Relation != api.DupeRelationCoexists {
		t.Fatalf("related non-exact release evaluation = %#v", evaluation)
	}
	reasons := evaluation.Candidates[0].Reasons
	if len(reasons) != 1 || reasons[0].Code != "literal_identity_differs" ||
		reasons[0].Message != "Candidate has different literal release identity." {
		t.Fatalf("related non-exact release reasons = %#v", reasons)
	}
}

func TestExactNameDoesNotOverrideConflictingLiteralPayload(t *testing.T) {
	t.Parallel()

	policy := Profile().DupePolicy
	if policy == nil {
		t.Fatal("DVL duplicate policy is missing")
	}
	target := api.TrackerDuplicateTarget{
		Names:     []string{"Example.Release.2026.1080p.WEB-DL-GRP"},
		SizeBytes: 1_000,
		FileNames: []string{"Example.Release.2026.1080p.WEB-DL-GRP.mkv"},
	}
	candidate := dupe.TrackerCandidate{
		Name:      "Example.Release.2026.1080p.WEB-DL-GRP",
		SizeBytes: 900,
		SizeKnown: true,
		Files:     []string{"Example.Release.2026.1080p.WEB-DL-GRP-REPACK.mkv"},
		FileCount: 1,
	}

	evaluation := dupe.Evaluate(
		target,
		[]dupe.TrackerCandidate{candidate},
		*policy,
		dupe.SearchEvidence{Complete: true, WorkScope: dupe.WorkScopeProviderID},
	)
	if evaluation.Blocks || evaluation.RequiresAction || len(evaluation.Candidates) != 1 ||
		evaluation.Candidates[0].Relation != api.DupeRelationCoexists {
		t.Fatalf("conflicting exact-name release evaluation = %#v", evaluation)
	}
}

func TestRenamedReleaseWithIdenticalFilesAndSizeBlocks(t *testing.T) {
	t.Parallel()

	policy := Profile().DupePolicy
	if policy == nil {
		t.Fatal("DVL duplicate policy is missing")
	}
	target := api.TrackerDuplicateTarget{
		Names:     []string{"Example.Release.2026.1080p.WEB-DL-GRP"},
		SizeBytes: 1_000,
		FileNames: []string{
			"Example.Release.2026/Example.Release.2026.mkv",
			"Example.Release.2026/Example.Release.2026.en.srt",
		},
	}
	candidate := dupe.TrackerCandidate{
		Name:      "Example.Release.Renamed.2026.1080p.WEB-DL-OTHER",
		SizeBytes: 1_000,
		SizeKnown: true,
		Files:     []string{"release/EXAMPLE.RELEASE.2026.EN.SRT", "release/EXAMPLE.RELEASE.2026.MKV"},
		FileCount: 2,
	}

	evaluation := dupe.Evaluate(
		target,
		[]dupe.TrackerCandidate{candidate},
		*policy,
		dupe.SearchEvidence{Complete: true, WorkScope: dupe.WorkScopeProviderID},
	)
	if !evaluation.Blocks || evaluation.RequiresAction || len(evaluation.Candidates) != 1 ||
		evaluation.Candidates[0].Relation != api.DupeRelationExactDuplicate {
		t.Fatalf("renamed literal duplicate evaluation = %#v", evaluation)
	}
}

func TestIdenticalFilesWithDifferentSizeCoexist(t *testing.T) {
	t.Parallel()

	policy := Profile().DupePolicy
	if policy == nil {
		t.Fatal("DVL duplicate policy is missing")
	}
	target := api.TrackerDuplicateTarget{
		Names:     []string{"Example.Release.2026.1080p.WEB-DL-GRP"},
		SizeBytes: 1_000,
		FileNames: []string{"Example.Release.2026.mkv"},
	}
	candidate := dupe.TrackerCandidate{
		Name:      "Example.Release.2026.1080p.WEB-DL-GRP",
		SizeBytes: 999,
		SizeKnown: true,
		Files:     []string{"Example.Release.2026.mkv"},
		FileCount: 1,
	}

	evaluation := dupe.Evaluate(
		target,
		[]dupe.TrackerCandidate{candidate},
		*policy,
		dupe.SearchEvidence{Complete: true, WorkScope: dupe.WorkScopeProviderID},
	)
	if evaluation.Blocks || evaluation.RequiresAction || evaluation.Candidates[0].Relation != api.DupeRelationCoexists {
		t.Fatalf("different-size release evaluation = %#v", evaluation)
	}
}

func TestIdenticalFilesWithUnknownSizeCoexist(t *testing.T) {
	t.Parallel()

	policy := Profile().DupePolicy
	if policy == nil {
		t.Fatal("DVL duplicate policy is missing")
	}
	target := api.TrackerDuplicateTarget{
		Names:     []string{"Example.Release.2026.1080p.WEB-DL-GRP"},
		SizeBytes: 1_000,
		FileNames: []string{"Example.Release.2026.mkv"},
	}
	candidate := dupe.TrackerCandidate{
		Name:      "Example.Release.2026.1080p.WEB-DL-GRP",
		Files:     []string{"Example.Release.2026.mkv"},
		FileCount: 1,
	}

	evaluation := dupe.Evaluate(
		target,
		[]dupe.TrackerCandidate{candidate},
		*policy,
		dupe.SearchEvidence{Complete: true, WorkScope: dupe.WorkScopeProviderID},
	)
	if evaluation.Blocks || evaluation.RequiresAction || evaluation.Candidates[0].Relation != api.DupeRelationCoexists {
		t.Fatalf("unknown-size release evaluation = %#v", evaluation)
	}
}

func TestCompanionFileDifferenceCoexists(t *testing.T) {
	t.Parallel()

	policy := Profile().DupePolicy
	if policy == nil {
		t.Fatal("DVL duplicate policy is missing")
	}
	target := api.TrackerDuplicateTarget{
		Names:     []string{"Example.Release.2026.1080p.WEB-DL-GRP"},
		SizeBytes: 1_000,
		FileNames: []string{"Example.Release.2026.mkv", "Example.Release.2026.en.srt"},
	}
	candidate := dupe.TrackerCandidate{
		Name:      "Example.Release.2026.1080p.WEB-DL-GRP",
		SizeBytes: 1_000,
		SizeKnown: true,
		Files:     []string{"Example.Release.2026.mkv", "Example.Release.2026.nfo"},
		FileCount: 2,
	}

	evaluation := dupe.Evaluate(
		target,
		[]dupe.TrackerCandidate{candidate},
		*policy,
		dupe.SearchEvidence{Complete: true, WorkScope: dupe.WorkScopeProviderID},
	)
	if evaluation.Blocks || evaluation.RequiresAction || evaluation.Candidates[0].Relation != api.DupeRelationCoexists {
		t.Fatalf("companion-file mismatch evaluation = %#v", evaluation)
	}
}

func TestMissingCandidateFileListUsesCountAndSize(t *testing.T) {
	t.Parallel()

	policy := Profile().DupePolicy
	if policy == nil {
		t.Fatal("DVL duplicate policy is missing")
	}
	target := api.TrackerDuplicateTarget{
		Names:     []string{"Example.Release.2026.1080p.WEB-DL-GRP"},
		SizeBytes: 1_000,
		FileNames: []string{"Example.Release.2026.mkv", "Example.Release.2026.en.srt"},
	}
	candidate := dupe.TrackerCandidate{
		Name:      "Example.Release.Renamed.2026.1080p.WEB-DL-OTHER",
		SizeBytes: 1_000,
		SizeKnown: true,
		FileCount: 2,
	}

	evaluation := dupe.Evaluate(
		target,
		[]dupe.TrackerCandidate{candidate},
		*policy,
		dupe.SearchEvidence{Complete: true, WorkScope: dupe.WorkScopeProviderID},
	)
	if !evaluation.Blocks || evaluation.RequiresAction || evaluation.Candidates[0].Relation != api.DupeRelationExactDuplicate {
		t.Fatalf("missing-list identity evaluation = %#v", evaluation)
	}
}

func TestMissingCandidateFileListRequiresMatchingCount(t *testing.T) {
	t.Parallel()

	policy := Profile().DupePolicy
	if policy == nil {
		t.Fatal("DVL duplicate policy is missing")
	}
	target := api.TrackerDuplicateTarget{
		Names:     []string{"Example.Release.2026.1080p.WEB-DL-GRP"},
		SizeBytes: 1_000,
		FileNames: []string{"Example.Release.2026.mkv", "Example.Release.2026.en.srt"},
	}
	candidate := dupe.TrackerCandidate{
		Name:      "Example.Release.Renamed.2026.1080p.WEB-DL-OTHER",
		SizeBytes: 1_000,
		SizeKnown: true,
		FileCount: 1,
	}

	evaluation := dupe.Evaluate(
		target,
		[]dupe.TrackerCandidate{candidate},
		*policy,
		dupe.SearchEvidence{Complete: true, WorkScope: dupe.WorkScopeProviderID},
	)
	if evaluation.Blocks || evaluation.RequiresAction || evaluation.Candidates[0].Relation != api.DupeRelationCoexists {
		t.Fatalf("missing-list count mismatch evaluation = %#v", evaluation)
	}
}

func TestRenamedDiscWithSameSizeBlocks(t *testing.T) {
	t.Parallel()

	policy := Profile().DupePolicy
	if policy == nil {
		t.Fatal("DVL duplicate policy is missing")
	}
	target := api.TrackerDuplicateTarget{
		Names:     []string{"Example.Release.2026.COMPLETE.UHD.BLURAY-GRP"},
		Type:      "DISC",
		SizeBytes: 1_000,
	}
	candidate := dupe.TrackerCandidate{
		Name:      "Example.Release.Renamed.2026.COMPLETE.UHD.BLURAY-OTHER",
		SizeBytes: 1_000,
		SizeKnown: true,
		FileCount: 42,
	}

	evaluation := dupe.Evaluate(
		target,
		[]dupe.TrackerCandidate{candidate},
		*policy,
		dupe.SearchEvidence{Complete: true, WorkScope: dupe.WorkScopeProviderID},
	)
	if !evaluation.Blocks || evaluation.RequiresAction || evaluation.Candidates[0].Relation != api.DupeRelationExactDuplicate {
		t.Fatalf("renamed disc identity evaluation = %#v", evaluation)
	}
}

func TestDiscIdentityIgnoresLocalFileInventory(t *testing.T) {
	t.Parallel()

	policy := Profile().DupePolicy
	if policy == nil {
		t.Fatal("DVL duplicate policy is missing")
	}
	target := api.TrackerDuplicateTarget{
		Names:     []string{"Example.Release.2026.COMPLETE.UHD.BLURAY-GRP"},
		Type:      "DISC",
		SizeBytes: 1_000,
		FileNames: []string{"BDMV/index.bdmv", "BDMV/STREAM/00000.m2ts"},
	}
	candidate := dupe.TrackerCandidate{
		Name:      "Example.Release.Renamed.2026.COMPLETE.UHD.BLURAY-OTHER",
		SizeBytes: 1_000,
		SizeKnown: true,
		FileCount: 42,
	}

	evaluation := dupe.Evaluate(
		target,
		[]dupe.TrackerCandidate{candidate},
		*policy,
		dupe.SearchEvidence{Complete: true, WorkScope: dupe.WorkScopeProviderID},
	)
	if !evaluation.Blocks || evaluation.RequiresAction || evaluation.Candidates[0].Relation != api.DupeRelationExactDuplicate {
		t.Fatalf("disc identity with local inventory evaluation = %#v", evaluation)
	}
}

func TestRenamedReleaseWithoutFileListsUsesSize(t *testing.T) {
	t.Parallel()

	policy := Profile().DupePolicy
	if policy == nil {
		t.Fatal("DVL duplicate policy is missing")
	}
	target := api.TrackerDuplicateTarget{
		Names:     []string{"Example.Release.2026.1080p.WEB-DL-GRP"},
		Type:      "WEB-DL",
		SizeBytes: 1_000,
	}
	candidate := dupe.TrackerCandidate{
		Name:      "Example.Release.Renamed.2026.1080p.WEB-DL-OTHER",
		SizeBytes: 1_000,
		SizeKnown: true,
	}

	evaluation := dupe.Evaluate(
		target,
		[]dupe.TrackerCandidate{candidate},
		*policy,
		dupe.SearchEvidence{Complete: true, WorkScope: dupe.WorkScopeProviderID},
	)
	if !evaluation.Blocks || evaluation.RequiresAction || evaluation.Candidates[0].Relation != api.DupeRelationExactDuplicate {
		t.Fatalf("missing-list size identity evaluation = %#v", evaluation)
	}
}

func TestExistingSeasonPackContainingEpisodeCoexists(t *testing.T) {
	t.Parallel()

	policy := Profile().DupePolicy
	if policy == nil {
		t.Fatal("DVL duplicate policy is missing")
	}
	target := api.TrackerDuplicateTarget{
		Names:      []string{"Example.Show.S01E01.1080p.WEB-DL-GRP"},
		Type:       "WEB-DL",
		Resolution: "1080p",
		Season:     1,
		Episode:    1,
		SizeBytes:  1_000,
		FileNames:  []string{"Example.Show.S01E01.mkv"},
	}
	candidate := dupe.TrackerCandidate{
		Name:       "Example.Show.S01.1080p.WEB-DL-OTHER",
		Type:       "WEB-DL",
		Resolution: "1080p",
		Season:     1,
		Pack:       true,
		SizeBytes:  2_000,
		SizeKnown:  true,
		Files:      []string{"Example.Show.S01E01.mkv", "Example.Show.S01E02.mkv"},
		FileCount:  2,
	}

	evaluation := dupe.Evaluate(
		target,
		[]dupe.TrackerCandidate{candidate},
		*policy,
		dupe.SearchEvidence{Complete: true, WorkScope: dupe.WorkScopeProviderID},
	)
	if evaluation.Blocks || evaluation.RequiresAction || evaluation.Candidates[0].Relation != api.DupeRelationCoexists {
		t.Fatalf("existing season-pack evaluation = %#v", evaluation)
	}
}

func TestNonExactTrumpableCandidateCoexists(t *testing.T) {
	t.Parallel()

	policy := Profile().DupePolicy
	if policy == nil {
		t.Fatal("DVL duplicate policy is missing")
	}
	target := api.TrackerDuplicateTarget{
		Names:     []string{"Example.Release.2026.1080p.WEB-DL-GRP"},
		SizeBytes: 1_000,
		FileNames: []string{"Example.Release.2026-GRP.mkv"},
	}
	candidate := dupe.TrackerCandidate{
		Name:      "Example.Release.2026.1080p.WEB-DL-OTHER",
		SizeBytes: 900,
		SizeKnown: true,
		Files:     []string{"Example.Release.2026-OTHER.mkv"},
		FileCount: 1,
		Trumpable: true,
	}

	evaluation := dupe.Evaluate(
		target,
		[]dupe.TrackerCandidate{candidate},
		*policy,
		dupe.SearchEvidence{Complete: true, WorkScope: dupe.WorkScopeProviderID},
	)
	if evaluation.Blocks || evaluation.RequiresAction || evaluation.Candidates[0].Relation != api.DupeRelationCoexists {
		t.Fatalf("non-exact trumpable evaluation = %#v", evaluation)
	}
}

func TestIncompleteSearchRequiresActionForNonExactCandidate(t *testing.T) {
	t.Parallel()

	policy := Profile().DupePolicy
	if policy == nil {
		t.Fatal("DVL duplicate policy is missing")
	}
	target := api.TrackerDuplicateTarget{
		Names:     []string{"Example.Release.2026.1080p.WEB-DL-GRP"},
		SizeBytes: 1_000,
		FileNames: []string{"Example.Release.2026-GRP.mkv"},
	}
	candidate := dupe.TrackerCandidate{
		Name:      "Example.Release.2026.1080p.WEB-DL-OTHER",
		SizeBytes: 900,
		SizeKnown: true,
		Files:     []string{"Example.Release.2026-OTHER.mkv"},
		FileCount: 1,
	}

	evaluation := dupe.Evaluate(
		target,
		[]dupe.TrackerCandidate{candidate},
		*policy,
		dupe.SearchEvidence{Complete: false, WorkScope: dupe.WorkScopeProviderID},
	)
	if evaluation.Blocks || !evaluation.RequiresAction || evaluation.Complete ||
		evaluation.Candidates[0].Relation != api.DupeRelationCoexists {
		t.Fatalf("incomplete non-exact evaluation = %#v", evaluation)
	}
}

func TestExactNameFallbackRequiresNoKnownSizeConflict(t *testing.T) {
	t.Parallel()

	policy := Profile().DupePolicy
	if policy == nil {
		t.Fatal("DVL duplicate policy is missing")
	}
	for _, test := range []struct {
		name          string
		targetSize    int64
		candidateSize int64
		sizeKnown     bool
		wantBlock     bool
	}{
		{
			name:          "equal known size",
			targetSize:    1_000,
			candidateSize: 1_000,
			sizeKnown:     true,
			wantBlock:     true,
		},
		{
			name:          "unknown target size",
			candidateSize: 1_000,
			sizeKnown:     true,
			wantBlock:     true,
		},
		{
			name:       "unknown candidate size",
			targetSize: 1_000,
			wantBlock:  true,
		},
		{
			name:          "conflicting known size",
			targetSize:    1_000,
			candidateSize: 900,
			sizeKnown:     true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			target := api.TrackerDuplicateTarget{
				Names:     []string{"Example.Release.2026.1080p.WEB-DL-GRP"},
				SizeBytes: test.targetSize,
			}
			candidate := dupe.TrackerCandidate{
				Name:      "example.release.2026.1080p.web-dl-grp",
				SizeBytes: test.candidateSize,
				SizeKnown: test.sizeKnown,
				Files:     []string{"tracker-file-list-is-available.mkv"},
				FileCount: 1,
			}
			evaluation := dupe.Evaluate(
				target,
				[]dupe.TrackerCandidate{candidate},
				*policy,
				dupe.SearchEvidence{Complete: true, WorkScope: dupe.WorkScopeProviderID},
			)
			if evaluation.Blocks != test.wantBlock || evaluation.RequiresAction {
				t.Fatalf("exact-name fallback evaluation = %#v", evaluation)
			}
		})
	}
}

func TestFallbackIdentityRequiresMatchingKnownSize(t *testing.T) {
	t.Parallel()

	policy := Profile().DupePolicy
	if policy == nil {
		t.Fatal("DVL duplicate policy is missing")
	}
	for _, test := range []struct {
		name      string
		target    api.TrackerDuplicateTarget
		candidate dupe.TrackerCandidate
	}{
		{
			name: "file count with different size",
			target: api.TrackerDuplicateTarget{
				Names:     []string{"Example.Release.2026.1080p.WEB-DL-GRP"},
				SizeBytes: 1_000,
				FileNames: []string{"video.mkv", "subtitle.srt"},
			},
			candidate: dupe.TrackerCandidate{
				Name:      "Example.Release.Renamed.2026.1080p.WEB-DL-OTHER",
				SizeBytes: 900,
				SizeKnown: true,
				FileCount: 2,
			},
		},
		{
			name: "file count with unknown size",
			target: api.TrackerDuplicateTarget{
				Names:     []string{"Example.Release.2026.1080p.WEB-DL-GRP"},
				SizeBytes: 1_000,
				FileNames: []string{"video.mkv", "subtitle.srt"},
			},
			candidate: dupe.TrackerCandidate{
				Name:      "Example.Release.Renamed.2026.1080p.WEB-DL-OTHER",
				FileCount: 2,
			},
		},
		{
			name: "disc with different size",
			target: api.TrackerDuplicateTarget{
				Names:     []string{"Example.Release.2026.COMPLETE.UHD.BLURAY-GRP"},
				Type:      "DISC",
				SizeBytes: 1_000,
			},
			candidate: dupe.TrackerCandidate{
				Name:      "Example.Release.Renamed.2026.COMPLETE.UHD.BLURAY-OTHER",
				SizeBytes: 900,
				SizeKnown: true,
			},
		},
		{
			name: "disc with unknown size",
			target: api.TrackerDuplicateTarget{
				Names:     []string{"Example.Release.2026.COMPLETE.UHD.BLURAY-GRP"},
				Type:      "DISC",
				SizeBytes: 1_000,
			},
			candidate: dupe.TrackerCandidate{
				Name: "Example.Release.Renamed.2026.COMPLETE.UHD.BLURAY-OTHER",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			evaluation := dupe.Evaluate(
				test.target,
				[]dupe.TrackerCandidate{test.candidate},
				*policy,
				dupe.SearchEvidence{Complete: true, WorkScope: dupe.WorkScopeProviderID},
			)
			if evaluation.Blocks || evaluation.RequiresAction || evaluation.Candidates[0].Relation != api.DupeRelationCoexists {
				t.Fatalf("fallback identity evaluation = %#v", evaluation)
			}
		})
	}
}

func TestFileStemNameAloneDoesNotEstablishLiteralIdentity(t *testing.T) {
	t.Parallel()

	policy := Profile().DupePolicy
	if policy == nil {
		t.Fatal("DVL duplicate policy is missing")
	}
	target := api.TrackerDuplicateTarget{
		Names:     []string{"Generated.Upload.Name.2026-GRP"},
		SizeBytes: 1_000,
		FileNames: []string{"Original.Release.Name.2026-GRP.mkv"},
	}
	candidate := dupe.TrackerCandidate{
		Name:      "Original.Release.Name.2026-GRP",
		SizeBytes: 1_000,
		SizeKnown: true,
	}

	evaluation := dupe.Evaluate(
		target,
		[]dupe.TrackerCandidate{candidate},
		*policy,
		dupe.SearchEvidence{Complete: true, WorkScope: dupe.WorkScopeProviderID},
	)
	if evaluation.Blocks || evaluation.RequiresAction || evaluation.Candidates[0].Relation != api.DupeRelationCoexists {
		t.Fatalf("file-stem-only evaluation = %#v", evaluation)
	}
}
