// SPDX-License-Identifier: AGPL-3.0-or-later
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// moltBin is the path to the built molt binary.
// Empty when the build failed — CLI tests skip themselves in that case.
var (
	moltBin   string
	driverDir string // directory containing molt-driver-nanoclaw, prepended to PATH
)

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "molt-cli-test-*")
	if err != nil {
		// Can't create temp dir; skip CLI tests but run everything else.
		os.Exit(m.Run())
	}

	wd, _ := os.Getwd() // drivers/nanoclaw/
	repoRoot := filepath.Join(wd, "..", "..")

	// Build molt from the root module (./src/ is package main).
	moltOut := filepath.Join(tmp, "molt")
	buildMolt := exec.Command("go", "build", "-o", moltOut, "./src/")
	buildMolt.Dir = repoRoot
	if err := buildMolt.Run(); err == nil {
		moltBin = moltOut
	}

	// Build molt-driver-nanoclaw from the current module.
	driverOut := filepath.Join(tmp, "molt-driver-nanoclaw")
	buildDriver := exec.Command("go", "build", "-o", driverOut, ".")
	buildDriver.Dir = wd
	if err := buildDriver.Run(); err != nil {
		moltBin = "" // driver build failed; disable all CLI tests
	} else {
		driverDir = tmp
	}

	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}
