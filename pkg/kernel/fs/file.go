// Package fs provides file system operations and utilities for the Kitsunium kernel.
//
// This package offers low-level file system operations using direct Unix system calls
// for optimal performance in kernel-level applications. It provides abstractions for
// files, directories, archives, and system statistics while maintaining high performance
// through minimal allocations and direct system call usage.
//
// Key Features:
//   - High-performance file I/O using Unix system calls
//   - Zero-copy operations where possible
//   - Parallel I/O pipelines for large file operations
//   - Comprehensive file system statistics
//   - Archive creation and extraction support
//   - Cross-platform compatibility through golang.org/x/sys
//
// Performance Characteristics:
//   - Uses direct Unix system calls (unix.Open, unix.Read, unix.Write)
//   - Implements parallel copy operations with pipelines
//   - Configurable buffer sizes for optimal throughput
//   - Minimal memory allocations in hot paths
package fs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

const (
	DefaultFilePerm uint32 = 0644 // Default permission for file creation.
)

// File interface defines operations for file management.
// Implementations provide high-performance file operations using
// direct system calls where possible.
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

// file struct represents a file with its path and metadata.
// It caches file statistics and provides configurable options for
// permissions, ownership, and I/O buffer sizes.
type file struct {
	parent     string
	path       string
	chmod      *uint32
	uid        *int
	gid        *int
	overwrite  bool
	stats      *stats
	buffersize *int
}

// NewFile creates a new File object based on the given options.
// It validates the path, checks if it exists, and optionally creates
// the file if CreateIfNotExist is set in options.
func NewFile(option Option) (File, error) {
	if !option.Validate() {
		return nil, fmt.Errorf("invalid option: %v", option)
	}

	stats := NewStats(option.Path)
	exist := stats.Exists()

	if exist && !stats.IsFile() {
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
		stats:      stats,
	}

	if !exist && option.CreateIfNotExist {
		return f.Create()
	}

	return f, nil
}

// Size returns the size of the file in bytes.
// It refreshes the file statistics before returning the size.
func (f *file) Size() int64 {
	_ = f.stats.Refresh() // Ignore refresh error for size calculation
	return f.stats.meta.Size
}

// Path returns the file's path.
func (f *file) Path() string {
	return f.path
}

// Parent retrieves the parent directory of the file.
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

// Exists checks if the file exists.
func (f *file) Exists() bool {
	_ = f.stats.Refresh() // Ignore refresh error for existence check
	return f.stats.Exists()
}

// IsDotFile checks if the file is a dotfile (hidden file).
func (f *file) IsDotFile() bool {
	base := filepath.Base(f.path)
	return strings.HasPrefix(base, ".") && len(base) > 1
}

// Create ensures the file exists, creating it if necessary, and optionally overwrites it.
// If Overwrite is true, the file is truncated, and permissions and ownership are updated.
//
// Returns:
// - File: The created or updated file object.
// - error: Error if the operation fails.
func (f *file) Create() (File, error) {
	if err := os.MkdirAll(filepath.Dir(f.path), os.FileMode(DefaultDirPerm)); err != nil {
		return nil, fmt.Errorf("failed to create parent directories: %w", err)
	}

	flags := unix.O_CREAT | unix.O_WRONLY
	if f.overwrite {
		flags |= unix.O_TRUNC
	}

	fd, err := unix.Open(f.path, flags, *f.chmod)
	if err != nil {
		return nil, fmt.Errorf("failed to create or open file: %w", err)
	}
	defer unix.Close(fd)

	// Update permissions and ownership if specified.
	if err := unix.Fchmod(fd, *f.chmod); err != nil {
		return nil, fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := unix.Fchown(fd, *f.uid, *f.gid); err != nil {
		return nil, fmt.Errorf("failed to set ownership: %w", err)
	}

	// Refresh file stats.
	f.stats = NewStats(f.path)
	return f, nil
}

// chunk represents a data chunk for file copying.
type chunk struct {
	data []byte
	n    int
	err  error
	done bool
}

// Copy copies the file to a new destination using unix.Read and unix.Write with parallel pipelines..
func (f *file) Copy(dst string) error {
	srcFD, dstFD, err := f.openFiles(dst)
	if err != nil {
		return err
	}
	defer unix.Close(srcFD)
	defer unix.Close(dstFD)

	readChan := make(chan chunk, 2) // Buffered channel for pipelining
	go f.readFileAsync(srcFD, readChan)

	return f.writeFromChannel(dstFD, readChan)
}

// openFiles opens source and destination files for copying.
func (f *file) openFiles(dst string) (srcFD, dstFD int, err error) {
	srcFD, err = unix.Open(f.path, unix.O_RDONLY, 0)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open source file: %w", err)
	}

	// Get source file stats for mode.
	srcStat := &unix.Stat_t{}
	if err := unix.Fstat(srcFD, srcStat); err != nil {
		unix.Close(srcFD)
		return 0, 0, fmt.Errorf("failed to stat source file: %w", err)
	}

	// Open destination file with permission bits only (mask out file type bits).
	dstFD, err = unix.Open(dst, unix.O_WRONLY|unix.O_CREAT|unix.O_TRUNC, uint32(srcStat.Mode&0777))
	if err != nil {
		unix.Close(srcFD)
		return 0, 0, fmt.Errorf("failed to open destination file: %w", err)
	}

	return srcFD, dstFD, nil
}

// readFileAsync reads file data asynchronously and sends chunks to the channel.
func (f *file) readFileAsync(fd int, readChan chan<- chunk) {
	defer close(readChan)
	for {
		buffer := make([]byte, *f.buffersize)
		bytesRead, readErr := unix.Read(fd, buffer)
		readChan <- chunk{
			data: buffer[:bytesRead],
			n:    bytesRead,
			err:  readErr,
			done: bytesRead == 0 || readErr != nil,
		}
		if bytesRead == 0 || readErr != nil {
			break
		}
	}
}

// writeFromChannel writes data from the channel to the destination file.
func (f *file) writeFromChannel(dstFD int, readChan <-chan chunk) error {
	for chunk := range readChan {
		if chunk.err != nil && !errors.Is(chunk.err, io.EOF) {
			return fmt.Errorf("failed to read from source file: %w", chunk.err)
		}

		if err := f.writeChunk(dstFD, chunk.data[:chunk.n]); err != nil {
			return err
		}

		if chunk.done {
			break
		}
	}
	return nil
}

// writeChunk writes a complete chunk of data to the file.
func (f *file) writeChunk(fd int, data []byte) error {
	bytesWritten := 0
	for bytesWritten < len(data) {
		n, err := unix.Write(fd, data[bytesWritten:])
		if err != nil {
			return fmt.Errorf("failed to write to destination file: %w", err)
		}
		bytesWritten += n
	}
	return nil
}

// Move moves the file to a new destination.
func (f *file) Move(dst string) error {
	if err := os.Rename(f.path, dst); err != nil {
		return fmt.Errorf("failed to move file from %s to %s: %w", f.path, dst, err)
	}

	f.path = dst
	f.stats = NewStats(dst)
	return nil
}

// Write writes data to the file.
func (f *file) Write(data []byte) (int, error) {
	fd, err := unix.Open(f.path, unix.O_CREAT|unix.O_WRONLY|unix.O_TRUNC, *f.chmod)
	if err != nil {
		return 0, fmt.Errorf("failed to open file %s for writing: %w", f.path, err)
	}
	defer unix.Close(fd)

	n, err := unix.Write(fd, data)
	if err != nil {
		return n, fmt.Errorf("failed to write data to file %s: %w", f.path, err)
	}

	return n, nil
}

// Read reads the file content in chunks and returns the complete data as a byte slice.
// It uses low-level Unix system calls to minimize overhead.
//
// Returns:
// - []byte: The file content.
// - error: Error if the operation fails.
func (f *file) Read() ([]byte, error) {
	// Open the file using a low-level Unix system call.
	fd, err := unix.Open(f.path, unix.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer unix.Close(fd)

	// Use a dynamically growing buffer for storing file content.
	var result []byte
	buffer := make([]byte, *f.buffersize)

	for {
		// Read a chunk of data.
		n, err := unix.Read(fd, buffer)
		if n > 0 {
			result = append(result, buffer[:n]...)
		}

		// Handle errors.
		if err != nil {
			if errors.Is(err, io.EOF) {
				break // End of file reached
			}
			return nil, fmt.Errorf("failed to read file: %w", err)
		}

		// Stop if no more data is available.
		if n == 0 {
			break
		}
	}

	return result, nil
}
