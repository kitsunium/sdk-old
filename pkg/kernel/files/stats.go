//go:build !windows

// Package files provides file system abstractions for the Kitsunium kernel layer.
// It exposes File, Directory, and Stats interfaces backed by pure Go stdlib.
package files

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
)

// maxUID is the maximum valid POSIX UID/GID value (2^31 - 1).
const maxUID = 0x7FFFFFFF

var (
	ownerCache = &sync.Map{} // UID → user name
	groupCache = &sync.Map{} // GID → group name
)

// POSIX permission bit masks (stdlib `syscall` exposes only a subset; we
// inline the canonical values from <sys/stat.h> to avoid adding a dep).
const (
	sIFMT  uint32 = 0o170000
	sIFREG uint32 = 0o100000
	sIFDIR uint32 = 0o040000
	sIFLNK uint32 = 0o120000

	sIRUSR uint32 = 0o000400
	sIWUSR uint32 = 0o000200
	sIXUSR uint32 = 0o000100
	sIRGRP uint32 = 0o000040
	sIWGRP uint32 = 0o000020
	sIXGRP uint32 = 0o000010
	sIROTH uint32 = 0o000004
	sIWOTH uint32 = 0o000002
	sIXOTH uint32 = 0o000001
)

// PermissionSet is a decoded view of a `rwx` triple.
//
// It carries the read, write, and execute flags for a single POSIX
// permission class (owner, group, or other).
type PermissionSet struct {
	//: individual permission flags decoded from the mode bits
	Read, Write, Exec bool
}

// UserEntity carries the owning user's identity and permission bits.
//
// ID is the POSIX UID. Permissions holds the decoded rwx bits.
// Name is resolved lazily from the system user database.
type UserEntity struct {
	//: user identity fields populated by Refresh
	ID          uint32
	Permissions PermissionSet
	name        string
}

// Name returns the cached user name, resolving lazily on first call.
//
// Returns:
//   - string: the user name or empty string if lookup fails.
func (ui UserEntity) Name() (result string) {
	if ui.name == "" {
		ui.name = lookupUser(ui.ID)
	}
	//: return resolved or cached name
	return ui.name
}

// GroupEntity carries the owning group's identity and permission bits.
//
// ID is the POSIX GID. Permissions holds the decoded rwx bits.
// Name is resolved lazily from the system group database.
type GroupEntity struct {
	//: group identity fields populated by Refresh
	ID          uint32
	Permissions PermissionSet
	name        string
}

// Name returns the cached group name, resolving lazily on first call.
//
// Returns:
//   - string: the group name or empty string if lookup fails.
func (gi GroupEntity) Name() (result string) {
	if gi.name == "" {
		gi.name = lookupGroup(gi.ID)
	}
	//: return resolved or cached name
	return gi.name
}

// OtherEntity carries the "other" permission bits.
//
// It represents the third POSIX permission class (world/other).
type OtherEntity struct {
	//: other-class permission bits
	Permissions PermissionSet
}

// Stats exposes filesystem metadata for a path.
type Stats interface {
	IsReadable(uid, gid uint32) bool
	IsWritable(uid, gid uint32) bool
	IsExecutable(uid, gid uint32) bool
	HasPermissions(permissions os.FileMode) bool
	Owner() UserEntity
	Group() GroupEntity
	Other() OtherEntity
	Chmod(permissions uint32) error
	Chown(uid, gid int) error
	IsFile() bool
	IsDirectory() bool
	Exists() bool
	Refresh() error
}

// fileMeta is the subset of stat information we consume.
type fileMeta struct {
	Mode uint32
	UID  uint32
	GID  uint32
	Size int64
}

type statsImpl struct {
	path   string
	exists bool
	mode   uint32
	meta   *fileMeta

	user  UserEntity
	group GroupEntity
	other OtherEntity
}

// NewStats returns a *statsImpl initialized from the given path.
//
// Missing paths yield a Stats with Exists() == false and no error.
//
// Params:
//   - path: the filesystem path to stat.
//
// Returns:
//   - Stats: initialized stats handle (never nil).
func NewStats(path string) Stats {
	si := &statsImpl{
		path: path,
		meta: &fileMeta{},
	}
	//: ignore error; caller uses Exists() to detect missing paths
	_ = si.Refresh()
	return si
}

// HasPermissions reports whether the permission bits in `permissions` are all set.
//
// Params:
//   - permissions: the os.FileMode bits to check.
//
// Returns:
//   - bool: true when all requested bits are set.
func (si *statsImpl) HasPermissions(permissions os.FileMode) (ok bool) {
	//: reject modes that exceed the valid 0o777 range
	if permissions > os.ModePerm {
		return false
	}
	//: bitwise intersection must equal the requested mask
	return si.mode&uint32(permissions) == uint32(permissions)
}

// IsReadable reports whether the given (uid, gid) can read the file.
//
// It applies the standard POSIX lookup: owner → group → other.
//
// Params:
//   - uid: the user ID to check.
//   - gid: the group ID to check.
//
// Returns:
//   - bool: true when the identity has read access.
func (si *statsImpl) IsReadable(uid, gid uint32) (ok bool) {
	switch {
	case si.user.ID == uid:
		//: owner match — use owner read bit
		return si.user.Permissions.Read
	case si.group.ID == gid:
		//: group match — use group read bit
		return si.group.Permissions.Read
	default:
		//: fall back to other read bit
		return si.other.Permissions.Read
	}
}

// IsWritable reports whether the given (uid, gid) can write the file.
//
// Params:
//   - uid: the user ID to check.
//   - gid: the group ID to check.
//
// Returns:
//   - bool: true when the identity has write access.
func (si *statsImpl) IsWritable(uid, gid uint32) (ok bool) {
	switch {
	case si.user.ID == uid:
		//: owner match — use owner write bit
		return si.user.Permissions.Write
	case si.group.ID == gid:
		//: group match — use group write bit
		return si.group.Permissions.Write
	default:
		//: fall back to other write bit
		return si.other.Permissions.Write
	}
}

// IsExecutable reports whether the given (uid, gid) can execute the file.
//
// Params:
//   - uid: the user ID to check.
//   - gid: the group ID to check.
//
// Returns:
//   - bool: true when the identity has execute access.
func (si *statsImpl) IsExecutable(uid, gid uint32) (ok bool) {
	switch {
	case si.user.ID == uid:
		//: owner match — use owner exec bit
		return si.user.Permissions.Exec
	case si.group.ID == gid:
		//: group match — use group exec bit
		return si.group.Permissions.Exec
	default:
		//: fall back to other exec bit
		return si.other.Permissions.Exec
	}
}

// IsFile reports whether the path is a regular file.
//
// Returns:
//   - bool: true when the mode indicates a regular file.
func (si *statsImpl) IsFile() (ok bool) {
	//: check file type bits against regular-file mask
	return si.mode&sIFMT == sIFREG
}

// IsDirectory reports whether the path is a directory.
//
// Returns:
//   - bool: true when the mode indicates a directory.
func (si *statsImpl) IsDirectory() (ok bool) {
	//: check file type bits against directory mask
	return si.mode&sIFMT == sIFDIR
}

// Exists reports whether the last Refresh found the path on disk.
//
// Returns:
//   - bool: true when the path exists.
func (si *statsImpl) Exists() (ok bool) {
	//: return cached existence flag
	return si.exists
}

// Owner returns a snapshot of the owning user's UserEntity.
//
// Returns:
//   - UserEntity: the owning user's identity and permissions.
func (si *statsImpl) Owner() (result UserEntity) {
	//: return snapshot copy to avoid mutation
	return si.user
}

// Group returns a snapshot of the owning group's GroupEntity.
//
// Returns:
//   - GroupEntity: the owning group's identity and permissions.
func (si *statsImpl) Group() (result GroupEntity) {
	//: return snapshot copy to avoid mutation
	return si.group
}

// Other returns a snapshot of the "other" class's OtherEntity.
//
// Returns:
//   - OtherEntity: the other-class permission bits.
func (si *statsImpl) Other() (result OtherEntity) {
	//: return snapshot copy to avoid mutation
	return si.other
}

// Chmod changes the file mode.
//
// Params:
//   - permissions: new permission bits to apply.
//
// Returns:
//   - error: wrapped error if chmod fails.
func (si *statsImpl) Chmod(permissions uint32) (err error) {
	//: guard against chmod failure
	if chmodErr := os.Chmod(si.path, os.FileMode(permissions)); chmodErr != nil {
		return fmt.Errorf("failed to chmod %s: %w", si.path, chmodErr)
	}
	//: success path
	return nil
}

// Chown changes the owning user and group.
//
// Params:
//   - uid: new user ID to assign.
//   - gid: new group ID to assign.
//
// Returns:
//   - error: syscall.EINVAL for out-of-range IDs or wrapped OS error.
func (si *statsImpl) Chown(uid, gid int) (err error) {
	//: validate UID/GID range before syscall
	if uid < 0 || gid < 0 || uid > maxUID || gid > maxUID {
		return syscall.EINVAL
	}
	//: delegate to os.Chown with range-validated values
	if chownErr := os.Chown(si.path, uid, gid); chownErr != nil {
		return fmt.Errorf("failed to chown %s: %w", si.path, chownErr)
	}
	//: success path
	return nil
}

// Refresh re-reads stat info from disk and repopulates cached fields.
//
// Returns:
//   - error: wrapped error if stat or symlink resolution fails.
func (si *statsImpl) Refresh() (err error) {
	info, statErr := os.Lstat(si.path)
	//: mark missing on any stat error
	if statErr != nil {
		si.exists = false
		return fmt.Errorf("failed to stat file %s: %w", si.path, statErr)
	}

	//: resolve symlinks so the rest of the object reflects the target
	if info.Mode()&os.ModeSymlink != 0 {
		resolved, resolveErr := filepath.EvalSymlinks(si.path)
		//: fail on unresolvable symlink
		if resolveErr != nil {
			return fmt.Errorf("failed to resolve symlink %s: %w", si.path, resolveErr)
		}
		si.path = resolved
		info, statErr = os.Stat(resolved)
		//: guard after stat of resolved target
		if statErr != nil {
			return fmt.Errorf("failed to stat resolved path %s: %w", resolved, statErr)
		}
	}

	sys, ok := info.Sys().(*syscall.Stat_t)
	//: platform must provide a *syscall.Stat_t
	if !ok {
		return errors.New(fmt.Sprintf("unsupported FileInfo.Sys() for %s", si.path))
	}

	si.exists = true
	si.meta.Mode = sys.Mode
	si.meta.UID = sys.Uid
	si.meta.GID = sys.Gid
	si.meta.Size = info.Size()

	si.mode = si.meta.Mode
	si.user.ID = si.meta.UID
	si.group.ID = si.meta.GID

	si.user.Permissions = si.permissionSet(sIRUSR, sIWUSR, sIXUSR)
	si.group.Permissions = si.permissionSet(sIRGRP, sIWGRP, sIXGRP)
	si.other.Permissions = si.permissionSet(sIROTH, sIWOTH, sIXOTH)
	//: all fields refreshed successfully
	return nil
}

// permissionSet decodes a rwx triple from the cached mode bits.
//
// Params:
//   - readMask: bit mask for the read permission.
//   - writeMask: bit mask for the write permission.
//   - execMask: bit mask for the execute permission.
//
// Returns:
//   - PermissionSet: the decoded permission triple.
func (si *statsImpl) permissionSet(readMask, writeMask, execMask uint32) (ps PermissionSet) {
	//: decode each bit independently
	return PermissionSet{
		Read:  si.mode&readMask != 0,
		Write: si.mode&writeMask != 0,
		Exec:  si.mode&execMask != 0,
	}
}

// lookupCached is the shared cached-resolver for uid/gid → name.
//
// Params:
//   - id: the numeric UID or GID.
//   - cache: the sync.Map used for caching resolved names.
//   - resolve: function that resolves a string ID to a name.
//
// Returns:
//   - string: the resolved name or empty string on failure.
func lookupCached(id uint32, cache *sync.Map, resolve func(string) string) (result string) {
	//: fast path — return cached value if present
	if name, ok := cache.Load(id); ok {
		cached, assertOK := name.(string)
		//: guard against unexpected non-string value
		if assertOK {
			return cached
		}
	}
	result = resolve(strconv.Itoa(int(id)))
	cache.Store(id, result)
	//: return freshly resolved name
	return result
}

// lookupUser resolves a UID to a username using the system user database.
//
// Params:
//   - uid: the POSIX user ID to resolve.
//
// Returns:
//   - string: the username or empty string if lookup fails.
func lookupUser(uid uint32) (result string) {
	//: delegate to cached resolver
	return lookupCached(uid, ownerCache, func(id string) string {
		userInfo, lookupErr := user.LookupId(id)
		//: return empty string on any lookup failure
		if lookupErr != nil {
			return ""
		}
		//: return resolved username
		return userInfo.Username
	})
}

// lookupGroup resolves a GID to a group name using the system group database.
//
// Params:
//   - gid: the POSIX group ID to resolve.
//
// Returns:
//   - string: the group name or empty string if lookup fails.
func lookupGroup(gid uint32) (result string) {
	//: delegate to cached resolver
	return lookupCached(gid, groupCache, func(id string) string {
		groupInfo, lookupErr := user.LookupGroupId(id)
		//: return empty string on any lookup failure
		if lookupErr != nil {
			return ""
		}
		//: return resolved group name
		return groupInfo.Name
	})
}
