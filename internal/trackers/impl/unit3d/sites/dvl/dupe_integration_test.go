// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package dvl

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/autobrr/upbrr/internal/config"
	"github.com/autobrr/upbrr/internal/trackers"
	"github.com/autobrr/upbrr/internal/trackers/dupe"
	"github.com/autobrr/upbrr/internal/trackers/impl/unit3d"
	"github.com/autobrr/upbrr/pkg/api"
)

type dvlRoundTripFunc func(*http.Request) (*http.Response, error)

func (f dvlRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestDVLAdapterSearchAndLiteralIdentityPolicyOffline(t *testing.T) {
	const apiKey = "synthetic-dvl-api-key"
	calls := 0
	client := &http.Client{Transport: dvlRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != http.MethodGet {
			t.Fatalf("method = %q, want GET", req.Method)
		}
		if req.URL.Scheme != "https" || req.URL.Host != "dreadvault.org" || req.URL.Path != "/api/torrents/filter" {
			t.Fatalf("request URL = %q", req.URL.String())
		}
		wantQuery := map[string]string{
			"tmdbId":       "1234567",
			"categories[]": "1",
			"name":         "",
			"perPage":      "100",
			"page":         "1",
		}
		query := req.URL.Query()
		if len(query) != len(wantQuery) {
			t.Fatalf("query = %#v", query)
		}
		for key, want := range wantQuery {
			values, ok := query[key]
			if !ok || len(values) != 1 || values[0] != want {
				t.Fatalf("query[%q] = %#v, want %q", key, values, want)
			}
		}
		if got := req.Header.Get("Authorization"); got != "Bearer "+apiKey {
			t.Fatal("Authorization header mismatch")
		}
		if got := req.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("Accept = %q", got)
		}
		if got := req.Header.Get("User-Agent"); got != "upbrr" {
			t.Fatalf("User-Agent = %q", got)
		}

		body := `{"data":[{"id":101,"attributes":{"name":"Example.Release.Renamed.2026.1080p.WEB-DL-OTHER","size":1000,"files":[{"name":"release/EXAMPLE.RELEASE.2026.MKV"}],"details_link":"https://dreadvault.org/torrents/101","download_link":"https://dreadvault.org/download/101","type":"WEBDL","resolution":"1080p","tmdb_id":1234567}},{"id":102,"attributes":{"name":"Example.Release.2026.1080p.WEB-DL-OTHER","size":900,"files":[{"name":"Example.Release.2026.1080p.WEB-DL-OTHER.mkv"}],"details_link":"https://dreadvault.org/torrents/102","download_link":"https://dreadvault.org/download/102","type":"WEBDL","resolution":"1080p","tmdb_id":1234567}}],"links":{"next":null}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})}

	definition := unit3d.NewWithProfile(Profile())
	registry := trackers.NewRegistry()
	if err := registry.Register(definition); err != nil {
		t.Fatalf("register DVL definition: %v", err)
	}
	adapter := dupe.NewAdapter(
		definition,
		"DVL",
		config.Config{Trackers: config.TrackersConfig{Trackers: map[string]config.TrackerConfig{
			"DVL": {APIKey: apiKey},
		}}},
		client,
		api.NopLogger{},
		registry,
	)

	result := adapter.Search(t.Context(), api.DuplicateSubject{
		SourcePath:  "/synthetic/Example.Release.2026",
		ReleaseName: "Example.Release.2026.1080p.WEB-DL-GRP",
		Identity: api.ExternalIdentity{
			TMDBID:   1234567,
			Category: api.CanonicalCategoryMovie,
		},
	})
	if result.Disposition() != dupe.DispositionResolved || result.Cause() != nil {
		t.Fatalf("adapter result: disposition=%v code=%q cause=%v", result.Disposition(), result.Code(), result.Cause())
	}
	if calls != 1 {
		t.Fatalf("HTTP calls = %d, want exactly one search request", calls)
	}

	evidence := result.SearchEvidence()
	if !evidence.Complete || !evidence.EffectiveComplete() || evidence.WorkScope != dupe.WorkScopeProviderID ||
		evidence.Pages != 1 || evidence.Scope != "work_category" || evidence.WrongWorkCount != 0 || len(evidence.Warnings) != 0 {
		t.Fatalf("search evidence = %#v", evidence)
	}
	entries := result.Entries()
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	if entries[0].ID != "101" || entries[0].Name != "Example.Release.Renamed.2026.1080p.WEB-DL-OTHER" ||
		!entries[0].SizeKnown || entries[0].SizeBytes != 1000 || entries[0].FileCount != 1 ||
		len(entries[0].Files) != 1 || entries[0].Files[0] != "release/EXAMPLE.RELEASE.2026.MKV" ||
		entries[0].Type != "WEBDL" || entries[0].CanonicalType != "WEBDL" || entries[0].Res != "1080p" {
		t.Fatalf("exact entry = %#v", entries[0])
	}

	candidates := make([]dupe.TrackerCandidate, 0, len(entries))
	for _, entry := range entries {
		candidates = append(candidates, dupe.NormalizeCandidate(entry, "DVL"))
	}
	policy, ok := registry.LookupDupePolicy("DVL")
	if !ok || !policy.LiteralIdentityOnly {
		t.Fatalf("DVL policy = %#v, found=%t", policy, ok)
	}
	evaluation := dupe.Evaluate(api.TrackerDuplicateTarget{
		Names:      []string{"Example.Release.2026.1080p.WEB-DL-GRP"},
		Type:       "WEB-DL",
		Resolution: "1080p",
		SizeBytes:  1000,
		FileNames:  []string{"Example.Release.2026.mkv"},
	}, candidates, policy, evidence)
	if !evaluation.Complete || !evaluation.Blocks || evaluation.RequiresAction || len(evaluation.Candidates) != 2 {
		t.Fatalf("evaluation = %#v", evaluation)
	}
	if got := evaluation.Candidates[0]; got.Candidate.ID != "101" || got.Relation != api.DupeRelationExactDuplicate ||
		got.WinningRule != "dvl/duplicate/v1/literal_identity" || len(got.Reasons) != 1 || got.Reasons[0].Code != "exact_identity" {
		t.Fatalf("exact evaluation = %#v", got)
	}
	if got := evaluation.Candidates[1]; got.Candidate.ID != "102" || got.Relation != api.DupeRelationCoexists ||
		got.WinningRule != "dvl/duplicate/v1/literal_identity" || len(got.Reasons) != 1 || got.Reasons[0].Code != "literal_identity_differs" {
		t.Fatalf("related evaluation = %#v", got)
	}
}
