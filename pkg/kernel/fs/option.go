package fs

import (
	"path/filepath"
	"strings"

	"github.com/kistunium/sdk/pkg/lib/pointer"
)

// Path represents a file or directory path.
type Path string

func (p Path) Parent() string {
	return filepath.Dir(string(p))
}

func (p Path) Clean() string {
	return filepath.Clean(strings.TrimSpace(string(p)))
}

func (p Path) String() string {
	return string(p)
}

// Option represents configuration options for creating or manipulating files and directories.
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

func (o *Option) Validate() bool {
	o.Path = Path(o.Path).Clean()
	if o.Path == "." || o.Path != o.Path {
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
