// Command browser-switcher is a tray icon that lets you pick the
// system default web browser, built without any GTK dependency: the
// tray icon and menu are implemented directly against the freedesktop
// StatusNotifierItem / com.canonical.dbusmenu D-Bus specs.
//
// Requires a StatusNotifierHost to actually show the icon — on niri
// that's typically waybar's `tray` module (or snixembed for other bars).
package main

import (
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/godbus/dbus/v5"

	"browser-switcher/browser"
	"browser-switcher/tray"
	"browser-switcher/xdgbrowser"
)

const menuObjectPath = dbus.ObjectPath("/MenuBar")

func main() {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		log.Fatalf("connecting to session bus: %v", err)
	}
	defer conn.Close()

	menu, err := tray.NewMenu(conn, menuObjectPath)
	if err != nil {
		log.Fatalf("exporting menu: %v", err)
	}

	item, err := tray.NewItem(conn, "web-browser", "Default browser", menu)
	if err != nil {
		log.Fatalf("exporting tray item: %v", err)
	}
	_ = item

	// entryToID maps menu entry id -> browser id, since dbusmenu ids
	// must be int32 but browsers are keyed by their .desktop file id.
	var entryToID map[int32]string

	refresh := func() {
		browsers, err := browser.Discover()
		if err != nil {
			log.Printf("scanning browsers: %v", err)
			return
		}
		sort.Slice(browsers, func(i, j int) bool {
			return browsers[i].Name < browsers[j].Name
		})

		current, err := xdgbrowser.Current()
		if err != nil {
			log.Printf("reading current default browser: %v", err)
		}

		entryToID = make(map[int32]string, len(browsers))
		entries := make([]tray.MenuEntry, 0, len(browsers))
		for i, b := range browsers {
			id := int32(i + 1) // 0 is reserved for the layout root
			entryToID[id] = b.ID
			entries = append(entries, tray.MenuEntry{
				ID:       id,
				Label:    b.Name,
				IconName: b.Icon,
				Checked:  b.ID == current,
			})
		}
		menu.SetEntries(entries)
	}

	menu.OnSelect = func(entryID int32) {
		browserID, ok := entryToID[entryID]
		if !ok {
			return
		}
		if err := xdgbrowser.SetDefault(browserID); err != nil {
			log.Printf("setting default browser to %s: %v", browserID, err)
			return
		}
		// Rescan so the checkmark moves to the new default immediately.
		refresh()
	}

	refresh()

	fmt.Fprintln(os.Stderr, "browser-switcher running; waiting for tray host to display the icon")
	select {} // block forever; all work happens in D-Bus callbacks above
}
