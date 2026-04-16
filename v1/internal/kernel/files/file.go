// Package files provides file system operations for the Kitsunium kernel.
//
// All file I/O uses the Go standard library (`os`, `io`, `path/filepath`)
// rather than direct syscalls. The previous implementation used
// golang.org/x/sys/unix and a hand-rolled goroutine-piped Copy; both have
// been replaced with the obvious io.Copy / os.OpenFile equivalents.
package files

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DefaultFilePerm is the default permission for file creation.
const DefaultFilePerm uint32 = 0644

// File is a high-level view over a filesystem file.
type File interface {
	Path() string
	Parent() (Directory, error)
	Remove() error
	Create() (File, error)
	Write(data []byte) (int, error)
	Copy(dst string) error
	Move(dst string) error
	Exists() bool
	Size() int64
	Read() ([]byte, error)
	IsDotFile() bool
}

type file struct {
	parent     string
	path       string
	chmod      *uint32
	uid        *int
	gid        *int
	overwrite  bool
	stats      *statsImpl
	buffersize *int
}

// NewFile validates option, resolves existing state, and returns a File
// handle. When the path does not exist and option.CreateIfNotExist is set,
// the underlying file is created with the requested permissions.
func NewFile(option Option) (File, error) {
	if !option.Validate() {
		return nil, fmt.Errorf("invalid option: %v", option)
	}

	st := NewStats(option.Path)
	exist := st.Exists()

	if exist && !st.IsFile() {
		return nil, os.ErrInvalid
	}

	f := &file{
		path:       option.Path,
		parent:     option.parent(),
		chmod:      option.Chmod,
		uid:        option.UID,
		gid:        option.GID,
		overwrite:  option.Overwrite,
		buffersize: option.BufferSize,
		stats:      st,
	}

	if !exist && option.CreateIfNotExist {
		return f.Create()
	}

	return f, nil
}

// Size returns the file size in bytes, refreshing the cached stats first.
func (f *file) Size() int64 {
	_ = f.stats.Refresh()
	return f.stats.meta.Size
}

// Path returns the absolute path.
func (f *file) Path() string { return f.path }

// Parent returns a Directory handle to the parent path.
func (f *file) Parent() (Directory, error) {
	return &directory{
		path:  f.parent,
		stats: NewStats(f.parent),
	}, nil
}

// Remove deletes the file.
func (f *file) Remove() error {
	if err := os.Remove(f.path); err != nil {
		return fmt.Errorf("failed to remove file %s: %w", f.path, err)
	}
	return nil
}

// Exists reports whether the file currently exists on disk.
func (f *file) Exists() bool {
	_ = f.stats.Refresh()
	return f.stats.Exists()
}

// IsDotFile reports whether the base name starts with a dot ("hidden" on POSIX).
func (f *file) IsDotFile() bool {
	base := filepath.Base(f.path)
	return strings.HasPrefix(base, ".") && len(base) > 1
}

// Create ensures the file exists. With Overwrite set, an existing file is
// truncated. Permissions and ownership are always (re)applied on success.
func (f *file) Create() (File, error) {
	if err := os.MkdirAll(filepath.Dir(f.path), os.FileMode(DefaultDirPerm)); err != nil {
		return nil, fmt.Errorf("failed to create parent directories: %w", err)
	}

	flags := os.O_CREATE | os.O_WRONLY
	if f.overwrite {
		flags |= os.O_TRUNC
	}

	fh, err := os.OpenFile(f.path, flags, os.FileMode(*f.chmod))
	if err != nil {
		return nil, fmt.Errorf("failed to create or open file: %w", err)
	}
	defer func() { _ = fh.Close() }()

	if err := fh.Chmod(os.FileMode(*f.chmod)); err != nil {
		return nil, fmt.Errorf("failed to set permissions: %w", err)
	}
	if err := fh.Chown(*f.uid, *f.gid); err != nil {
		return nil, fmt.Errorf("failed to set ownership: %w", err)
	}

	f.stats = NewStats(f.path)
	return f, nil
}

// Copy streams the file content to dst. The destination inherits the
// source's permission bits. Uses io.Copy — no hand-rolled pipeline.
func (f *file) Copy(dst string) error {
	src, err := os.Open(f.path)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() { _ = src.Close() }()

	srcInfo, err := src.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode().Perm())
	if err != nil {
		return fmt.Errorf("failed to open destination file: %w", err)
	}
	defer func() { _ = dstFile.Close() }()

	if _, err := io.Copy(dstFile, src); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}
	return nil
}

// Move renames the file; equivalent to os.Rename plus internal bookkeeping.
func (f *file) Move(dst string) error {
	if err := os.Rename(f.path, dst); err != nil {
		return fmt.Errorf("failed to move file from %s to %s: %w", f.path, dst, err)
	}
	f.path = dst
	f.stats = NewStats(dst)
	return nil
}

// Write replaces the file contents with data. Permissions are applied on open.
func (f *file) Write(data []byte) (int, error) {
	fh, err := os.OpenFile(f.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(*f.chmod))
	if err != nil {
		return 0, fmt.Errorf("failed to open file %s for writing: %w", f.path, err)
	}
	defer func() { _ = fh.Close() }()

	n, err := fh.Write(data)
	if err != nil {
		return n, fmt.Errorf("failed to write data to file %s: %w", f.path, err)
	}
	return n, nil
}

// Read returns the full file content. Uses io.ReadAll — no custom buffer loop.
func (f *file) Read() ([]byte, error) {
	fh, err := os.Open(f.path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = fh.Close() }()

	data, err := io.ReadAll(fh)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return data, nil
}
