package files

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"sync"

	"golang.org/x/sys/unix"
)

var (
	ownerCache = &sync.Map{} // Cache for user names by UID.
	groupCache = &sync.Map{} // Cache for group names by GID.
)

// Stats interface provides operations to manage and query file or directory metadata..
type Stats interface {
	IsReadable() bool
	IsWritable() bool
	IsExecutable() bool
	HasPermissions(permissions os.FileMode) bool
	Owner() UserInfo
	Group() GroupInfo
	Other() OtherInfo
	Chmod(permissions uint32) error
	Chown(uid, gid int) error
	IsFile() bool
	IsDirectory() bool
	Exists() bool
	Refresh() error
}

type stats struct {
	path   string       // Path to the file or directory
	exists bool         // Indicates if the file or directory exists
	mode   uint32       // Combined permissions
	meta   *unix.Stat_t // File metadata

	user  UserInfo  // File owner information
	group GroupInfo // Group information
	other OtherInfo // Other users' information
}

// NewStats creates and initializes a stats object for a given path..
//
// Parameters:
// - path: string - The path to the file or directory.
//
// Returns:
// - *stats: A pointer to the initialized stats object.
func NewStats(path string) *stats {
	s := &stats{
		path: path,
		meta: &unix.Stat_t{},
	}

	_ = s.Refresh() // Ignore refresh error in constructor

	return s
}

// UserInfo contains the owner information of a file or directory..
type UserInfo struct {
	ID          uint32      // UID of the user
	Permissions Permissions // Permissions: RWX bits (read, write, execute)
	name        string      // Name of the user
}

// Name retrieves the name of the user associated with the UID.
//
// Returns:
// - string: The user name.
func (u UserInfo) Name() string {
	if u.name == "" {
		if cachedName, ok := ownerCache.Load(u.ID); ok {
			u.name = cachedName.(string)
		} else {
			u.name = lookupUser(u.ID)
		}
	}
	return u.name
}

// GroupInfo contains the group information of a file or directory..
type GroupInfo struct {
	ID          uint32      // GID of the group
	Permissions Permissions // Permissions: RWX bits
	name        string      // Name of the group
}

// Name retrieves the name of the group associated with the GID.
//
// Returns:
// - string: The group name.
func (u GroupInfo) Name() string {
	if u.name == "" {
		if cachedName, ok := groupCache.Load(u.ID); ok {
			u.name = cachedName.(string)
		} else {
			u.name = lookupGroup(u.ID)
		}
	}
	return u.name
}

// OtherInfo contains the permissions for other users of a file or directory..
type OtherInfo struct {
	Permissions Permissions // Permissions: RWX bits
}

type Permissions struct {
	Read, Write, Exec bool // Boolean flags for read, write, and execute permissions.
}

// HasPermissions checks if the file or directory has specific permissions.
//
// Parameters:
// - permissions: os.FileMode - The permissions to check.
//
// Returns:
// - bool: True if the file has the specified permissions, otherwise false.
func (s *stats) HasPermissions(permissions os.FileMode) bool {
	if permissions > os.ModePerm {
		return false // Invalid permissions
	}
	return s.mode&uint32(permissions) == uint32(permissions)
}

// IsReadable checks if the file or directory is readable by a specific user.
//
// Parameters:
// - uid: uint32 - User ID to check.
// - gid: uint32 - Group ID to check.
//
// Returns:
// - bool: True if the file is readable by the user or group, otherwise false.
func (s *stats) IsReadable(uid, gid uint32) bool {
	if s.user.ID == uid {
		return s.user.Permissions.Read
	} else if s.group.ID == gid {
		return s.group.Permissions.Read
	}
	return s.other.Permissions.Read
}

// IsWritable checks if the file or directory is writable by a specific user.
//
// Parameters:
// - uid: uint32 - User ID to check.
// - gid: uint32 - Group ID to check.
//
// Returns:
// - bool: True if the file is writable by the user or group, otherwise false.
func (s *stats) IsWritable(uid, gid uint32) bool {
	if s.user.ID == uid {
		return s.user.Permissions.Write
	} else if s.group.ID == gid {
		return s.group.Permissions.Write
	}
	return s.other.Permissions.Write
}

// IsExecutable checks if the file or directory is executable by a specific user.
//
// Parameters:
// - uid: uint32 - User ID to check.
// - gid: uint32 - Group ID to check.
//
// Returns:
// - bool: True if the file is executable by the user or group, otherwise false.
func (s *stats) IsExecutable(uid, gid uint32) bool {
	if s.user.ID == uid {
		return s.user.Permissions.Exec
	} else if s.group.ID == gid {
		return s.group.Permissions.Exec
	}
	return s.other.Permissions.Exec
}

// IsFile checks if the path corresponds to a regular file.
//
// Returns:
// - bool: True if the path is a regular file, otherwise false.
func (s *stats) IsFile() bool {
	return s.mode&unix.S_IFREG != 0
}

// IsDirectory checks if the path corresponds to a directory.
//
// Returns:
// - bool: True if the path is a directory, otherwise false.
func (s *stats) IsDirectory() bool {
	return s.mode&unix.S_IFDIR != 0
}

// Exists checks if the file or directory exists.
//
// Returns:
// - bool: True if the file or directory exists, otherwise false.
func (s *stats) Exists() bool {
	return s.exists
}

// Owner retrieves the owner information of the file or directory.
//
// Returns:
// - UserInfo: The owner information (UID, permissions, and name).
func (s *stats) Owner() UserInfo {
	return s.user
}

// Group retrieves the group information of the file or directory.
//
// Returns:
// - GroupInfo: The group information (GID, permissions, and name).
func (s *stats) Group() GroupInfo {
	return s.group
}

// Other retrieves the permissions information for other users of the file or directory.
//
// Returns:
// - OtherInfo: The permissions for other users (read, write, execute).
func (s *stats) Other() OtherInfo {
	return s.other
}

// Chmod changes the permissions of the file or directory.
//
// Parameters:
// - permissions: uint32 - The new permissions to set.
//
// Returns:
// - error: Error if the operation fails, otherwise nil.
func (s *stats) Chmod(permissions uint32) error {
	if err := unix.Chmod(s.path, permissions); err != nil {
		return fmt.Errorf("failed to chmod %s: %w", s.path, err)
	}
	return nil
}

// Chown changes the owner and group of the file or directory.
//
// Parameters:
// - uid: int - New user ID.
// - gid: int - New group ID.
//
// Returns:
// - error: Error if the operation fails, otherwise nil.
func (s *stats) Chown(uid, gid int) error {
	if uid < 0 || gid < 0 || uid > 0x7FFFFFFF || gid > 0x7FFFFFFF {
		return unix.EINVAL
	}
	if err := unix.Chown(s.path, uid, gid); err != nil {
		return fmt.Errorf("failed to chown %s: %w", s.path, err)
	}
	return nil
}

// Refresh reloads the file or directory metadata.
//
// Returns:
// - error: Error if the operation fails, otherwise nil.
func (s *stats) Refresh() error {
	// Perform Lstat to retrieve basic metadata.
	err := unix.Lstat(s.path, s.meta)
	if err != nil {
		s.exists = false
		return fmt.Errorf("failed to stat file %s: %w", s.path, err)
	}

	// Check if the file is a symbolic link.
	if s.meta.Mode&unix.S_IFMT == unix.S_IFLNK {
		// Resolve symbolic links only if necessary.
		resolvedPath, err := filepath.EvalSymlinks(s.path)
		if err != nil {
			return fmt.Errorf("failed to resolve symlink %s: %w", s.path, err)
		}
		s.path = resolvedPath
		// Re-stat the resolved target
		targetInfo, err := os.Stat(resolvedPath)
		if err != nil {
			return err
		}
		s.mode = uint32(targetInfo.Mode())
		if stat, ok := targetInfo.Sys().(*unix.Stat_t); ok {
			s.meta = stat
		}
	}

	// Mark the file as existing.
	s.exists = true

	// Set direct permissions to avoid repeated calls.
	s.mode = uint32(s.meta.Mode)
	s.group.ID = s.meta.Gid
	s.user.ID = s.meta.Uid

	// Calculate permissions inline.
	s.user.Permissions = s.calculatePermissions(unix.S_IRUSR, unix.S_IWUSR, unix.S_IXUSR)
	s.group.Permissions = s.calculatePermissions(unix.S_IRGRP, unix.S_IWGRP, unix.S_IXGRP)
	s.other.Permissions = s.calculatePermissions(unix.S_IROTH, unix.S_IWOTH, unix.S_IXOTH)

	return nil
}

// calculatePermissions computes the read, write, and execute permissions for a user, group, or others.
//
// Parameters:
// - readMask: uint32 - Bitmask for read permission.
// - writeMask: uint32 - Bitmask for write permission.
// - execMask: uint32 - Bitmask for execute permission.
//
// Returns:
// - Permissions: The computed permissions.
func (s *stats) calculatePermissions(readMask, writeMask, execMask uint32) Permissions {
	return Permissions{
		Read:  s.mode&readMask != 0,
		Write: s.mode&writeMask != 0,
		Exec:  s.mode&execMask != 0,
	}
}

// _lookupCache retrieves a cached value or performs a lookup if not cached.
//
// Parameters:
// - id: uint32 - The ID (UID or GID) to look up.
// - cache: *sync.Map - The cache to use for storing and retrieving values.
// - lookup: func(string) string - A function to perform the lookup if the value is not cached.
//
// Returns:
// - string: The cached or looked-up value.
func _lookupCache(id uint32, cache *sync.Map, lookup func(string) string) string {
	if name, found := cache.Load(id); found {
		return name.(string)
	}

	result := lookup(strconv.Itoa(int(id)))
	cache.Store(id, result)
	return result
}

// lookupUser retrieves the username associated with a given UID.
//
// Parameters:
// - uid: uint32 - User ID to look up.
//
// Returns:
// - string: The username, or an empty string if not found.
func lookupUser(uid uint32) string {
	return _lookupCache(uid, ownerCache, func(id string) string {
		usr, err := user.LookupId(id)
		if err != nil {
			return ""
		}
		return usr.Username
	})
}

// lookupGroup retrieves the group name associated with a given GID.
//
// Parameters:
// - gid: uint32 - Group ID to look up.
//
// Returns:
// - string: The group name, or an empty string if not found.
func lookupGroup(gid uint32) string {
	return _lookupCache(gid, groupCache, func(id string) string {
		grp, err := user.LookupGroupId(id)
		if err != nil {
			return ""
		}
		return grp.Name
	})
}
