package fs

import (
	"path/filepath"
	"strings"

	"github.com/kitsunium/sdk/pkg/lib/pointer"
)

// Path represents a file or directory path with convenience methods.
//
// This type wraps string paths and provides utility methods for path
// manipulation, cleaning, and parent directory resolution.
type Path string

// Parent returns the parent directory of the path.
// Uses filepath.Dir to resolve the parent path.
func (p Path) Parent() string {
	return filepath.Dir(string(p))
}

// Clean returns the cleaned and normalized path.
// Removes extra slashes, resolves . and .. elements, and trims whitespace.
func (p Path) Clean() string {
	return filepath.Clean(strings.TrimSpace(string(p)))
}

// String returns the string representation of the path.
func (p Path) String() string {
	return string(p)
}

// Option represents configuration options for creating or manipulating files and directories.
//
// This struct provides comprehensive configuration for file system operations,
// including permissions, ownership, creation behavior, and I/O optimization settings.
// All pointer fields (Chmod, UID, GID, BufferSize) allow for optional configuration
// where nil values result in system defaults being used.
type Option struct {
	Path             string  // The file or directory path.
	Chmod            *uint32 // The permission mode to apply (e.g., 0644).
	UID              *int    // The user ID to assign ownership.
	GID              *int    // The group ID to assign ownership.
	CreateIfNotExist bool    // If true, the file or directory will be created if it does not exist.
	Overwrite        bool    // If true, existing files will be overwritten.
	PreserveTimes    bool    // If true, the original access and modification times will be preserved.
	BufferSize       *int    // The buffer size to use when reading or writing files.
}

// Validate checks and normalizes the Option configuration.
//
// This method:
//   - Cleans and validates the file path
//   - Sets default values for nil pointer fields
//   - Ensures the configuration is consistent and valid
//
// Returns true if the option is valid and has been normalized,
// false if the configuration is invalid (e.g., empty or invalid path).
func (o *Option) Validate() bool {
	o.Path = Path(o.Path).Clean()
	if o.Path == "." {
		return false
	}

	if o.Chmod == nil {
		o.Chmod = pointer.Uint32(DefaultFilePerm)
	}

	if o.UID == nil {
		o.UID = pointer.Int(uid)
	}

	if o.GID == nil {
		o.GID = pointer.Int(gid)
	}

	if o.BufferSize == nil {
		o.BufferSize = pointer.Int(32 * 1024)
	}

	return true
}

func (o Option) parent() string {
	return filepath.Dir(o.Path)
}
