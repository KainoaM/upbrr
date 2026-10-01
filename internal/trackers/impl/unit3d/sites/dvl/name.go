// Copyright (c) 2025-2026, Audionut and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package dvl

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/autobrr/upbrr/internal/config"
	"github.com/autobrr/upbrr/internal/languageutil"
	"github.com/autobrr/upbrr/internal/trackers"
	"github.com/autobrr/upbrr/internal/trackers/impl/unit3d"
	"github.com/autobrr/upbrr/pkg/api"
)

var dvlOmittedEdition = regexp.MustCompile(`(?i)\b(?:(?:4K\s+)?Remaster(?:ed)?|Criterion|Arrow|RESTORED|Internal|Limited|Retail|Version|Edition)\b`)

func namePolicy() trackers.ReleaseNamePolicyBinding {
	return trackers.StructuredReleaseNamePolicy("unit3d/dvl/v2", trackers.StructuredNamePolicy{
		Defaults: applyDVLNameDefaults,
	})
}

func applyDVLNameDefaults(editor *trackers.NameEditor, meta api.UploadSubject, _ config.TrackerConfig) error {
	nameType := strings.ToUpper(strings.TrimSpace(meta.Type))
	source := strings.TrimSpace(meta.Source)
	if err := editor.MoveBefore(api.NameRoleAlternateTitle, api.NameRoleYear); err != nil {
		return fmt.Errorf("move DVL alternate title before year: %w", err)
	}
	if err := normalizeDVLEdition(editor); err != nil {
		return err
	}
	if err := applyDVLAudio(editor, meta, nameType == "DISC"); err != nil {
		return err
	}

	switch {
	case nameType == "DVDRIP":
		if err := omitDVDEncodeMarkers(editor); err != nil {
			return err
		}
		if err := applyDVDRipNameOrder(editor); err != nil {
			return err
		}
	case nameType == "ENCODE" && unit3d.Resolution(meta) != "" && isDVDSource(source):
		if err := omitDVDEncodeMarkers(editor); err != nil {
			return err
		}
		if err := editor.Set(api.NameRoleSource, "DVDRip"); err != nil {
			return fmt.Errorf("set DVL DVD encode source: %w", err)
		}
	case nameType == "DISC" && strings.EqualFold(meta.DiscType, "DVD"):
		if err := addDVDSystemForDisc(editor, meta, source); err != nil {
			return err
		}
	case nameType == "REMUX" && isDVDSource(source):
		if err := editor.Set(api.NameRoleSource, "DVD"); err != nil {
			return fmt.Errorf("set DVL DVD remux source: %w", err)
		}
		if err := editor.Include(api.NameRoleResolution); err != nil {
			return fmt.Errorf("include DVL DVD remux resolution: %w", err)
		}
		if err := editor.MoveBefore(api.NameRoleResolution, api.NameRoleSource); err != nil {
			return fmt.Errorf("move DVL DVD remux resolution: %w", err)
		}
	}
	if err := applyDVLSource(editor, meta, nameType); err != nil {
		return err
	}
	if strings.EqualFold(meta.Release.Category, "TV") && (nameType == "DISC" || nameType == "DVDRIP") &&
		!meta.NamePresentation.OmitSeasonEpisode && !meta.NamePresentation.UseDailyDate {
		for _, role := range []api.ReleaseNameRole{api.NameRoleSeason, api.NameRoleEpisode, api.NameRoleEpisodeTitle} {
			if role != api.NameRoleSeason && (meta.TVPack || meta.EpisodeInt == 0 && meta.EpisodeStr == "") {
				continue
			}
			if err := editor.Include(role); err != nil {
				return fmt.Errorf("include DVL episode %s: %w", role, err)
			}
		}
		if err := editor.MoveAfter(api.NameRoleEpisode, api.NameRoleSeason); err != nil {
			return fmt.Errorf("move DVL episode after season: %w", err)
		}
		if err := editor.MoveAfter(api.NameRoleEpisodeTitle, api.NameRoleEpisode); err != nil {
			return fmt.Errorf("move DVL episode title: %w", err)
		}
	}
	if nameType != "DISC" {
		if err := addDVLLanguageMarker(editor, meta); err != nil {
			return err
		}
	}
	return nil
}

func applyDVDRipNameOrder(editor *trackers.NameEditor) error {
	if err := editor.Omit(api.NameRoleSource); err != nil {
		return fmt.Errorf("omit DVL DVDRip source: %w", err)
	}
	if err := editor.Include(api.NameRoleResolution); err != nil {
		return fmt.Errorf("include DVL DVDRip resolution: %w", err)
	}
	cluster := dvlAudioClusterRoles(editor.PresentRoles())
	videoAnchor := api.NameRoleVideoFormat
	if len(cluster) > 0 {
		videoAnchor = cluster[len(cluster)-1]
		if err := editor.MoveBefore(api.NameRoleVideoFormat, cluster[0]); err != nil {
			return fmt.Errorf("move DVL DVDRip format before audio cluster: %w", err)
		}
	}
	for _, role := range []api.ReleaseNameRole{api.NameRoleVideoEncode, api.NameRoleVideoCodec} {
		if err := editor.MoveAfter(role, videoAnchor); err != nil {
			return fmt.Errorf("move DVL DVDRip %s after format or audio: %w", role, err)
		}
	}
	if err := editor.MoveBefore(api.NameRoleResolution, api.NameRoleVideoFormat); err != nil {
		return fmt.Errorf("move DVL DVDRip resolution before format: %w", err)
	}
	return nil
}

func omitDVDEncodeMarkers(editor *trackers.NameEditor) error {
	for _, role := range []api.ReleaseNameRole{api.NameRoleEdition, api.NameRoleHybrid, api.NameRoleRepack} {
		if err := editor.Omit(role); err != nil {
			return fmt.Errorf("omit DVL DVD encode %s: %w", role, err)
		}
	}
	return nil
}

func addDVDSystemForDisc(editor *trackers.NameEditor, meta api.UploadSubject, source string) error {
	if !isDVDSource(source) || !isDVDSize(meta.Release.Size) || source != "DVD" {
		return nil
	}
	system := dvdSystem(unit3d.Resolution(meta))
	if system == "" {
		return nil
	}
	if err := editor.InsertBefore(api.NameRoleDVDSystem, system, api.NameRoleDVDSize); err != nil {
		return fmt.Errorf("insert DVL DVD disc system: %w", err)
	}
	return nil
}

func addDVLLanguageMarker(editor *trackers.NameEditor, meta api.UploadSubject) error {
	language, _, _ := dvlAudioLanguages(meta)
	if language == "" {
		return nil
	}

	anchor := firstPresentNameRole(editor.PresentRoles(), api.NameRoleEpisodeTitle, api.NameRoleEpisode, api.NameRoleSeason,
		api.NameRoleDailyDate, api.NameRoleYear, api.NameRoleAlternateTitle, api.NameRoleTitle)
	if !anchor.Valid() {
		return nil
	}
	if err := editor.InsertAfter(api.NameRoleLanguageMarker, language, anchor); err != nil {
		return fmt.Errorf("insert DVL language marker: %w", err)
	}
	return nil
}

func dvlAudioLanguages(meta api.UploadSubject) (string, string, bool) {
	// Prepared audio languages retain track order and exclude commentary.
	languages := make([]string, 0, len(meta.AudioLanguages))
	marker, dub := "", ""
	nonLinguistic := false
	for _, value := range meta.AudioLanguages {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "", "und", "undetermined":
			continue
		case "zxx", "no linguistic content":
			nonLinguistic = true
			continue
		}
		if normalized := languageutil.NormalizeLanguageLabel(value); normalized != "" {
			value = normalized
		}
		if marker == "" {
			marker = strings.ToUpper(strings.TrimSpace(value))
		}
		languages = append(languages, value)
	}
	languages = languageutil.NormalizeLanguageList(languages)
	switch {
	case len(languages) >= 3:
		dub = "Multi-Audio"
	case len(languages) == 2:
		dub = "Dual-Audio"
	case len(languages) == 1 && unit3d.HasEnglishLanguage(languages):
		original := languageutil.NormalizeLanguageCode(meta.EffectiveMetadata.OriginalLanguage)
		if original != "" && original != "en" && original != "und" && original != "zxx" && original != "mul" {
			dub = "Dubbed"
		}
	}
	if unit3d.HasEnglishLanguage(languages) {
		marker = ""
	} else if len(languages) == 0 && nonLinguistic {
		marker = "ZXX"
	}
	return marker, dub, len(languages) > 0 || nonLinguistic
}

func firstPresentNameRole(present []api.ReleaseNameRole, candidates ...api.ReleaseNameRole) api.ReleaseNameRole {
	for _, role := range candidates {
		if slices.Contains(present, role) {
			return role
		}
	}
	return ""
}

func dvlAudioClusterRoles(present []api.ReleaseNameRole) []api.ReleaseNameRole {
	roles := make([]api.ReleaseNameRole, 0, 3)
	for _, role := range present {
		if role == api.NameRoleDubbed || role == api.NameRoleAudio || role == api.NameRoleDualAudio {
			roles = append(roles, role)
		}
	}
	return roles
}

func isDVDSource(source string) bool {
	switch strings.ToUpper(strings.TrimSpace(source)) {
	case "PAL DVD", "NTSC DVD", "DVD":
		return true
	default:
		return false
	}
}

func isDVDSize(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "DVD5") || strings.EqualFold(strings.TrimSpace(value), "DVD9")
}

func dvdSystem(resolution string) string {
	switch resolution {
	case "576i", "576p":
		return "PAL"
	case "480i", "480p":
		return "NTSC"
	default:
		return ""
	}
}

func normalizeDVLEdition(editor *trackers.NameEditor) error {
	if component, ok := editor.Component(api.NameRoleRepack); ok && strings.EqualFold(component.Value, "RERIP") {
		if err := editor.Set(api.NameRoleRepack, "RERiP"); err != nil {
			return fmt.Errorf("normalize DVL rerip: %w", err)
		}
	}
	if component, ok := editor.Component(api.NameRoleEdition); ok && component.Present {
		value := strings.Join(strings.Fields(dvlOmittedEdition.ReplaceAllString(component.Value, "")), " ")
		if value == "" {
			if err := editor.Omit(api.NameRoleEdition); err != nil {
				return fmt.Errorf("omit DVL edition: %w", err)
			}
		} else if err := editor.Set(api.NameRoleEdition, value); err != nil {
			return fmt.Errorf("normalize DVL edition: %w", err)
		}
	}
	if err := editor.Omit(api.NameRoleDistributor); err != nil {
		return fmt.Errorf("omit DVL distributor: %w", err)
	}
	roles := editor.PresentRoles()
	if repack := slices.Index(roles, api.NameRoleRepack); repack >= 0 && slices.Index(roles, api.NameRoleEdition) > repack {
		if err := editor.MoveBefore(api.NameRoleEdition, api.NameRoleRepack); err != nil {
			return fmt.Errorf("move DVL edition before repack: %w", err)
		}
	}
	return nil
}

func applyDVLAudio(editor *trackers.NameEditor, meta api.UploadSubject, fullDisc bool) error {
	_, dub, known := dvlAudioLanguages(meta)
	if !fullDisc && !known {
		return nil
	}
	for _, role := range []api.ReleaseNameRole{api.NameRoleDubbed, api.NameRoleDualAudio} {
		if err := editor.Omit(role); err != nil {
			return fmt.Errorf("omit DVL audio marker %s: %w", role, err)
		}
	}
	if fullDisc {
		return nil
	}
	if dub != "" {
		role := api.NameRoleDualAudio
		if dub == "Dubbed" {
			role = api.NameRoleDubbed
		}
		if err := editor.InsertBefore(role, dub, api.NameRoleAudio); err != nil {
			return fmt.Errorf("insert DVL dub token: %w", err)
		}
	}
	return nil
}

func applyDVLSource(editor *trackers.NameEditor, meta api.UploadSubject, nameType string) error {
	fullDisc := nameType == "DISC"
	if nameType == "REMUX" || nameType == "HDTV" {
		if err := editor.Include(api.NameRoleHybrid); err != nil {
			return fmt.Errorf("include DVL hybrid: %w", err)
		}
		anchor := firstPresentNameRole(editor.PresentRoles(), api.NameRoleRepack, api.NameRoleResolution, api.NameRoleSource)
		if anchor.Valid() {
			if err := editor.MoveBefore(api.NameRoleHybrid, anchor); err != nil {
				return fmt.Errorf("move DVL hybrid: %w", err)
			}
		}
	}
	source := strings.TrimSpace(meta.Source)
	switch {
	case fullDisc && (strings.EqualFold(source, "BluRay") || strings.EqualFold(source, "Blu-ray")):
		source = "Blu-ray"
	default:
		source = ""
	}
	if source != "" {
		if err := editor.Set(api.NameRoleSource, source); err != nil {
			return fmt.Errorf("normalize DVL source: %w", err)
		}
	}
	if fullDisc {
		if err := editor.Omit(api.NameRoleHybrid); err != nil {
			return fmt.Errorf("omit DVL full disc hybrid: %w", err)
		}
		if err := editor.Include(api.NameRoleRegion); err != nil {
			return fmt.Errorf("include DVL disc region: %w", err)
		}
		if err := editor.MoveAfter(api.NameRoleRegion, api.NameRoleResolution); err != nil {
			return fmt.Errorf("move DVL disc region: %w", err)
		}
	}
	if component, ok := editor.Component(api.NameRoleVideoEncode); ok && component.Present {
		value := strings.Join(strings.Fields(strings.ReplaceAll(component.Value, "Hi10P", "")), " ")
		if value == "x264" && meta.BitDepth == "10" && strings.TrimSpace(meta.HDR) == "" {
			value = "Hi10P x264"
		}
		if err := editor.Set(api.NameRoleVideoEncode, value); err != nil {
			return fmt.Errorf("normalize DVL Hi10P: %w", err)
		}
	}
	return nil
}
