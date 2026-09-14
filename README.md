# browser-switcher

Tray icon to change the default web browser on Linux without depending
on GTK. The icon and menu talk directly to the freedesktop
StatusNotifierItem and com.canonical.dbusmenu D-Bus interfaces, so the
only runtime dependency is D-Bus itself.

## How it works

- On start (and after every change), it scans `.desktop` files under
  `$XDG_DATA_HOME/applications` and each `$XDG_DATA_DIRS/applications`
  dir, keeping only entries that declare `x-scheme-handler/https`.
- If two entries resolve to the same underlying binary (a system
  package entry and a user override, for example), the one in
  `XDG_DATA_HOME` wins.
- The menu lists the discovered browsers with a radio mark on the
  current default (read via `xdg-settings get default-web-browser`).
- Clicking one runs `xdg-settings set default-web-browser <id>.desktop`
  and rescans.

## Requirements

- A StatusNotifierHost running to actually display the icon — e.g.
  waybar's `tray` module, or `snixembed` if your bar doesn't support
  StatusNotifierItem. Without one, the app runs but the icon shows
  nowhere, and it logs an error saying registration with the watcher
  failed.
- `xdg-settings` (part of `xdg-utils`, present on virtually every
  distro).

## Build

```
make build
```

## Install

```
sudo make install
```

Installs to `/usr/local/bin` by default. Override the prefix with
`sudo make install PREFIX=/usr`.

## Uninstall

```
sudo make uninstall
```

## Running at login

Not handled by `make install`. Add it to your compositor's
`spawn-at-startup` (niri), a systemd `--user` unit, or your bar's
autostart config, depending on your setup.

## License

AGPLv3 — see [LICENSE](LICENSE).
