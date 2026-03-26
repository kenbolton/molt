// SPDX-License-Identifier: AGPL-3.0-or-later
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// exportSessions exports Claude session caches for all groups — best effort.
// Returns any warnings encountered.
func exportSessions(sourceDir string) []string {
	sessionsDir := filepath.Join(sourceDir, "data", "sessions")
	if _, err := os.Stat(sessionsDir); err != nil {
		return nil // no sessions dir, nothing to do
	}

	var warnings []string

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return []string{fmt.Sprintf("sessions: could not read sessions dir: %v", err)}
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		slug := entry.Name()
		sessionDir := filepath.Join(sessionsDir, slug)

		files, walkWarnings, err := walkSessionDir(sessionDir)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("sessions/%s: %v", slug, err))
			continue
		}
		for _, w := range walkWarnings {
			warnings = append(warnings, fmt.Sprintf("sessions/%s/%s", slug, w))
		}
		if len(files) == 0 {
			continue
		}

		write(map[string]interface{}{
			"type":        "session",
			"slug":        slug,
			"best_effort": true,
			"files":       files,
		})
	}
	return warnings
}

// walkSessionDir reads a session directory, skipping files above MaxSessionFileSize.
// Returns files, warnings, and any walk error.
func walkSessionDir(dir string) ([]BundleFile, []string, error) {
	var files []BundleFile
	var warnings []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		if info.Size() > MaxSessionFileSize {
			warnings = append(warnings, fmt.Sprintf(
				"%s: skipped (%.1f MB exceeds %d MB limit)",
				rel, float64(info.Size())/(1024*1024), MaxSessionFileSize/(1024*1024),
			))
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		files = append(files, BundleFile{Path: rel, Content: content})
		return nil
	})
	return files, warnings, err
}
