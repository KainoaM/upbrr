// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package dvl

import (
	"strconv"
	"strings"
	"testing"

	"github.com/autobrr/upbrr/internal/languageutil"
	"github.com/autobrr/upbrr/internal/metadata"
	"github.com/autobrr/upbrr/internal/trackers"
	"github.com/autobrr/upbrr/internal/trackers/impl/unit3d"
	"github.com/autobrr/upbrr/pkg/api"
)

func TestDVLStructuredReleaseNamePolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		request   api.ReleaseNameRequest
		languages []string
		want      string
	}{
		{
			name: "DVD rip moves resolution format audio and encode without touching title tokens",
			request: api.ReleaseNameRequest{
				Category:    "MOVIE",
				Type:        "DVDRIP",
				Title:       "DD 2.0 x264 DVDRip Tales",
				Year:        2001,
				Resolution:  "480p",
				Source:      "PAL DVD",
				VideoEncode: "x264",
				Audio:       "DD 2.0",
				Tag:         "-GRP",
			},
			want: "DD 2.0 x264 DVDRip Tales 2001 480p DVDRip DD 2.0 x264-GRP",
		},
		{
			name: "DVD rip falls back to video codec",
			request: api.ReleaseNameRequest{
				Category:   "MOVIE",
				Type:       "DVDRIP",
				Title:      "Example Movie",
				Year:       2001,
				Resolution: "480p",
				Source:     "PAL DVD",
				VideoCodec: "XviD",
				Audio:      "DD 2.0",
				Tag:        "-GRP",
			},
			want: "Example Movie 2001 480p DVDRip DD 2.0 XviD-GRP",
		},
		{
			name: "DVD rip keeps a dubbed audio prefix with its audio",
			request: api.ReleaseNameRequest{
				Category:    "MOVIE",
				Type:        "DVDRIP",
				Title:       "Example Movie",
				Year:        2001,
				Resolution:  "480p",
				Source:      "PAL DVD",
				VideoEncode: "x264",
				Audio:       "Dubbed DD 2.0",
				Tag:         "-GRP",
			},
			want: "Example Movie 2001 480p DVDRip Dubbed DD 2.0 x264-GRP",
		},
		{
			name: "TV DVD rip keeps a dual audio suffix with its audio",
			request: api.ReleaseNameRequest{
				Category:    "TV",
				Type:        "DVDRIP",
				Title:       "Example Show",
				Season:      "S01",
				Resolution:  "576p",
				Source:      "PAL DVD",
				VideoEncode: "x264",
				Audio:       "DD 2.0 Dual-Audio",
				Tag:         "-GRP",
			},
			want: "Example Show S01 576p DVDRip DD 2.0 Dual-Audio x264-GRP",
		},
		{
			name: "DVD rip keeps a marker only audio cluster before the codec",
			request: api.ReleaseNameRequest{
				Category:    "MOVIE",
				Type:        "DVDRIP",
				Title:       "Example Movie",
				Year:        2001,
				Resolution:  "480p",
				Source:      "PAL DVD",
				VideoEncode: "x264",
				Audio:       "Dubbed",
				Tag:         "-GRP",
			},
			want: "Example Movie 2001 480p DVDRip Dubbed x264-GRP",
		},
		{
			name: "DVD rip without resolution evidence uses its format anchor",
			request: api.ReleaseNameRequest{
				Category:    "MOVIE",
				Type:        "DVDRIP",
				Title:       "Example Movie",
				Year:        2001,
				Source:      "PAL DVD",
				VideoEncode: "x264",
				Audio:       "DD 2.0",
				Tag:         "-GRP",
			},
			languages: []string{"Japanese"},
			want:      "Example Movie 2001 JAPANESE DVDRip DD 2.0 x264-GRP",
		},
		{
			name: "DVD encode becomes DVD rip and omits edition and repack",
			request: api.ReleaseNameRequest{
				Category:    "MOVIE",
				Type:        "ENCODE",
				Title:       "Uncut Shadows",
				Year:        2001,
				Resolution:  "480p",
				Source:      "PAL DVD",
				Edition:     "Uncut",
				Repack:      "REPACK",
				VideoEncode: "x264",
				Audio:       "DD 2.0",
				Tag:         "-GRP",
			},
			want: "Uncut Shadows 2001 480p DVDRip DD 2.0 x264-GRP",
		},
		{
			name: "DVD encode without resolution keeps canonical source and edition",
			request: api.ReleaseNameRequest{
				Category:    "MOVIE",
				Type:        "ENCODE",
				Title:       "Example Movie",
				Year:        2001,
				Source:      "DVD",
				Edition:     "Uncut",
				VideoEncode: "x264",
				Audio:       "DD 2.0",
				Tag:         "-GRP",
			},
			want: "Example Movie 2001 Uncut DVD DD 2.0 x264-GRP",
		},
		{
			name: "DVD disc derives PAL using the DVD system role",
			request: api.ReleaseNameRequest{
				Category:   "MOVIE",
				Type:       "DISC",
				DiscType:   "DVD",
				Title:      "Example Movie",
				Year:       1997,
				Resolution: "576p",
				Source:     "DVD",
				DVDSize:    "DVD5",
				Audio:      "DD 5.1",
				Tag:        "-GRP",
			},
			want: "Example Movie 1997 PAL DVD5 DD 5.1-GRP",
		},
		{
			name: "DVD remux keeps resolution and puts foreign language after year",
			request: api.ReleaseNameRequest{
				Category:   "MOVIE",
				Type:       "REMUX",
				Title:      "Example Movie",
				Year:       2001,
				Resolution: "480i",
				Source:     "DVD",
				VideoCodec: "MPEG-2",
				Audio:      "DD 5.1",
				Tag:        "-GRP",
			},
			languages: []string{"Japanese"},
			want:      "Example Movie 2001 JAPANESE 480i DVD REMUX DD 5.1-GRP",
		},
		{
			name: "DVD remux puts foreign language before edition after year",
			request: api.ReleaseNameRequest{
				Category:   "MOVIE",
				Type:       "REMUX",
				Title:      "Example Movie",
				Year:       2001,
				Resolution: "480i",
				Source:     "DVD",
				Edition:    "Director Cut",
				Audio:      "DD 5.1",
				Tag:        "-GRP",
			},
			languages: []string{"Japanese"},
			want:      "Example Movie 2001 JAPANESE Director Cut 480i DVD REMUX DD 5.1-GRP",
		},
		{
			name: "TV DVD remux puts foreign language after season",
			request: api.ReleaseNameRequest{
				Category:   "TV",
				Type:       "REMUX",
				Title:      "Example Show",
				Year:       2024,
				AltTitle:   "AKA Alt Show",
				Season:     "S01",
				Resolution: "576p",
				Source:     "PAL DVD",
				VideoCodec: "MPEG-2",
				Audio:      "DD 2.0",
				SearchYear: "2024",
				Tag:        "-GRP",
			},
			languages: []string{"Japanese"},
			want:      "Example Show AKA Alt Show 2024 S01 JAPANESE 576p DVD REMUX DD 2.0-GRP",
		},
		{
			name: "foreign language is a separate marker even when title has matching text",
			request: api.ReleaseNameRequest{
				Category:    "MOVIE",
				Type:        "ENCODE",
				Title:       "Uncut JAPANESE 480p DVD Tales",
				Year:        2001,
				Resolution:  "480p",
				Source:      "BluRay",
				VideoEncode: "x264",
				Audio:       "DD 5.1",
				Tag:         "-GRP",
			},
			languages: []string{"Japanese"},
			want:      "Uncut JAPANESE 480p DVD Tales 2001 JAPANESE 480p BluRay DD 5.1 x264-GRP",
		},
		{
			name: "English and Japanese audio add a dual audio token to a non disc name",
			request: api.ReleaseNameRequest{
				Category:    "MOVIE",
				Type:        "ENCODE",
				Title:       "Example Movie",
				Year:        2001,
				Resolution:  "1080p",
				Source:      "BluRay",
				VideoEncode: "x264",
				Audio:       "DD 5.1",
				Tag:         "-GRP",
			},
			languages: []string{"Japanese", "English"},
			want:      "Example Movie 2001 1080p BluRay Dual-Audio DD 5.1 x264-GRP",
		},
		{
			name: "BDMV omits foreign language marker",
			request: api.ReleaseNameRequest{
				Category:   "MOVIE",
				Type:       "DISC",
				DiscType:   "BDMV",
				Title:      "Example Movie",
				Year:       2001,
				Resolution: "1080p",
				Source:     "BluRay",
				VideoCodec: "AVC",
				Audio:      "DD 5.1",
				Tag:        "-GRP",
			},
			languages: []string{"Japanese"},
			want:      "Example Movie 2001 1080p Blu-ray AVC DD 5.1-GRP",
		},
		{
			name: "first linguistic language marker skips non linguistic values",
			request: api.ReleaseNameRequest{
				Category:    "MOVIE",
				Type:        "ENCODE",
				Title:       "Example Movie",
				Year:        2001,
				Resolution:  "1080p",
				Source:      "BluRay",
				VideoEncode: "x264",
				Audio:       "DD 5.1",
				Tag:         "-GRP",
			},
			languages: []string{"zxx", "no linguistic content", "und", "undetermined", "Norwegian Nynorsk"},
			want:      "Example Movie 2001 NORWEGIAN NYNORSK 1080p BluRay DD 5.1 x264-GRP",
		},
		{
			name: "all non linguistic audio values use the ZXX marker",
			request: api.ReleaseNameRequest{
				Category:    "MOVIE",
				Type:        "ENCODE",
				Title:       "Example Movie",
				Year:        2001,
				Resolution:  "1080p",
				Source:      "BluRay",
				VideoEncode: "x264",
				Audio:       "DD 5.1",
				Tag:         "-GRP",
			},
			languages: []string{"zxx", "no linguistic content", "und", "undetermined"},
			want:      "Example Movie 2001 ZXX 1080p BluRay DD 5.1 x264-GRP",
		},
		{
			name: "yearless DVD remux places foreign language before source",
			request: api.ReleaseNameRequest{
				Category:   "MOVIE",
				Type:       "REMUX",
				Title:      "Example Movie",
				Resolution: "576p",
				Source:     "PAL DVD",
				Audio:      "DD 2.0",
				Tag:        "-GRP",
			},
			languages: []string{"Japanese"},
			want:      "Example Movie JAPANESE 576p DVD REMUX DD 2.0-GRP",
		},
		{
			name: "DVD disc omits foreign language before region and system",
			request: api.ReleaseNameRequest{
				Category:   "MOVIE",
				Type:       "DISC",
				DiscType:   "DVD",
				Title:      "Example Movie",
				Year:       1977,
				Resolution: "480p",
				Source:     "DVD",
				Region:     "USA",
				DVDSize:    "DVD9",
				Audio:      "LPCM 2.0",
			},
			languages: []string{"Japanese"},
			want:      "Example Movie 1977 USA NTSC DVD9 LPCM 2.0",
		},
		{
			name: "DVD disc omits foreign language before visible PAL system and source",
			request: api.ReleaseNameRequest{
				Category:   "MOVIE",
				Type:       "DISC",
				DiscType:   "DVD",
				Title:      "Example Movie",
				Year:       1977,
				Resolution: "576p",
				Source:     "PAL DVD",
				Audio:      "LPCM 2.0",
				Tag:        "-GRP",
			},
			languages: []string{"Japanese"},
			want:      "Example Movie 1977 PAL DVD LPCM 2.0-GRP",
		},
		{
			name: "DVD disc omits foreign language before visible NTSC system and source",
			request: api.ReleaseNameRequest{
				Category:   "MOVIE",
				Type:       "DISC",
				DiscType:   "DVD",
				Title:      "Example Movie",
				Year:       1977,
				Resolution: "480p",
				Source:     "NTSC DVD",
				Audio:      "LPCM 2.0",
				Tag:        "-GRP",
			},
			languages: []string{"Japanese"},
			want:      "Example Movie 1977 NTSC DVD LPCM 2.0-GRP",
		},
		{
			name: "TV DVD encode retains episode attachment while becoming a rip",
			request: api.ReleaseNameRequest{
				Category:     "TV",
				Type:         "ENCODE",
				Title:        "Example Show",
				Season:       "S04",
				Episode:      "E03",
				EpisodeTitle: "Title",
				Resolution:   "480p",
				Source:       "NTSC DVD",
				VideoEncode:  "x264",
				Audio:        "DD 2.0",
				Tag:          "-GRP",
			},
			languages: []string{"German"},
			want:      "Example Show S04E03 Title GERMAN 480p DVDRip DD 2.0 x264-GRP",
		},
		{
			name: "DVD encode preserves repack text in the title while omitting repack component",
			request: api.ReleaseNameRequest{
				Category:    "MOVIE",
				Type:        "ENCODE",
				Title:       "REPACK3 Shadows",
				Year:        1994,
				Resolution:  "480p",
				Source:      "DVD",
				Repack:      "REPACK3",
				VideoEncode: "x264",
				Audio:       "DD 5.1",
				Tag:         "-GRP",
			},
			want: "REPACK3 Shadows 1994 480p DVDRip DD 5.1 x264-GRP",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			subject := dvlGeneratedSubject(t, test.request, test.languages)
			if got := dvlReviewedName(t, subject, nil); got != test.want {
				t.Fatalf("reviewed name = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDVLStructuredReleaseNamePolicyPreservesOpaqueAndManualNames(t *testing.T) {
	t.Parallel()

	request := api.ReleaseNameRequest{
		Category:    "MOVIE",
		Type:        "DVDRIP",
		Title:       "Example Movie",
		Year:        2001,
		Resolution:  "480p",
		Source:      "PAL DVD",
		VideoEncode: "x264",
		Audio:       "DD 2.0",
		Tag:         "-GRP",
	}
	subject := dvlGeneratedSubject(t, request, []string{"Japanese"})
	override := "Manual DVL Name-GRP"
	if got := dvlReviewedName(t, subject, &override); got != override {
		t.Fatalf("opaque override = %q, want %q", got, override)
	}
	missingDocument := subject
	missingDocument.GeneratedName = nil
	missingDocument.ReleaseName = "Opaque Name Without Components-GRP"
	if got := dvlReviewedName(t, missingDocument, nil); got != missingDocument.ReleaseName {
		t.Fatalf("missing document name = %q, want %q", got, missingDocument.ReleaseName)
	}

	manual := subject
	manual.GeneratedName = manual.GeneratedName.Clone()
	for index := range manual.GeneratedName.Components {
		if manual.GeneratedName.Components[index].Role == api.NameRoleSource {
			manual.GeneratedName.Components[index].Present = true
			manual.GeneratedName.Components[index].Manual = true
		}
	}
	manual.ReleaseName = manual.GeneratedName.Render().Name
	if got := dvlReviewedName(t, manual, nil); !strings.Contains(got, "PAL DVD") {
		t.Fatalf("manual source was changed: %q", got)
	}

	manualDisc := dvlGeneratedSubject(t, api.ReleaseNameRequest{
		Category:   "MOVIE",
		Type:       "DISC",
		DiscType:   "DVD",
		Title:      "Example Movie",
		Year:       1997,
		Resolution: "576p",
		Source:     "DVD",
		DVDSize:    "DVD9",
		Audio:      "DD 5.1",
		Tag:        "-GRP",
	}, []string{"Japanese"})
	markDVLGeneratedComponentManual(t, manualDisc.GeneratedName, api.NameRoleSource, true)
	markDVLGeneratedComponentManual(t, manualDisc.GeneratedName, api.NameRoleDVDSystem, false)
	manualDisc.ReleaseName = manualDisc.GeneratedName.Render().Name
	if got, want := dvlReviewedName(t, manualDisc, nil), "Example Movie 1997 DVD DVD9 DD 5.1-GRP"; got != want {
		t.Fatalf("manual DVD source name = %q, want %q", got, want)
	}

	manualRemux := dvlGeneratedSubject(t, api.ReleaseNameRequest{
		Category:   "MOVIE",
		Type:       "REMUX",
		Title:      "Example Movie",
		Resolution: "576p",
		Source:     "DVD",
		Audio:      "DD 2.0",
		Tag:        "-GRP",
	}, []string{"Japanese"})
	markDVLGeneratedComponentManual(t, manualRemux.GeneratedName, api.NameRoleSource, true)
	appendDVLManualComponent(manualRemux.GeneratedName, api.NameRoleDVDSystem)
	manualRemux.ReleaseName = manualRemux.GeneratedName.Render().Name
	if got, want := dvlReviewedName(t, manualRemux, nil), "Example Movie JAPANESE 576p DVD REMUX DD 2.0-GRP"; got != want {
		t.Fatalf("manual DVD remux name = %q, want %q", got, want)
	}

	manualResolution := dvlGeneratedSubject(t, request, []string{"Japanese"})
	markDVLGeneratedComponentManual(t, manualResolution.GeneratedName, api.NameRoleResolution, false)
	manualResolution.ReleaseName = manualResolution.GeneratedName.Render().Name
	if got, want := dvlReviewedName(t, manualResolution, nil), "Example Movie 2001 JAPANESE DVDRip DD 2.0 x264-GRP"; got != want {
		t.Fatalf("manual omitted resolution name = %q, want %q", got, want)
	}
}

func TestDVLProfileUsesStructuredPolicy(t *testing.T) {
	t.Parallel()

	policy := unit3d.NewWithProfile(Profile()).ReleaseNamePolicy()
	if policy.ID != "unit3d/dvl/v2" || policy.Structured == nil || policy.Resolver != nil {
		t.Fatalf("DVL policy = %#v, want structured unit3d/dvl/v2", policy)
	}
}

func TestDVLRipKeepsManualAudioMarkersWithAudio(t *testing.T) {
	for _, category := range []string{"MOVIE", "TV"} {
		for _, audio := range []string{"Dubbed DD 2.0", "DD 2.0 Dual-Audio", "Dubbed", "Dual-Audio", ""} {
			t.Run(category+"/"+audio, func(t *testing.T) {
				subject := dvlGeneratedSubject(t, api.ReleaseNameRequest{
					Category:    category,
					Type:        "DVDRIP",
					Title:       "Example Release",
					Year:        2001,
					Source:      "PAL DVD",
					Resolution:  "480p",
					Audio:       audio,
					VideoEncode: "x264",
					Tag:         "-GRP",
				}, nil)
				for index := range subject.GeneratedName.Components {
					component := &subject.GeneratedName.Components[index]
					if component.Role == api.NameRoleDubbed || component.Role == api.NameRoleDualAudio || component.Role == api.NameRoleVideoFormat {
						component.Manual = true
					}
				}
				want := "Example Release 2001 480p DVDRip " + audio + " x264-GRP"
				if category == "TV" {
					want = "Example Release 480p DVDRip " + audio + " x264-GRP"
				}
				want = strings.Join(strings.Fields(want), " ")
				if got := dvlReviewedName(t, subject, nil); got != want {
					t.Fatalf("manual audio marker name = %q, want %q", got, want)
				}
			})
		}
	}
}

func dvlGeneratedSubject(t *testing.T, request api.ReleaseNameRequest, languages []string) api.UploadSubject {
	t.Helper()

	result := metadata.BuildReleaseName(request, api.NopLogger{})
	if result.GeneratedName == nil {
		t.Fatal("BuildReleaseName did not produce a structured document")
	}
	return api.UploadSubject{
		ReleaseName:      result.Name,
		ReleaseNameNoTag: result.NameNoTag,
		GeneratedName:    result.GeneratedName,
		Release: api.ReleaseInfo{
			Category:   request.Category,
			Title:      request.Title,
			Year:       request.Year,
			Resolution: request.Resolution,
			Size:       request.DVDSize,
		},
		AlternateTitle: request.AltTitle,
		Type:           request.Type,
		DiscType:       request.DiscType,
		Source:         request.Source,
		Audio:          request.Audio,
		VideoCodec:     request.VideoCodec,
		VideoEncode:    request.VideoEncode,
		AudioLanguages: languages,
		SeasonStr:      request.Season,
		EpisodeStr:     request.Episode,
		EpisodeTitle:   request.EpisodeTitle,
		TVPack:         request.Category == "TV" && request.Episode == "",
	}
}

func dvlReviewedName(t *testing.T, subject api.UploadSubject, requested *string) string {
	t.Helper()

	prepared, failure := trackers.PrepareInputWithReleaseNamePolicy(trackers.PreparationInput{
		Tracker:             "DVL",
		Meta:                subject,
		RequestedUploadName: requested,
	}, unit3d.NewWithProfile(Profile()).ReleaseNamePolicy())
	if failure != nil {
		t.Fatal(failure)
	}
	name, err := prepared.ReviewedUploadName()
	if err != nil {
		t.Fatal(err)
	}
	return name
}

func markDVLGeneratedComponentManual(t *testing.T, document *api.ReleaseNameDocument, role api.ReleaseNameRole, present bool) {
	t.Helper()

	for index := range document.Components {
		component := &document.Components[index]
		if component.Role != role {
			continue
		}
		component.Manual = true
		component.Present = present
		return
	}
	t.Fatalf("generated name is missing %s", role)
}

func appendDVLManualComponent(document *api.ReleaseNameDocument, role api.ReleaseNameRole) {
	document.Components = append(document.Components, api.ReleaseNameComponent{
		Role:   role,
		Join:   " ",
		Manual: true,
	})
}

func TestDVLWikiNamingGuide20260930(t *testing.T) {
	t.Parallel()

	// Shared DVL naming vectors, projected to finalized pipeline facts.
	// Audio retains the shared Dual-Audio/Dubbed prefix so DVL must recompute its own token.
	tests := []struct {
		name             string
		request          api.ReleaseNameRequest
		languages        []string
		originalLanguage string
		channels         string
		commentary       bool
		hardcodedSubs    bool
		want             string
	}{
		{
			name: "lang-basic",
			request: api.ReleaseNameRequest{
				Category:    "MOVIE",
				Type:        "ENCODE",
				Title:       "Kuroko's Basketball",
				AltTitle:    "AKA Kuroko no Basket",
				Year:        2017,
				Resolution:  "1080p",
				Source:      "BluRay",
				VideoCodec:  "AVC",
				VideoEncode: "x264",
				Audio:       "DD+ 5.1",
				Tag:         "-Kitsune",
			},
			languages:        []string{"Japanese"},
			originalLanguage: "ja",
			channels:         "5.1",
			want:             "Kuroko's Basketball AKA Kuroko no Basket 2017 JAPANESE 1080p BluRay DD+ 5.1 x264-Kitsune",
		},
		{
			name: "lang-before-hybrid-repack",
			request: api.ReleaseNameRequest{
				Category:    "MOVIE",
				Type:        "ENCODE",
				Title:       "The Wailing",
				Year:        2016,
				Edition:     "Hybrid",
				Repack:      "REPACK",
				Resolution:  "1080p",
				Source:      "BluRay",
				VideoCodec:  "AVC",
				VideoEncode: "x264",
				Audio:       "DD 5.1",
				Tag:         "-GRP",
			},
			languages:        []string{"Korean"},
			originalLanguage: "ko",
			channels:         "5.1",
			want:             "The Wailing 2016 KOREAN Hybrid REPACK 1080p BluRay DD 5.1 x264-GRP",
		},
		{
			name: "lang-tv-episode-named",
			request: api.ReleaseNameRequest{
				Category:     "TV",
				Type:         "WEBDL",
				Title:        "Kingdom",
				Season:       "S01",
				Episode:      "E01",
				EpisodeTitle: "The Hungry Dead",
				Resolution:   "1080p",
				Source:       "Web",
				Service:      "NF",
				VideoCodec:   "AVC",
				VideoEncode:  "H.264",
				Audio:        "DD+ 5.1",
				Tag:          "-GRP",
			},
			languages:        []string{"Korean"},
			originalLanguage: "ko",
			channels:         "5.1",
			want:             "Kingdom S01E01 The Hungry Dead KOREAN 1080p NF WEB-DL DD+ 5.1 H.264-GRP",
		},
		{
			name: "dual-no-english",
			request: api.ReleaseNameRequest{
				Category:    "MOVIE",
				Type:        "ENCODE",
				Title:       "Ringu",
				Year:        1998,
				Resolution:  "1080p",
				Source:      "BluRay",
				VideoCodec:  "AVC",
				VideoEncode: "x264",
				Audio:       "Dual-Audio FLAC 2.0",
				Tag:         "-GRP",
			},
			languages:        []string{"Japanese", "French"},
			originalLanguage: "ja",
			channels:         "2.0",
			want:             "Ringu 1998 JAPANESE 1080p BluRay Dual-Audio FLAC 2.0 x264-GRP",
		},
		{
			name: "multi-with-english",
			request: api.ReleaseNameRequest{
				Category:    "MOVIE",
				Type:        "ENCODE",
				Title:       "Inferno",
				Year:        1980,
				Resolution:  "1080p",
				Source:      "BluRay",
				VideoCodec:  "AVC",
				VideoEncode: "x264",
				Audio:       "Dual-Audio DTS-HD MA 1.0",
				Tag:         "-GRP",
			},
			languages:        []string{"Italian", "English", "German"},
			originalLanguage: "it",
			channels:         "1.0",
			want:             "Inferno 1980 1080p BluRay Multi-Audio DTS-HD MA 1.0 x264-GRP",
		},
		{
			name: "dubbed",
			request: api.ReleaseNameRequest{
				Category:    "MOVIE",
				Type:        "ENCODE",
				Title:       "Zombie Flesh Eaters",
				Year:        1979,
				Resolution:  "1080p",
				Source:      "BluRay",
				VideoCodec:  "AVC",
				VideoEncode: "x264",
				Audio:       "Dubbed DTS-HD MA 1.0",
				Tag:         "-GRP",
			},
			languages:        []string{"English"},
			originalLanguage: "it",
			channels:         "1.0",
			want:             "Zombie Flesh Eaters 1979 1080p BluRay Dubbed DTS-HD MA 1.0 x264-GRP",
		},
		{
			name: "lang-zxx",
			request: api.ReleaseNameRequest{
				Category:    "MOVIE",
				Type:        "ENCODE",
				Title:       "Nosferatu",
				Year:        1922,
				Resolution:  "1080p",
				Source:      "BluRay",
				VideoCodec:  "AVC",
				VideoEncode: "x264",
				Audio:       "FLAC 2.0",
				Tag:         "-GRP",
			},
			languages:        []string{"No linguistic content"},
			originalLanguage: "de",
			channels:         "2.0",
			want:             "Nosferatu 1922 ZXX 1080p BluRay FLAC 2.0 x264-GRP",
		},
		{
			name: "zxx-music-track-before-dialogue",
			request: api.ReleaseNameRequest{
				Category:    "MOVIE",
				Type:        "ENCODE",
				Title:       "Dead Snow",
				Year:        2009,
				Resolution:  "1080p",
				Source:      "BluRay",
				VideoCodec:  "AVC",
				VideoEncode: "x264",
				Audio:       "DTS 5.1",
				Tag:         "-GRP",
			},
			languages:        []string{"No linguistic content", "Norwegian"},
			originalLanguage: "no",
			channels:         "5.1",
			want:             "Dead Snow 2009 NORWEGIAN 1080p BluRay DTS 5.1 x264-GRP",
		},
		{
			name: "dvd-remux-pal",
			request: api.ReleaseNameRequest{
				Category:   "MOVIE",
				Type:       "REMUX",
				Title:      "Alien Abduction: Incident in Lake County",
				Year:       1998,
				Resolution: "576i",
				Source:     "PAL DVD",
				VideoCodec: "MPEG-2",
				Audio:      "DD 2.0",
				Tag:        "-DVL",
			},
			channels: "2.0",
			want:     "Alien Abduction: Incident in Lake County 1998 576i DVD REMUX DD 2.0-DVL",
		},
		{
			name: "dvd-remux-foreign",
			request: api.ReleaseNameRequest{
				Category:   "MOVIE",
				Type:       "REMUX",
				Title:      "Kuroneko",
				Year:       1968,
				Resolution: "480i",
				Source:     "NTSC DVD",
				VideoCodec: "MPEG-2",
				Audio:      "DD 1.0",
				Tag:        "-GRP",
			},
			languages:        []string{"Japanese"},
			originalLanguage: "ja",
			channels:         "1.0",
			want:             "Kuroneko 1968 JAPANESE 480i DVD REMUX DD 1.0-GRP",
		},
		{
			name: "dvd-disc-foreign-no-marker",
			request: api.ReleaseNameRequest{
				Category:   "MOVIE",
				Type:       "DISC",
				DiscType:   "DVD",
				Title:      "Rabies",
				Year:       2010,
				Resolution: "576i",
				Source:     "PAL",
				DVDSize:    "DVD9",
				VideoCodec: "MPEG-2",
				Audio:      "DD 5.1",
				Tag:        "-GRP",
			},
			languages:        []string{"Hebrew", "French"},
			originalLanguage: "he",
			channels:         "5.1",
			want:             "Rabies 2010 PAL DVD9 DD 5.1-GRP",
		},
		{
			name: "dvdrip-foreign",
			request: api.ReleaseNameRequest{
				Category:    "MOVIE",
				Type:        "ENCODE",
				Title:       "The Vampires of Coyoacan",
				Year:        1974,
				Resolution:  "480p",
				Source:      "NTSC DVD",
				VideoCodec:  "AVC",
				VideoEncode: "x264",
				Audio:       "MP3 2.0",
				Tag:         "-GRP",
			},
			languages:        []string{"Spanish"},
			originalLanguage: "es",
			channels:         "2.0",
			want:             "The Vampires of Coyoacan 1974 SPANISH 480p DVDRip MP3 2.0 x264-GRP",
		},
		{
			name: "bd-disc-foreign-no-marker",
			request: api.ReleaseNameRequest{
				Category:   "MOVIE",
				Type:       "DISC",
				DiscType:   "BDMV",
				Title:      "Kwaidan",
				Year:       1964,
				Resolution: "1080p",
				Region:     "JPN",
				Source:     "Blu-ray",
				VideoCodec: "AVC",
				Audio:      "LPCM 1.0",
				Tag:        "-GRP",
			},
			languages:        []string{"Japanese", "English"},
			originalLanguage: "ja",
			channels:         "1.0",
			want:             "Kwaidan 1964 1080p JPN Blu-ray AVC LPCM 1.0-GRP",
		},
		{
			name: "bd-disc-cut-kept-distributor-dropped",
			request: api.ReleaseNameRequest{
				Category:   "MOVIE",
				Type:       "DISC",
				DiscType:   "BDMV",
				Title:      "Night of the Living Dead",
				Year:       1968,
				Edition:    "Criterion Director's Cut",
				Resolution: "1080p",
				Region:     "USA",
				Source:     "Blu-ray",
				VideoCodec: "AVC",
				Audio:      "LPCM 1.0",
				Tag:        "-GRP",
			},
			channels: "1.0",
			want:     "Night of the Living Dead 1968 Director's Cut 1080p USA Blu-ray AVC LPCM 1.0-GRP",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			subject := dvlWikiSubject(t, test.request, test.languages, test.originalLanguage, test.channels)
			subject.HasCommentary = test.commentary
			subject.HardcodedSubs = test.hardcodedSubs
			if got := dvlReviewedName(t, subject, nil); got != test.want {
				t.Fatalf("reviewed name = %q, want %q", got, test.want)
			}
		})
	}
}

func dvlWikiSubject(t *testing.T, request api.ReleaseNameRequest, languages []string, originalLanguage, channels string) api.UploadSubject {
	t.Helper()

	if languages == nil {
		languages = []string{"English"}
	}
	if originalLanguage == "" {
		originalLanguage = "en"
	}
	// Match media_tracks.aggregateTrackLanguages and preparedrelease.ResolveUploadSubject.
	subject := dvlGeneratedSubject(t, request, languageutil.NormalizeLanguageList(languages))
	subject.EffectiveMetadata = api.EffectiveMetadata{
		Title:            request.Title,
		AlternateTitle:   request.AltTitle,
		Year:             request.Year,
		OriginalLanguage: originalLanguage,
	}
	subject.ProviderMetadata.TMDB = &api.TMDBMetadata{OriginalLanguage: originalLanguage}
	subject.Tag = request.Tag
	subject.Channels = channels
	subject.Is3D = request.ThreeD
	subject.UHD = request.UHD
	subject.HDR = request.HDR
	subject.Region = request.Region
	subject.Edition = request.Edition
	subject.Repack = request.Repack
	subject.Service = request.Service
	subject.EpisodeTitle = request.EpisodeTitle
	subject.SeasonStr = request.Season
	subject.EpisodeStr = request.Episode
	if request.Season != "" {
		season, err := strconv.Atoi(strings.TrimPrefix(request.Season, "S"))
		if err != nil {
			t.Fatal(err)
		}
		subject.SeasonInt = season
	}
	if request.Episode != "" {
		episode, err := strconv.Atoi(strings.TrimPrefix(request.Episode, "E"))
		if err != nil {
			t.Fatal(err)
		}
		subject.EpisodeInt = episode
	}
	subject.TVPack = request.Category == "TV" && request.Episode == ""
	if strings.HasPrefix(request.VideoEncode, "Hi10P ") {
		subject.BitDepth = "10"
	}
	subject.Release.Type = request.Type
	subject.Release.Source = request.Source
	subject.Release.Alt = request.AltTitle
	subject.Release.Channels = channels
	subject.Release.Region = request.Region
	subject.Release.Group = strings.TrimPrefix(request.Tag, "-")
	subject.Release.Season = subject.SeasonInt
	subject.Release.Episode = subject.EpisodeInt
	if request.Edition != "" {
		subject.Release.Edition = []string{request.Edition}
	}
	return subject
}
