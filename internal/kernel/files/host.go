package files

import (
	"os/user"
	"strconv"
	"sync"
)

var (
	hostUIDOnce sync.Once
	hostGIDOnce sync.Once
	hostUID     int
	hostGID     int
)

// uid returns the current process UID, cached after first lookup.
// Returns 0 if lookup fails.
func uid() int {
	hostUIDOnce.Do(func() {
		u, err := user.Current()
		if err != nil {
			return
		}
		if n, err := strconv.Atoi(u.Uid); err == nil {
			hostUID = n
		}
	})
	return hostUID
}

// gid returns the current process GID, cached after first lookup.
// Returns 0 if lookup fails.
func gid() int {
	hostGIDOnce.Do(func() {
		u, err := user.Current()
		if err != nil {
			return
		}
		if n, err := strconv.Atoi(u.Gid); err == nil {
			hostGID = n
		}
	})
	return hostGID
}
