package cmd

import (
	"path/filepath"

	"github.com/kenbolton/molt/src/driver"
)

func locateDriver(arch string) (*driver.Driver, error) {
	return driver.Locate(arch)
}

func detectOrFlagArch(sourceDir string) (string, error) {
	if flagArch != "" {
		return flagArch, nil
	}
	return driver.DetectArch(sourceDir)
}

func bundleNameFromSource(sourceDir string) string {
	base := filepath.Base(filepath.Clean(sourceDir))
	return base + ".molt"
}
