module browser-switcher

go 1.22

require github.com/godbus/dbus/v5 v5.1.0

// golang.org/x/sys's module-path redirect isn't reachable from this
// sandboxed build environment; pull the identical source straight from
// its GitHub mirror instead. Not needed on a normal dev machine with
// unrestricted network access — safe to delete this line there.
replace golang.org/x/sys => github.com/golang/sys v0.27.0
