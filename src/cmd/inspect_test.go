package cmd

import (
	"encoding/json"
	"testing"
)

// makeGroupConfig returns a config.json for a group, optionally with a
// symlink_target in the _arch_nanoclaw field.
func makeGroupConfig(t *testing.T, name, symlinkTarget string) []byte {
	t.Helper()
	arch := map[string]interface{}{}
	if symlinkTarget != "" {
		arch["symlink_target"] = symlinkTarget
	}
	archJSON, _ := json.Marshal(arch)
	cfg := groupConfigSlim{
		Name:         name,
		ArchNanoclaw: archJSON,
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestBuildGroupSet_SymlinkTargetsNotOrphaned(t *testing.T) {
	// foo-signal is a symlink group pointing to foo.
	// Sessions stored under "foo" belong to foo-signal and must not be orphaned.
	files := map[string][]byte{
		"groups/foo-signal/config.json": makeGroupConfig(t, "Foo (Signal)", "foo"),
		"groups/bar/config.json":        makeGroupConfig(t, "Bar", ""),
	}
	groups := []string{"foo-signal", "bar"}

	set := buildGroupSet(files, groups)

	for _, slug := range []string{"foo-signal", "bar", "foo"} {
		if !set[slug] {
			t.Errorf("expected %q in group set", slug)
		}
	}
}

func TestBuildGroupSet_GenuineOrphanNotIncluded(t *testing.T) {
	files := map[string][]byte{
		"groups/bar/config.json": makeGroupConfig(t, "Bar", ""),
	}
	groups := []string{"bar"}

	set := buildGroupSet(files, groups)

	if set["arty"] {
		t.Error("arty should not be in group set — it has no group or symlink association")
	}
}

func TestBuildGroupSet_MissingConfig(t *testing.T) {
	// Group in manifest but no config.json in files — should still be in set, not panic.
	files := map[string][]byte{}
	groups := []string{"ghost"}

	set := buildGroupSet(files, groups)

	if !set["ghost"] {
		t.Error("ghost should be in group set even without a config.json")
	}
}
