// SPDX-License-Identifier: AGPL-3.0-or-later
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func skipIfNoBinary(t *testing.T) {
	t.Helper()
	if moltBin == "" || driverDir == "" {
		t.Skip("molt or molt-driver-nanoclaw binary not built — skipping CLI tests")
	}
}

// runMolt runs the molt binary with the test driver on PATH.
// Returns stdout, stderr, and the run error (nil = exit 0).
func runMolt(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := exec.Command(moltBin, args...)
	// Prepend driverDir so molt finds molt-driver-nanoclaw without ~/.molt/drivers/.
	cmd.Env = append(os.Environ(),
		"PATH="+driverDir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// exportBundle runs molt export and returns the path to the produced .molt file.
func exportBundle(t *testing.T, srcDir string) string {
	t.Helper()
	bundlePath := filepath.Join(t.TempDir(), "test.molt")
	stdout, stderr, err := runMolt(t, "export", srcDir, "--arch", "nanoclaw", "--out", bundlePath)
	if err != nil {
		t.Fatalf("exportBundle failed:\nstdout: %s\nstderr: %s\nerr: %v", stdout, stderr, err)
	}
	return bundlePath
}

// readManifest opens a .molt bundle and returns manifest.json as a generic map.
func readManifest(t *testing.T, bundlePath string) map[string]interface{} {
	t.Helper()
	f, err := os.Open(bundlePath)
	if err != nil {
		t.Fatalf("readManifest: open %s: %v", bundlePath, err)
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("readManifest: gzip: %v", err)
	}
	defer func() { _ = gz.Close() }()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("readManifest: tar: %v", err)
		}
		if hdr.Name != "manifest.json" {
			continue
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("readManifest: read manifest.json: %v", err)
		}
		var m map[string]interface{}
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatalf("readManifest: parse manifest.json: %v", err)
		}
		return m
	}
	t.Fatal("readManifest: manifest.json not found in bundle")
	return nil
}

// queryGroupFolders returns sorted folder names from registered_groups in destDir.
func queryGroupFolders(t *testing.T, destDir string) []string {
	t.Helper()
	dbPath := filepath.Join(destDir, "store", "messages.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("queryGroupFolders: open DB: %v", err)
	}
	defer func() { _ = db.Close() }()
	rows, err := db.Query(`SELECT folder FROM registered_groups`)
	if err != nil {
		t.Fatalf("queryGroupFolders: query: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var folders []string
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			t.Fatal(err)
		}
		folders = append(folders, f)
	}
	sort.Strings(folders)
	return folders
}

// queryTaskCount returns the number of rows in scheduled_tasks.
func queryTaskCount(t *testing.T, destDir string) int {
	t.Helper()
	dbPath := filepath.Join(destDir, "store", "messages.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("queryTaskCount: open DB: %v", err)
	}
	defer func() { _ = db.Close() }()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM scheduled_tasks`).Scan(&count); err != nil {
		t.Fatalf("queryTaskCount: %v", err)
	}
	return count
}

// ── tests ─────────────────────────────────────────────────────────────────────

// TestCLIExport verifies that molt export produces a valid .molt bundle with
// the expected manifest content.
func TestCLIExport(t *testing.T) {
	skipIfNoBinary(t)

	srcDir := setupSourceDir(t)
	bundlePath := filepath.Join(t.TempDir(), "out.molt")

	stdout, stderr, err := runMolt(t, "export", srcDir, "--arch", "nanoclaw", "--out", bundlePath)
	if err != nil {
		t.Fatalf("molt export failed:\nstdout: %s\nstderr: %s\nerr: %v", stdout, stderr, err)
	}

	if _, err := os.Stat(bundlePath); err != nil {
		t.Fatalf("bundle file not created at %s", bundlePath)
	}

	m := readManifest(t, bundlePath)

	source, _ := m["source"].(map[string]interface{})
	if source["arch"] != "nanoclaw" {
		t.Errorf("manifest source.arch = %q, want nanoclaw", source["arch"])
	}

	groups, _ := m["groups"].([]interface{})
	groupSet := map[string]bool{}
	for _, g := range groups {
		if s, ok := g.(string); ok {
			groupSet[s] = true
		}
	}
	for _, want := range []string{"main", "other", "global"} {
		if !groupSet[want] {
			t.Errorf("manifest.groups missing %q; got %v", want, groups)
		}
	}

	// checksums should be absent (omitempty)
	if _, ok := m["checksums"]; ok {
		t.Error("manifest.checksums should be absent (reserved, never populated)")
	}
}

// TestCLIImport verifies that molt import writes group files, populates the
// destination DB, and restores session files.
func TestCLIImport(t *testing.T) {
	skipIfNoBinary(t)

	srcDir := setupSourceDir(t)
	bundlePath := exportBundle(t, srcDir)
	destDir := setupDestDir(t)

	stdout, stderr, err := runMolt(t, "import", bundlePath, destDir, "--arch", "nanoclaw")
	if err != nil {
		t.Fatalf("molt import failed:\nstdout: %s\nstderr: %s\nerr: %v", stdout, stderr, err)
	}

	// Group directories
	for _, slug := range []string{"main", "other", "global"} {
		if _, err := os.Stat(filepath.Join(destDir, "groups", slug)); err != nil {
			t.Errorf("groups/%s missing after import", slug)
		}
	}

	// Registered groups in DB (global has no JID, so only 2)
	folders := queryGroupFolders(t, destDir)
	if len(folders) != 2 {
		t.Errorf("expected 2 registered groups, got %d: %v", len(folders), folders)
	}

	// Session files (best-effort)
	sessionFile := filepath.Join(destDir, "data", "sessions", "main", "session.jsonl")
	if _, err := os.Stat(sessionFile); err != nil {
		t.Errorf("session file not restored: %v", err)
	}
}

// TestCLIRoundTrip exports a full NanoClaw install and imports it to a fresh
// destination, then verifies DB rows, files, tasks, and sessions all match.
func TestCLIRoundTrip(t *testing.T) {
	skipIfNoBinary(t)

	srcDir := setupSourceDir(t)
	destDir := setupDestDir(t)
	bundlePath := filepath.Join(t.TempDir(), "roundtrip.molt")

	if _, _, err := runMolt(t, "export", srcDir, "--arch", "nanoclaw", "--out", bundlePath); err != nil {
		t.Fatalf("export: %v", err)
	}
	if _, _, err := runMolt(t, "import", bundlePath, destDir, "--arch", "nanoclaw"); err != nil {
		t.Fatalf("import: %v", err)
	}

	// DB rows must match source
	srcRows, err := readGroupRows(srcDir)
	if err != nil {
		t.Fatalf("readGroupRows src: %v", err)
	}
	dstRows, err := readGroupRows(destDir)
	if err != nil {
		t.Fatalf("readGroupRows dst: %v", err)
	}
	if len(srcRows) != len(dstRows) {
		t.Errorf("registered group count: src=%d dst=%d", len(srcRows), len(dstRows))
	}
	srcByFolder := map[string]GroupRow{}
	for _, r := range srcRows {
		srcByFolder[r.Folder] = r
	}
	dstByFolder := map[string]GroupRow{}
	for _, r := range dstRows {
		dstByFolder[r.Folder] = r
	}
	for folder, srcRow := range srcByFolder {
		dstRow, ok := dstByFolder[folder]
		if !ok {
			t.Errorf("group %q in src but missing in dest", folder)
			continue
		}
		if srcRow.JID != dstRow.JID {
			t.Errorf("group %q JID: src=%q dst=%q", folder, srcRow.JID, dstRow.JID)
		}
		if srcRow.IsMain != dstRow.IsMain {
			t.Errorf("group %q is_main: src=%v dst=%v", folder, srcRow.IsMain, dstRow.IsMain)
		}
	}

	// Tasks
	if n := queryTaskCount(t, destDir); n != 1 {
		t.Errorf("expected 1 task in dest, got %d", n)
	}

	// Group files
	for _, slug := range []string{"main", "other", "global"} {
		p := filepath.Join(destDir, "groups", slug, "CLAUDE.md")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("groups/%s/CLAUDE.md missing after round-trip", slug)
		}
	}

	// Session files
	sessionFile := filepath.Join(destDir, "data", "sessions", "main", "session.jsonl")
	if _, err := os.Stat(sessionFile); err != nil {
		t.Errorf("session file missing after round-trip: %v", err)
	}
}

// TestCLICollision verifies that importing to a destination where a slug already
// exists fails with a helpful error message containing a --rename suggestion.
func TestCLICollision(t *testing.T) {
	skipIfNoBinary(t)

	srcDir := setupSourceDir(t)
	bundlePath := exportBundle(t, srcDir)
	destDir := setupDestDir(t)

	// Pre-create the group dir to trigger a collision on "main".
	if err := os.MkdirAll(filepath.Join(destDir, "groups", "main"), 0o755); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runMolt(t, "import", bundlePath, destDir, "--arch", "nanoclaw")
	if err == nil {
		t.Fatalf("expected import to fail on collision; stdout: %s", stdout)
	}

	combined := stdout + stderr
	if !strings.Contains(combined, "main") {
		t.Errorf("collision output should mention slug 'main':\n%s", combined)
	}
	if !strings.Contains(combined, "--rename") {
		t.Errorf("collision output should suggest --rename:\n%s", combined)
	}
}

// TestCLIRename verifies that --rename old=new correctly moves the group
// directory, updates the DB, and follows the rename into session files.
func TestCLIRename(t *testing.T) {
	skipIfNoBinary(t)

	srcDir := setupSourceDir(t)
	bundlePath := exportBundle(t, srcDir)
	destDir := setupDestDir(t)

	stdout, stderr, err := runMolt(t,
		"import", bundlePath, destDir,
		"--arch", "nanoclaw",
		"--rename", "main=main-imported",
	)
	if err != nil {
		t.Fatalf("import with rename failed:\nstdout: %s\nstderr: %s\nerr: %v", stdout, stderr, err)
	}

	// Renamed directory must exist; original must not.
	if _, err := os.Stat(filepath.Join(destDir, "groups", "main-imported")); err != nil {
		t.Error("groups/main-imported missing after rename import")
	}
	if _, err := os.Stat(filepath.Join(destDir, "groups", "main")); err == nil {
		t.Error("groups/main should not exist after rename import")
	}

	// DB must contain renamed slug, not original.
	folders := queryGroupFolders(t, destDir)
	found := false
	for _, f := range folders {
		if f == "main-imported" {
			found = true
		}
		if f == "main" {
			t.Error("original slug 'main' should not be in DB after rename")
		}
	}
	if !found {
		t.Errorf("renamed slug 'main-imported' not in DB; folders: %v", folders)
	}

	// Sessions should follow the rename.
	sessionFile := filepath.Join(destDir, "data", "sessions", "main-imported", "session.jsonl")
	if _, err := os.Stat(sessionFile); err != nil {
		t.Errorf("session not at data/sessions/main-imported/: %v", err)
	}
}
