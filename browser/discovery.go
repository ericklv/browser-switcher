// Package browser discovers installed web browsers by scanning XDG
// .desktop application entries, and dedupes entries that point at the
// same underlying binary (e.g. a distro package entry vs. a user
// override with extra flags).
package browser

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Browser represents one discovered, launchable web browser.
type Browser struct {
	// ID is the desktop file id (filename without .desktop), which is
	// exactly what goes into mimeapps.list / xdg-settings.
	ID string
	// Name is the human-readable name, as shown in menus.
	Name string
	// Icon is an icon-name (theme lookup key), not a file path.
	Icon string
	// Exec is the raw Exec= line (with %U/%u field codes still present).
	Exec string
	// BinaryPath is Exec with field codes and args stripped — used only
	// for de-duplication, not for launching.
	BinaryPath string
	// Path is the source .desktop file, kept for debugging/logging.
	Path string
	// Local is true if this entry came from a user data dir
	// (e.g. ~/.local/share/applications) rather than a system one.
	Local bool
}

// xdgDataDirs returns the ordered list of application dirs to scan,
// most specific (highest precedence) first: XDG_DATA_HOME, then each
// entry of XDG_DATA_DIRS. This mirrors the precedence rule from the
// XDG Base Directory spec, which is also why a single filename
// collision between these dirs already resolves itself for free.
func xdgDataDirs() []string {
	var dirs []string

	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}
	dirs = append(dirs, filepath.Join(dataHome, "applications"))

	dataDirs := os.Getenv("XDG_DATA_DIRS")
	if dataDirs == "" {
		dataDirs = "/usr/local/share:/usr/share"
	}
	for _, d := range strings.Split(dataDirs, ":") {
		if d == "" {
			continue
		}
		dirs = append(dirs, filepath.Join(d, "applications"))
	}
	return dirs
}

// Discover scans all XDG application dirs and returns the deduped list
// of browsers, i.e. desktop entries that declare themselves a handler
// for the https URI scheme.
func Discover() ([]Browser, error) {
	seenID := make(map[string]bool)    // dedup by desktop file id
	seenBinary := make(map[string]int) // binary path -> index in result
	var result []Browser

	for dirIdx, dir := range xdgDataDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue // dir may not exist; that's fine, keep scanning
		}
		local := dirIdx == 0 // XDG_DATA_HOME is always first in our list

		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".desktop") {
				continue
			}
			id := strings.TrimSuffix(e.Name(), ".desktop")
			if seenID[id] {
				continue // filename collision: higher-precedence dir already won
			}

			path := filepath.Join(dir, e.Name())
			b, ok, err := parseDesktopFile(path)
			if err != nil || !ok {
				continue
			}
			b.ID = id
			b.Local = local
			seenID[id] = true

			// Second-level dedup: same underlying binary under a
			// different desktop file id (the Yandex case). Prefer the
			// entry from a local (user) dir over a system one.
			if idx, exists := seenBinary[b.BinaryPath]; exists {
				if b.Local && !result[idx].Local {
					result[idx] = b
				}
				continue
			}
			seenBinary[b.BinaryPath] = len(result)
			result = append(result, b)
		}
	}
	return result, nil
}

// parseDesktopFile reads the [Desktop Entry] group of a .desktop file
// and returns a Browser if it handles https links. This is a minimal,
// purpose-built parser rather than a general INI reader: desktop files
// have quirks (locale-suffixed keys like Name[es], multiple groups for
// actions) that a generic parser would need special-casing for anyway.
func parseDesktopFile(path string) (Browser, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return Browser{}, false, err
	}
	defer f.Close()

	var b Browser
	b.Path = path
	inDesktopEntry := false
	handlesHTTPS := false
	isApplication := true

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			inDesktopEntry = (line == "[Desktop Entry]")
			continue
		}
		if !inDesktopEntry {
			continue
		}

		key, val, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)

		switch key {
		case "Name": // unsuffixed key only; skip Name[xx] localizations
			b.Name = val
		case "Icon":
			b.Icon = val
		case "Exec":
			b.Exec = val
			b.BinaryPath = firstToken(val)
		case "MimeType":
			if strings.Contains(val, "x-scheme-handler/https") {
				handlesHTTPS = true
			}
		case "NoDisplay":
			if val == "true" {
				isApplication = false
			}
		case "Type":
			if val != "Application" {
				isApplication = false
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return Browser{}, false, err
	}
	return b, handlesHTTPS && isApplication, nil
}

// firstToken extracts the binary path from an Exec= value, stripping
// arguments and desktop field codes (%U, %u, %F, etc.), so that two
// Exec lines invoking the same binary with different flags resolve to
// the same identity for dedup purposes.
func firstToken(exec string) string {
	fields := strings.Fields(exec)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}
