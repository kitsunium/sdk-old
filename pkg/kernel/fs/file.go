package fs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

const (
	DefaultFilePerm uint32 = 0644 // Default permission for file creation
)

// File interface defines operations for file management
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

// file struct represents a file with its path and metadata
type file struct {
	parent        string
	path          string
	chmod         *uint32
	uid           *int
	gid           *int
	overwrite     bool
	preserveTimes bool
	stats         *stats
	buffersize    *int
}

// NewFile creates a new File object based on the given options
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

func (f *file) Size() int64 {
	f.stats.Refresh()
	return f.stats.meta.Size
}

// Path returns the file's path
func (f *file) Path() string {
	return f.path
}

// Parent retrieves the parent directory of the file
func (f *file) Parent() (Directory, error) {
	return &directory{
		path:  f.parent,
		stats: NewStats(f.parent),
	}, nil
}

// Remove deletes the file
func (f *file) Remove() error {
	return os.Remove(f.path)
}

// Exists checks if the file exists
func (f *file) Exists() bool {
	f.stats.Refresh()
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

	// Update permissions and ownership if specified
	if err := unix.Fchmod(fd, *f.chmod); err != nil {
		return nil, fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := unix.Fchown(fd, *f.uid, *f.gid); err != nil {
		return nil, fmt.Errorf("failed to set ownership: %w", err)
	}

	// Refresh file stats
	f.stats = NewStats(f.path)
	return f, nil
}

// Copy copies the file to a new destination using unix.Read and unix.Write with parallel pipelines.
func (f *file) Copy(dst string) error {
	srcFD, err := unix.Open(f.path, unix.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer unix.Close(srcFD)

	// Get source file stats for mode and size
	srcStat := &unix.Stat_t{}
	if err := unix.Fstat(srcFD, srcStat); err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	// Open destination file
	dstFD, err := unix.Open(dst, unix.O_WRONLY|unix.O_CREAT|unix.O_TRUNC, uint32(srcStat.Mode))
	if err != nil {
		return fmt.Errorf("failed to open destination file: %w", err)
	}
	defer unix.Close(dstFD)

	// Channels for communication between reader and writer
	type chunk struct {
		data []byte
		n    int
		err  error
		done bool
	}
	readChan := make(chan chunk, 2) // Buffered channel for pipelining

	// Goroutine for reading
	go func() {
		defer close(readChan)
		buffer := make([]byte, *f.buffersize)
		for {
			bytesRead, readErr := unix.Read(srcFD, buffer)
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
	}()

	// Writing in the main goroutine
	for chunk := range readChan {
		if chunk.err != nil {
			if chunk.err == io.EOF {
				break // End of file
			}
			return fmt.Errorf("failed to read from source file: %w", chunk.err)
		}

		bytesWritten := 0
		for bytesWritten < chunk.n {
			n, writeErr := unix.Write(dstFD, chunk.data[bytesWritten:chunk.n])
			if writeErr != nil {
				return fmt.Errorf("failed to write to destination file: %w", writeErr)
			}
			bytesWritten += n
		}

		if chunk.done {
			break
		}
	}

	return nil
}

// Move moves the file to a new destination
func (f *file) Move(dst string) error {
	if err := os.Rename(f.path, dst); err != nil {
		return err
	}

	f.path = dst
	f.stats = NewStats(dst)
	return nil
}

// Write writes data to the file
func (f *file) Write(data []byte) (int, error) {
	fd, err := unix.Open(f.path, unix.O_CREAT|unix.O_WRONLY|unix.O_TRUNC, *f.chmod)
	if err != nil {
		return 0, err
	}
	defer unix.Close(fd)

	n, err := unix.Write(fd, data)
	if err != nil {
		return n, err
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
	// Open the file using a low-level Unix system call
	fd, err := unix.Open(f.path, unix.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer unix.Close(fd)

	// Use a dynamically growing buffer for storing file content
	var result []byte
	var buffer []byte = make([]byte, *f.buffersize)

	for {
		// Read a chunk of data
		n, err := unix.Read(fd, buffer)
		if n > 0 {
			result = append(result, buffer[:n]...)
		}

		// Handle errors
		if err != nil {
			if err == io.EOF {
				break // End of file reached
			}
			return nil, fmt.Errorf("failed to read file: %w", err)
		}

		// Stop if no more data is available
		if n == 0 {
			break
		}
	}

	return result, nil
}
