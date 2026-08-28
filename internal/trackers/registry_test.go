// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package trackers

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/autobrr/upbrr/internal/config"
	"github.com/autobrr/upbrr/pkg/api"
)

type stubDefinition struct {
	name string
}

type stubAuthDefinition struct{ stubDefinition }

func (stubAuthDefinition) AuthSessionResolver() AuthSessionResolver {
	return func(context.Context, config.TrackerConfig, string, api.TrackerAuthLoginRequest) error { return nil }
}

func (s stubDefinition) Name() string {
	return s.name
}

func (stubDefinition) UploadContentMode() UploadContentMode { return UploadContentModeDescription }

func (stubDefinition) DefaultBaseURL() string { return "https://tracker.example.invalid" }

func testTrackerFamily(name string) Family {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "AITHER", "BLU", "HHD", "LST", "OE", "RF", "RHD", "STC":
		return FamilyUnit3D
	default:
		return FamilyStandalone
	}
}

func testImageHostPolicyForTracker(name string) *ImageHostPolicy {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "GPW":
		return &ImageHostPolicy{AllowedHosts: []string{"kshare", "pixhost", "pterclub", "ilikeshots", "imgbox"}}
	case "HDB":
		return &ImageHostPolicy{
			AllowedHosts:         []string{"hdb"},
			OwnedHosts:           []string{"hdb"},
			DisableWithoutRehost: true,
		}
	case "LST":
		return &ImageHostPolicy{
			ConditionalHost:        "lostimg",
			OwnedHosts:             []string{"lostimg"},
			EnableWithImageHosting: true,
		}
	case "OE":
		return &ImageHostPolicy{AllowedHosts: []string{"imgbox", "imgbb", "onlyimage", "ptscreens", "passtheimage"}}
	case "PTP":
		return &ImageHostPolicy{AllowedHosts: []string{"pixhost", "imgbb", "onlyimage", "ptscreens", "passtheimage"}}
	case "RF":
		return &ImageHostPolicy{
			ConditionalHost:        "reelflix",
			OwnedHosts:             []string{"reelflix"},
			EnableWithImageHosting: true,
		}
	case "SAM":
		return &ImageHostPolicy{
			ConditionalHost:        "samaritano",
			OwnedHosts:             []string{"samaritano"},
			EnableWithImageHosting: true,
		}
	case "STC":
		return &ImageHostPolicy{AllowedHosts: []string{"imgbox", "imgbb"}}
	default:
		return nil
	}
}

func (s stubDefinition) Prepare(ctx context.Context, input PreparationInput) (TrackerPlan, *PreparationFailure) {
	return prepareTestDefinition(ctx, input, s)
}

//nolint:unparam // Error is required by the adapter submission callback contract.
func (s stubDefinition) submit(context.Context, PreparationInput) (api.UploadSummary, error) {
	return api.UploadSummary{Uploaded: 1}, nil
}

func TestRegistryRegisterLookup(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(stubDefinition{name: "Blu"}); err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}
	if _, ok := registry.Lookup("BLU"); !ok {
		t.Fatalf("expected lookup to succeed")
	}
	if _, ok := registry.Lookup("blu"); !ok {
		t.Fatalf("expected lookup to be case-insensitive")
	}
}

func TestRegistryRegistersAuthResolver(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(stubAuthDefinition{stubDefinition{name: "AUTH"}}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, ok := registry.LookupAuthSessionResolver("auth"); !ok {
		t.Fatal("expected auth resolver")
	}
}

func TestRegistryRegisterDuplicate(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(stubDefinition{name: "BLU"}); err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}
	if err := registry.Register(stubDefinition{name: "blu"}); err == nil {
		t.Fatalf("expected duplicate register error")
	}
}

func TestRegistryRegisterDescriptorRejectsMismatchedName(t *testing.T) {
	registry := NewRegistry()
	err := registry.RegisterDescriptor(Descriptor{Name: "BLU", Definition: stubDefinition{name: "AITHER"}})
	if err == nil {
		t.Fatal("expected mismatched descriptor name error")
	}
}

func TestRegistryDiscoversCapabilitiesAndSortsNames(t *testing.T) {
	registry := NewRegistry()
	for _, definition := range []Definition{
		stubDefinition{name: "ZNTH"},
		stubDefinition{name: "BLU"},
	} {
		if err := registry.Register(definition); err != nil {
			t.Fatalf("register: %v", err)
		}
	}
	if got, want := registry.Names(), []string{"BLU", "ZNTH"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("names = %v, want %v", got, want)
	}
}

func TestRegistryRuleCapability(t *testing.T) {
	registry := NewRegistry()
	rules := RuleSet{RequireMovieOnly: true}
	if err := registry.RegisterDescriptor(Descriptor{
		Name:       "BLU",
		Definition: stubDefinition{name: "BLU"},
		Rules:      &rules,
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	got, ok := registry.LookupRules("blu")
	if !ok || !got.RequireMovieOnly {
		t.Fatalf("rules = %#v, ok=%t", got, ok)
	}
}

func TestValidateDupePolicyRequiresEvidenceIDForAutomaticRules(t *testing.T) {
	t.Parallel()

	policy := DupePolicy{
		ID:             "example/duplicate/v2",
		SlotDimensions: []DupeDimension{DupeDimensionResolution},
	}
	if err := validateDupePolicy(policy); err == nil || !strings.Contains(err.Error(), "evidence ID") {
		t.Fatalf("missing traceability error = %v", err)
	}
	policy.EvidenceID = "example-rules"
	if err := validateDupePolicy(policy); err != nil {
		t.Fatalf("evidence-backed policy rejected: %v", err)
	}
}

func TestValidateDupePolicyLiteralIdentityOnlyRequiresEvidence(t *testing.T) {
	t.Parallel()

	for _, id := range []string{
		"example/duplicate/v1",
		"example/duplicate-compat/v1",
	} {
		t.Run(id, func(t *testing.T) {
			policy := DupePolicy{
				ID:                  id,
				LiteralIdentityOnly: true,
				SearchScope:         DupeSearchScope{MaxPages: 100},
			}
			if err := validateDupePolicy(policy); err == nil || !strings.Contains(err.Error(), "evidence ID") {
				t.Fatalf("missing literal-identity traceability error = %v", err)
			}

			policy.EvidenceID = "example-literal-identity-policy"
			if err := validateDupePolicy(policy); err != nil {
				t.Fatalf("evidence-backed literal policy rejected: %v", err)
			}
		})
	}
}

func TestDupePolicyOmitsDisabledLiteralIdentityModeFromSerializedPolicy(t *testing.T) {
	t.Parallel()

	disabled, err := json.Marshal(DupePolicy{ID: "example/duplicate/v1"})
	if err != nil {
		t.Fatalf("marshal disabled literal policy: %v", err)
	}
	if strings.Contains(string(disabled), "LiteralIdentityOnly") {
		t.Fatalf("disabled literal mode changed serialized policy: %s", disabled)
	}

	enabled, err := json.Marshal(DupePolicy{ID: "example/duplicate/v1", LiteralIdentityOnly: true})
	if err != nil {
		t.Fatalf("marshal enabled literal policy: %v", err)
	}
	if !strings.Contains(string(enabled), `"LiteralIdentityOnly":true`) {
		t.Fatalf("enabled literal mode missing from serialized policy: %s", enabled)
	}
}

func TestValidateDupePolicyLiteralIdentityOnlyRejectsComparisonOverlays(t *testing.T) {
	t.Parallel()

	base := func() DupePolicy {
		return DupePolicy{
			ID:                  "example/duplicate/v1",
			EvidenceID:          "example-literal-identity-policy",
			LiteralIdentityOnly: true,
			SearchScope:         DupeSearchScope{MaxPages: 100},
		}
	}
	if err := validateDupePolicy(base()); err != nil {
		t.Fatalf("plain literal policy rejected: %v", err)
	}

	tests := []struct {
		name      string
		configure func(*DupePolicy)
	}{
		{"slot dimension", func(policy *DupePolicy) {
			policy.SlotDimensions = []DupeDimension{DupeDimensionResolution}
		}},
		{"optional slot dimension", func(policy *DupePolicy) {
			policy.OptionalSlotDimensions = []DupeDimension{DupeDimensionProvider}
		}},
		{"complete slot dimension", func(policy *DupePolicy) {
			policy.CompleteSlotDimensions = []DupeDimension{DupeDimensionHDR}
		}},
		{"required dimension", func(policy *DupePolicy) {
			policy.RequiredDimensions = []DupeDimension{DupeDimensionCodec}
		}},
		{"general coexistence suppression", func(policy *DupePolicy) {
			policy.SuppressGeneralCoexistence = []DupeDimension{DupeDimensionResolution}
		}},
		{"HDR slot mode", func(policy *DupePolicy) {
			policy.HDRSlotMode = DupeHDRSlotModeGeneric
		}},
		{"HDR partial mode", func(policy *DupePolicy) {
			policy.HDRPartialMode = DupeHDRPartialGenericMarker
		}},
		{"HDR compatibility", func(policy *DupePolicy) {
			policy.HDRCompatibilityMode = DupeHDRCompatibilityDirectional
		}},
		{"DV profile requirement", func(policy *DupePolicy) {
			policy.RequireDolbyVisionProfile = true
		}},
		{"DV profile slot", func(policy *DupePolicy) {
			policy.DolbyVisionProfile5Slot = true
		}},
		{"slot contradiction review", func(policy *DupePolicy) {
			policy.SlotContradictionsRequireManualReview = true
		}},
		{"coexistence rule", func(policy *DupePolicy) {
			policy.CoexistenceRules = []DupeRule{{ID: "example-coexists", Relation: string(api.DupeRelationCoexists)}}
		}},
		{"precedence rule", func(policy *DupePolicy) {
			policy.PrecedenceRules = []DupeRule{{ID: "example-precedence", Relation: string(api.DupeRelationExistingPreferred)}}
		}},
		{"manual review rule", func(policy *DupePolicy) {
			policy.ManualReviewRules = []DupeRule{{ID: "example-review", RequiresManualStep: true}}
		}},
		{"set rule", func(policy *DupePolicy) {
			policy.SetRules = []DupeSetRule{{
				ID: "example-set",
				TargetPredicates: []DupeSetPredicate{{
					Dimension: DupeDimensionResolution,
					Values:    []string{"1080p"},
				}},
				CandidatePredicates: []DupeSetPredicate{{
					Dimension: DupeDimensionResolution,
					Values:    []string{"1080p"},
				}},
				Capacity: 1,
			}}
		}},
		{"size variance", func(policy *DupePolicy) {
			policy.SizeVariancePercent = 20
		}},
		{"size resolution scope", func(policy *DupePolicy) {
			policy.SizeVarianceResolutions = []string{"1080p"}
		}},
		{"size type scope", func(policy *DupePolicy) {
			policy.SizeVarianceTypes = []string{"ENCODE"}
		}},
		{"trump override", func(policy *DupePolicy) {
			policy.TrumpableOverridesSlot = true
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			policy := base()
			test.configure(&policy)
			if err := validateDupePolicy(policy); err == nil || !strings.Contains(err.Error(), "literal identity") {
				t.Fatalf("mixed literal policy error = %v", err)
			}
		})
	}
}

func TestCompatibilityDupePolicyAddsNoSyntheticFallback(t *testing.T) {
	t.Parallel()

	policy := compatibilityDupePolicy("EXAMPLE")
	if len(policy.ManualReviewRules) != 0 || len(policy.SlotDimensions) != 0 || len(policy.PrecedenceRules) != 0 {
		t.Fatalf("compatibility policy = %#v", policy)
	}
}

func TestCloneDupePolicyPreservesRuleConditions(t *testing.T) {
	t.Parallel()

	source := DupePolicy{
		PrecedenceRules: []DupeRule{{
			ID: "example",
			Conditions: []DupeCondition{{
				Dimension:       DupeDimensionType,
				TargetValues:    []string{"WEB-DL"},
				CandidateValues: []string{"WEBRip"},
			}},
		}},
		SetRules: []DupeSetRule{{
			ID: "example-set",
			TargetPredicates: []DupeSetPredicate{{
				Dimension: DupeDimensionResolution,
				Values:    []string{"1080p"},
			}},
			CandidatePredicates: []DupeSetPredicate{{
				Dimension:      DupeDimensionHDR,
				ExcludedValues: []string{"sdr"},
			}},
			CapacityOverrides: []DupeSetCapacityOverride{{
				Capacity: 1,
				CandidatePredicates: []DupeSetPredicate{{
					Dimension: DupeDimensionResolution,
					Values:    []string{"2160p"},
				}},
			}},
		}},
	}
	policy := cloneDupePolicy(source)
	condition := policy.PrecedenceRules[0].Conditions[0]
	if condition.Dimension != DupeDimensionType || len(condition.TargetValues) != 1 ||
		len(condition.CandidateValues) != 1 {
		t.Fatalf("cloned condition = %#v", condition)
	}
	policy.SetRules[0].TargetPredicates[0].Values[0] = "720p"
	policy.SetRules[0].CandidatePredicates[0].ExcludedValues[0] = "hdr10"
	policy.SetRules[0].CapacityOverrides[0].CandidatePredicates[0].Values[0] = "1080p"
	if source.SetRules[0].TargetPredicates[0].Values[0] != "1080p" ||
		source.SetRules[0].CandidatePredicates[0].ExcludedValues[0] != "sdr" ||
		source.SetRules[0].CapacityOverrides[0].CandidatePredicates[0].Values[0] != "2160p" {
		t.Fatalf("set rule clone mutated source: %#v", source.SetRules[0])
	}
}
