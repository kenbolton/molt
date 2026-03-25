package driver

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DetectArch probes all installed drivers and returns the best match for sourceDir.
func DetectArch(sourceDir string) (string, error) {
	drivers, err := FindAll()
	if err != nil {
		return "", err
	}
	if len(drivers) == 0 {
		return "", fmt.Errorf("no drivers installed — cannot auto-detect arch\n\nInstall a driver or use --arch <name>")
	}

	type probe struct {
		arch       string
		confidence float64
	}
	var best probe

	for _, d := range drivers {
		c, err := probeConfidence(d, sourceDir)
		if err != nil {
			continue
		}
		if c > best.confidence {
			best = probe{arch: d.Arch, confidence: c}
		}
	}

	if best.confidence == 0 {
		return "", fmt.Errorf("could not detect arch for %q\n\nUse --arch to specify manually", sourceDir)
	}
	return best.arch, nil
}

// probeConfidence asks a driver how confident it is that sourceDir is its arch.
func probeConfidence(d *Driver, sourceDir string) (float64, error) {
	req, _ := json.Marshal(map[string]string{
		"type":       "probe_request",
		"source_dir": sourceDir,
	})

	cmd := execDriver(d.Path, string(req))
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	var resp struct {
		Type       string  `json:"type"`
		Confidence float64 `json:"confidence"`
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if err := json.Unmarshal([]byte(line), &resp); err == nil && resp.Type == "probe_response" {
			return resp.Confidence, nil
		}
	}
	return 0, nil
}
