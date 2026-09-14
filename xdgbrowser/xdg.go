// Package xdgbrowser reads and sets the system default web browser via
// the xdg-settings CLI, rather than editing mimeapps.list directly, so
// we inherit whatever desktop-specific handling xdg-settings already
// does (and stay compatible if that mechanism changes).
package xdgbrowser

import (
	"fmt"
	"os/exec"
	"strings"
)

// Current returns the desktop file id of the current default browser,
// e.g. "yandex-browser" (without the .desktop suffix).
func Current() (string, error) {
	out, err := exec.Command("xdg-settings", "get", "default-web-browser").Output()
	if err != nil {
		return "", fmt.Errorf("xdg-settings get: %w", err)
	}
	id := strings.TrimSpace(string(out))
	id = strings.TrimSuffix(id, ".desktop")
	return id, nil
}

// SetDefault sets the default web browser to the given desktop file id
// (without the .desktop suffix — xdg-settings adds it back internally
// in some versions, so we pass the raw id as-is, matching what
// `xdg-settings get` itself returns).
func SetDefault(id string) error {
	out, err := exec.Command("xdg-settings", "set", "default-web-browser", id+".desktop").CombinedOutput()
	if err != nil {
		return fmt.Errorf("xdg-settings set %s: %w (%s)", id, err, strings.TrimSpace(string(out)))
	}
	return nil
}
