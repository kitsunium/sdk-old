package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

const (
	DefaultDirPerm uint32 = 0755 // Default permission for directory creation.
)

// Directory interface defines operations for directory management.
//
// This interface provides high-performance directory operations using direct
// Unix system calls for optimal kernel-level performance. All operations
// are designed to minimize allocations and provide fast execution.
type Directory interface {
	Path() string
	Parent() (Directory, error)
	Create() (Directory, error)
	Remove() error
	Exists() bool
	Has(string) bool
	Size() int64
	List() ([]File, []Directory, error)
}

// directory struct represents a directory with its path and metadata.
// It encapsulates directory-specific operations and maintains file system
// statistics for performance monitoring.
type directory struct {
	path       string
	parentPath string
	chmod      *uint32
	uid        *int
	gid        *int
	stats      *stats
}

// NewDirectory creates a new Directory object based on the given options.
//
// This function validates the provided options and creates a directory instance
// with the specified permissions, ownership, and creation behavior. If the
// directory already exists and is not a directory, it returns os.ErrInvalid.
//
// Example:
//
//	option := Option{
//		Path:             "/tmp/mydir",
//		CreateIfNotExist: true,
//		mode := uint32(0755)
//		Chmod:            &mode,
//	}
//	dir, err := NewDirectory(option)
//	if err != nil {
//		return err
//	}
//
// Returns an error if the option validation fails or if the path exists
// but is not a directory.
func NewDirectory(option Option) (Directory, error) {
	if !option.Validate() {
		return nil, fmt.Errorf("invalid option: %v", option)
	}

	stats := NewStats(option.Path)
	exists := stats.Exists()

	if exists && !stats.IsDirectory() {
		return nil, os.ErrInvalid
	}

	d := &directory{
		path:       option.Path,
		parentPath: option.parent(),
		chmod:      option.Chmod,
		uid:        option.UID,
		gid:        option.GID,
		stats:      stats,
	}

	if !exists && option.CreateIfNotExist {
		return d.Create()
	}

	return d, nil
}

// Path returns the directory's path.
func (d *directory) Path() string {
	return d.path
}

// Has checks if a given System (file or directory) is within the current directory.
func (d *directory) Has(s string) bool {
	dirPath := d.path
	sysPath := s

	// Check if the directory path is a prefix of the system path.
	// and ensure it's a proper directory boundary using `filepath.Rel`.
	rel, err := filepath.Rel(dirPath, sysPath)
	if err != nil {
		return false
	}

	// `rel` will not contain ".." if sysPath is within dirPath.
	return !strings.HasPrefix(rel, "..")
}

// Parent retrieves the parent directory of the current directory.
func (d *directory) Parent() (Directory, error) {
	// Utiliser filepath.Dir pour calculer le chemin du parent.
	parentPath := filepath.Dir(d.path)

	// Si le chemin du parent est identique au chemin courant, c'est la racine.
	if parentPath == d.path {
		return nil, fmt.Errorf("no parent directory found for %s", d.path)
	}

	return &directory{
		path:  parentPath,
		stats: NewStats(parentPath),
	}, nil
}

// Create ensures the directory exists, creating it if necessary, and sets permissions and ownership.
//
// Returns:
// - Directory: The created or updated directory object.
// - error: Error if the operation fails.
func (d *directory) Create() (Directory, error) {
	if err := unix.Mkdir(d.path, *d.chmod); err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Update permissions and ownership if specified.
	if err := unix.Chmod(d.path, *d.chmod); err != nil {
		return nil, fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := unix.Chown(d.path, *d.uid, *d.gid); err != nil {
		return nil, fmt.Errorf("failed to set ownership: %w", err)
	}

	// Refresh directory stats.
	d.stats = NewStats(d.path)
	return d, nil
}

// Remove deletes the directory.
func (d *directory) Remove() error {
	if err := os.RemoveAll(d.path); err != nil {
		return fmt.Errorf("failed to remove directory %s: %w", d.path, err)
	}
	return nil
}

// Exists checks if the directory exists.
func (d *directory) Exists() bool {
	_ = d.stats.Refresh() // Ignore refresh error for existence check
	return d.stats.Exists()
}

// Size calculates the total size of all files in the directory (recursively).
func (d *directory) Size() int64 {
	var totalSize int64

	// Walk through all files in the directory
	err := filepath.Walk(d.path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files/dirs we can't access
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		// Fallback to directory's own size if walk fails
		_ = d.stats.Refresh()
		return d.stats.meta.Size
	}

	return totalSize
}

// List retrieves all files and subdirectories within the directory.
//
// Returns:
// - []File: A slice of files in the directory.
// - []Directory: A slice of subdirectories in the directory.
// - error: Error if the operation fails.
// List retrieves all files and subdirectories within the directory.
func (d *directory) List() ([]File, []Directory, error) {
	dirEntries, err := os.ReadDir(d.path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list directory contents: %w", err)
	}

	var files []File
	var directories []Directory

	for _, entry := range dirEntries {
		entryPath := fmt.Sprintf("%s/%s", d.path, entry.Name())
		stats := NewStats(entryPath)

		if entry.IsDir() {
			directories = append(directories, &directory{
				path:  entryPath,
				stats: stats,
			})
		} else if stats.IsFile() {
			files = append(files, &file{
				path:  entryPath,
				stats: stats,
			})
		}
	}

	return files, directories, nil
}
