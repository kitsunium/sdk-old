package fs_test

import (
	"testing"

	"github.com/kitsunium/sdk/pkg/kernel/fs"
	"github.com/stretchr/testify/assert"
)

func TestPath(t *testing.T) {
	t.Run("Parent", func(t *testing.T) {
		t.Run("nominal path", func(t *testing.T) {
			path := fs.Path("/home/user/documents/file.txt")
			expected := "/home/user/documents"
			result := path.Parent()
			assert.Equal(t, expected, result, "Parent should return correct parent directory")
		})

		t.Run("root path", func(t *testing.T) {
			path := fs.Path("/")
			expected := "/"
			result := path.Parent()
			assert.Equal(t, expected, result, "Parent of root should return root itself")
		})

		t.Run("relative path", func(t *testing.T) {
			path := fs.Path("dir/file.txt")
			expected := "dir"
			result := path.Parent()
			assert.Equal(t, expected, result, "Parent should work with relative paths")
		})

		t.Run("single level path", func(t *testing.T) {
			path := fs.Path("file.txt")
			expected := "."
			result := path.Parent()
			assert.Equal(t, expected, result, "Parent of single level file should return '.'")
		})

		t.Run("path with trailing slash", func(t *testing.T) {
			path := fs.Path("/home/user/documents/")
			expected := "/home/user/documents"
			result := path.Parent()
			assert.Equal(t, expected, result, "Parent should handle paths with trailing slash")
		})

		t.Run("empty path", func(t *testing.T) {
			path := fs.Path("")
			expected := "."
			result := path.Parent()
			assert.Equal(t, expected, result, "Parent of empty path should return '.'")
		})
	})

	t.Run("String", func(t *testing.T) {
		t.Run("nominal path", func(t *testing.T) {
			pathStr := "/home/user/documents/file.txt"
			path := fs.Path(pathStr)
			result := path.String()
			assert.Equal(t, pathStr, result, "String should return exact string representation")
		})

		t.Run("empty path", func(t *testing.T) {
			pathStr := ""
			path := fs.Path(pathStr)
			result := path.String()
			assert.Equal(t, pathStr, result, "String should handle empty path")
		})

		t.Run("path with spaces", func(t *testing.T) {
			pathStr := "/path with spaces/file.txt"
			path := fs.Path(pathStr)
			result := path.String()
			assert.Equal(t, pathStr, result, "String should preserve spaces")
		})

		t.Run("complex path", func(t *testing.T) {
			pathStr := "/var/log/../tmp/./file"
			path := fs.Path(pathStr)
			result := path.String()
			assert.Equal(t, pathStr, result, "String should return original path without cleaning")
		})
	})

	t.Run("Clean", func(t *testing.T) {
		t.Run("path with dots", func(t *testing.T) {
			path := fs.Path("/var/log/../tmp/./file")
			expected := "/var/tmp/file"
			result := path.Clean()
			assert.Equal(t, expected, result, "Clean should resolve . and .. elements")
		})

		t.Run("path with spaces", func(t *testing.T) {
			path := fs.Path("  /home/user/file.txt  ")
			expected := "/home/user/file.txt"
			result := path.Clean()
			assert.Equal(t, expected, result, "Clean should trim whitespace")
		})

		t.Run("path with multiple slashes", func(t *testing.T) {
			path := fs.Path("/home//user///file.txt")
			expected := "/home/user/file.txt"
			result := path.Clean()
			assert.Equal(t, expected, result, "Clean should remove extra slashes")
		})
	})
}

func TestOption(t *testing.T) {
	t.Run("parent", func(t *testing.T) {
		t.Run("nominal path", func(t *testing.T) {
			option := fs.Option{
				Path: "/home/user/documents/file.txt",
			}
			// Call Validate first to ensure path is cleaned
			option.Validate()
			expected := "/home/user/documents"

			// We need to use reflection or test this indirectly since parent() is private
			// Let's test this through the Parent() method of created objects
			file, err := fs.NewFile(option)
			if err == nil {
				parent, err := file.Parent()
				assert.NoError(t, err)
				assert.Equal(t, expected, parent.Path())
			}
		})

		t.Run("root path", func(t *testing.T) {
			option := fs.Option{
				Path: "/file.txt",
			}
			option.Validate()
			expected := "/"

			file, err := fs.NewFile(option)
			if err == nil {
				parent, err := file.Parent()
				assert.NoError(t, err)
				assert.Equal(t, expected, parent.Path())
			}
		})

		t.Run("relative path", func(t *testing.T) {
			option := fs.Option{
				Path: "dir/file.txt",
			}
			option.Validate()
			expected := "dir"

			file, err := fs.NewFile(option)
			if err == nil {
				parent, err := file.Parent()
				assert.NoError(t, err)
				assert.Equal(t, expected, parent.Path())
			}
		})
	})

	t.Run("Validate", func(t *testing.T) {
		t.Run("valid path", func(t *testing.T) {
			option := fs.Option{
				Path: "/home/user/file.txt",
			}
			result := option.Validate()
			assert.True(t, result, "Validate should return true for valid path")
			assert.Equal(t, "/home/user/file.txt", option.Path, "Path should be cleaned")
			assert.NotNil(t, option.Chmod, "Chmod should be set to default")
			assert.NotNil(t, option.UID, "UID should be set to default")
			assert.NotNil(t, option.GID, "GID should be set to default")
			assert.NotNil(t, option.BufferSize, "BufferSize should be set to default")
		})

		t.Run("current directory path", func(t *testing.T) {
			option := fs.Option{
				Path: ".",
			}
			result := option.Validate()
			assert.False(t, result, "Validate should return false for current directory path")
		})

		t.Run("empty path", func(t *testing.T) {
			option := fs.Option{
				Path: "",
			}
			result := option.Validate()
			assert.False(t, result, "Validate should return false for empty path")
		})

		t.Run("path with only spaces", func(t *testing.T) {
			option := fs.Option{
				Path: "   ",
			}
			result := option.Validate()
			assert.False(t, result, "Validate should return false for whitespace-only path")
		})

		t.Run("path that cleans to current directory", func(t *testing.T) {
			option := fs.Option{
				Path: "./././.",
			}
			result := option.Validate()
			assert.False(t, result, "Validate should return false for path that cleans to '.'")
		})

		t.Run("with pre-set values", func(t *testing.T) {
			chmod := uint32(0755)
			uid := 100
			gid := 200
			bufferSize := 64 * 1024

			option := fs.Option{
				Path:       "/home/user/file.txt",
				Chmod:      &chmod,
				UID:        &uid,
				GID:        &gid,
				BufferSize: &bufferSize,
			}

			result := option.Validate()
			assert.True(t, result, "Validate should return true for valid path")
			assert.Equal(t, chmod, *option.Chmod, "Chmod should preserve pre-set value")
			assert.Equal(t, uid, *option.UID, "UID should preserve pre-set value")
			assert.Equal(t, gid, *option.GID, "GID should preserve pre-set value")
			assert.Equal(t, bufferSize, *option.BufferSize, "BufferSize should preserve pre-set value")
		})

		t.Run("complex path cleaning", func(t *testing.T) {
			option := fs.Option{
				Path: "  /var/log/../tmp/./file//  ",
			}
			result := option.Validate()
			assert.True(t, result, "Validate should return true for complex but valid path")
			assert.Equal(t, "/var/tmp/file", option.Path, "Path should be properly cleaned")
		})
	})
}
