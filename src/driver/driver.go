// Package driver implements the molt driver protocol.
// Drivers are standalone binaries (molt-driver-<arch>) that communicate
// via newline-delimited JSON on stdin/stdout.
package driver

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Driver is a located and version-verified arch driver.
type Driver struct {
	Arch          string
	ArchVersion   string
	DriverVersion string
	Path          string
}

// Message types in the driver protocol.
type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:",omitempty"`
}

// FindAll locates all molt-driver-* binaries in $PATH and ~/.molt/drivers/.
func FindAll() ([]*Driver, error) {
	var drivers []*Driver
	seen := map[string]bool{}

	paths := searchPaths()
	for _, dir := range paths {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !strings.HasPrefix(e.Name(), "molt-driver-") {
				continue
			}
			arch := strings.TrimPrefix(e.Name(), "molt-driver-")
			fullPath := filepath.Join(dir, e.Name())
			if seen[arch] {
				continue
			}
			seen[arch] = true

			d, err := probe(arch, fullPath)
			if err != nil {
				continue // skip broken drivers
			}
			drivers = append(drivers, d)
		}
	}
	return drivers, nil
}

// Locate finds the driver for a specific arch.
func Locate(arch string) (*Driver, error) {
	name := "molt-driver-" + arch
	for _, dir := range searchPaths() {
		fullPath := filepath.Join(dir, name)
		if _, err := os.Stat(fullPath); err == nil {
			return probe(arch, fullPath)
		}
	}
	// Also try PATH lookup
	if path, err := exec.LookPath(name); err == nil {
		return probe(arch, path)
	}
	return nil, fmt.Errorf("driver not found for arch %q\n\nInstall molt-driver-%s to your PATH or ~/.molt/drivers/", arch, arch)
}

// Export runs the export protocol against the driver.
func (d *Driver) Export(sourceDir, outPath string) error {
	// TODO: implement ndjson protocol
	return fmt.Errorf("export not yet implemented")
}

// Import runs the import protocol against the driver.
func (d *Driver) Import(bundlePath, destDir string, renames map[string]string) error {
	// TODO: implement ndjson protocol
	return fmt.Errorf("import not yet implemented")
}

// probe calls the driver's version endpoint to get metadata.
func probe(arch, path string) (*Driver, error) {
	cmd := exec.Command(path)
	cmd.Stdin = strings.NewReader(`{"type":"version_request"}` + "\n")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("driver %s failed version check: %w", path, err)
	}

	var resp struct {
		Type          string `json:"type"`
		Arch          string `json:"arch"`
		ArchVersion   string `json:"arch_version"`
		DriverVersion string `json:"driver_version"`
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		if err := json.Unmarshal(scanner.Bytes(), &resp); err == nil && resp.Type == "version_response" {
			break
		}
	}
	if resp.Arch == "" {
		return nil, fmt.Errorf("driver %s returned no version_response", path)
	}
	return &Driver{
		Arch:          resp.Arch,
		ArchVersion:   resp.ArchVersion,
		DriverVersion: resp.DriverVersion,
		Path:          path,
	}, nil
}

func searchPaths() []string {
	paths := []string{}
	// ~/.molt/drivers/ first (user-installed takes precedence)
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".molt", "drivers"))
	}
	// Then $PATH entries
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		paths = append(paths, p)
	}
	return paths
}
