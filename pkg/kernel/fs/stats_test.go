package fs_test

import (
	"os"
	"os/user"
	"strconv"
	"testing"

	"github.com/kitsunium/sdk/pkg/kernel/fs"
	"github.com/stretchr/testify/assert"
)

func TestStats(t *testing.T) {
	tempFile, err := os.CreateTemp("", "testfile")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	tempDir, err := os.MkdirTemp("", "testdir")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	u, err := user.Current()
	assert.NoError(t, err)
	uid, err := strconv.Atoi(u.Uid)
	assert.NoError(t, err)
	gid, err := strconv.Atoi(u.Gid)
	assert.NoError(t, err)

	t.Run("NewStats", func(t *testing.T) {
		t.Run("nominal", func(t *testing.T) {
			stats := fs.NewStats(tempFile.Name())
			assert.NotNil(t, stats, "NewStats should not return nil for a valid file")
		})

		t.Run("exception", func(t *testing.T) {
			stats := fs.NewStats("/invalid/path")
			assert.NotNil(t, stats, "NewStats should return nil for an invalid path")
		})
	})

	t.Run("HasPermissions", func(t *testing.T) {
		stats := fs.NewStats(tempFile.Name())
		assert.NotNil(t, stats)

		t.Run("nominal", func(t *testing.T) {
			err := stats.Chmod(0644)
			assert.NoError(t, err)

			stats.Refresh()
			assert.True(t, stats.HasPermissions(os.FileMode(0444)), "File should have read permissions")
			assert.False(t, stats.HasPermissions(os.FileMode(0111)), "File should not have execute permissions")
		})

		t.Run("exception", func(t *testing.T) {
			assert.False(t, stats.HasPermissions(os.FileMode(07777)), "Invalid permissions should return false")
		})
	})

	t.Run("IsReadable", func(t *testing.T) {
		stats := fs.NewStats(tempFile.Name())
		assert.NotNil(t, stats)

		t.Run("nominal", func(t *testing.T) {
			err := stats.Chmod(0444)
			assert.NoError(t, err)

			stats.Refresh()
			assert.True(t, stats.IsReadable(uint32(uid), 0), "File should be readable")
			assert.True(t, stats.IsReadable(0, uint32(gid)), "File should be readable")
			assert.True(t, stats.IsReadable(0, 0), "File should be readable")
		})

		t.Run("exception", func(t *testing.T) {
			err := stats.Chmod(0000)
			assert.NoError(t, err)

			stats.Refresh()
			assert.False(t, stats.IsReadable(uint32(uid), 0), "File with no read permissions should not be readable")
			assert.False(t, stats.IsReadable(0, uint32(gid)), "File with no read permissions should not be readable")
			assert.False(t, stats.IsReadable(0, 0), "File with no read permissions should not be readable")
		})
	})

	t.Run("IsWritable", func(t *testing.T) {
		stats := fs.NewStats(tempFile.Name())
		assert.NotNil(t, stats)

		t.Run("nominal", func(t *testing.T) {
			err := stats.Chmod(0222)
			assert.NoError(t, err)

			stats.Refresh()
			assert.True(t, stats.IsWritable(uint32(uid), 0), "File should be writable")
			assert.True(t, stats.IsWritable(0, uint32(gid)), "File should be writable")
			assert.True(t, stats.IsWritable(0, 0), "File should be writable")
		})

		t.Run("exception", func(t *testing.T) {
			err := stats.Chmod(0000)
			assert.NoError(t, err)

			stats.Refresh()
			assert.False(t, stats.IsWritable(uint32(uid), 0), "File without write permissions should not be writable")
			assert.False(t, stats.IsWritable(0, uint32(gid)), "File without write permissions should not be writable")
			assert.False(t, stats.IsWritable(0, 0), "File without write permissions should not be writable")
		})
	})

	t.Run("IsExecutable", func(t *testing.T) {
		stats := fs.NewStats(tempFile.Name())
		assert.NotNil(t, stats)

		t.Run("nominal", func(t *testing.T) {
			err := stats.Chmod(0111)
			assert.NoError(t, err)

			stats.Refresh()
			assert.True(t, stats.IsExecutable(uint32(uid), 0), "File should be writable")
			assert.True(t, stats.IsExecutable(0, uint32(gid)), "File should be writable")
			assert.True(t, stats.IsExecutable(0, 0), "File should be writable")
		})

		t.Run("exception", func(t *testing.T) {
			err := stats.Chmod(0000)
			assert.NoError(t, err)

			stats.Refresh()
			assert.False(t, stats.IsExecutable(uint32(uid), 0), "File without execute permissions should not be executable")
			assert.False(t, stats.IsExecutable(0, uint32(gid)), "File without execute permissions should not be executable")
			assert.False(t, stats.IsExecutable(0, 0), "File without execute permissions should not be executable")
		})
	})

	t.Run("IsFile", func(t *testing.T) {
		t.Run("nominal", func(t *testing.T) {
			stats := fs.NewStats(tempFile.Name())
			assert.NotNil(t, stats, "Stats should not be nil for a valid file")
			assert.True(t, stats.IsFile(), "Stats should detect a regular file")
		})

		t.Run("exception", func(t *testing.T) {
			stats := fs.NewStats(tempDir)
			assert.NotNil(t, stats, "Stats should not be nil for a valid directory")
			assert.False(t, stats.IsFile(), "Stats should not detect a directory as a file")
		})
	})

	t.Run("IsDirectory", func(t *testing.T) {
		t.Run("nominal", func(t *testing.T) {
			stats := fs.NewStats(tempDir)
			assert.NotNil(t, stats, "Stats should not be nil for a valid directory")
			assert.True(t, stats.IsDirectory(), "Stats should detect a directory")
		})

		t.Run("exception", func(t *testing.T) {
			stats := fs.NewStats(tempFile.Name())
			assert.NotNil(t, stats, "Stats should not be nil for a valid file")
			assert.False(t, stats.IsDirectory(), "Stats should not detect a regular file as a directory")
		})
	})

	t.Run("IsExecutable", func(t *testing.T) {
		stats := fs.NewStats(tempFile.Name())
		assert.NotNil(t, stats)

		t.Run("nominal", func(t *testing.T) {
			err := stats.Chmod(0755)
			assert.NoError(t, err)

			stats.Refresh()
			assert.True(t, stats.IsExecutable(uint32(uid), uint32(gid)), "File should be executable")
		})

		t.Run("exception", func(t *testing.T) {
			err := stats.Chmod(0644)
			assert.NoError(t, err)

			stats.Refresh()
			assert.False(t, stats.IsExecutable(uint32(uid), uint32(gid)), "File without execute permissions should not be executable")
		})
	})

	t.Run("IsExecutable", func(t *testing.T) {
		stats := fs.NewStats(tempFile.Name())
		assert.NotNil(t, stats)

		t.Run("nominal", func(t *testing.T) {
			err := stats.Chmod(0755)
			assert.NoError(t, err)

			stats.Refresh()
			assert.True(t, stats.IsExecutable(uint32(uid), uint32(gid)), "File should be executable")
		})

		t.Run("exception", func(t *testing.T) {
			err := stats.Chmod(0644)
			assert.NoError(t, err)

			stats.Refresh()
			assert.False(t, stats.IsExecutable(uint32(uid), uint32(gid)), "File without execute permissions should not be executable")
		})
	})

	t.Run("Owner", func(t *testing.T) {
		stats := fs.NewStats(tempFile.Name())
		assert.NotNil(t, stats)

		t.Run("nominal", func(t *testing.T) {
			owner := stats.Owner()
			assert.NotEmpty(t, owner.Name(), "Owner should not be empty")
			assert.NotZero(t, owner.ID, "Owner ID should not be zero")
			assert.True(t, owner.Permissions.Read, "Owner should have read permissions")
			assert.True(t, owner.Permissions.Write, "Owner should have write permissions")
			assert.False(t, owner.Permissions.Exec, "Owner should not have execute permissions by default")

			// second call to ensure the owner is cached
			assert.NotEmpty(t, owner.Name(), "Owner should not be empty")
		})
	})

	t.Run("Group", func(t *testing.T) {
		stats := fs.NewStats(tempFile.Name())
		assert.NotNil(t, stats)

		t.Run("nominal", func(t *testing.T) {
			group := stats.Group()
			assert.NotEmpty(t, group.Name(), "Group should not be empty")
			assert.NotZero(t, group.ID, "Group ID should not be zero")
			assert.True(t, group.Permissions.Read, "Group should have read permissions")
			assert.False(t, group.Permissions.Write, "Group should not have write permissions by default")
			assert.False(t, group.Permissions.Exec, "Group should not have execute permissions by default")

			// second call to ensure the group is cached
			assert.NotEmpty(t, group.Name(), "Group should not be empty")
		})
	})

	t.Run("Other", func(t *testing.T) {
		stats := fs.NewStats(tempFile.Name())
		assert.NotNil(t, stats)

		t.Run("nominal", func(t *testing.T) {
			other := stats.Other()
			assert.True(t, other.Permissions.Read, "Other should have read permissions by default")
			assert.False(t, other.Permissions.Write, "Other should not have write permissions by default")
			assert.False(t, other.Permissions.Exec, "Other should not have execute permissions by default")
		})
	})

	t.Run("Chmod", func(t *testing.T) {
		stats := fs.NewStats(tempFile.Name())
		assert.NotNil(t, stats)

		t.Run("nominal", func(t *testing.T) {
			err := stats.Chmod(0600)
			assert.NoError(t, err, "Chmod should not return an error")
			stats.Refresh()
			assert.True(t, stats.HasPermissions(os.FileMode(0600)), "File should have the new permissions")
		})
	})

	t.Run("Chown", func(t *testing.T) {
		stats := fs.NewStats(tempFile.Name())
		assert.NotNil(t, stats)

		currentUID := stats.Owner().ID
		currentGID := stats.Group().ID

		t.Run("nominal", func(t *testing.T) {
			err := stats.Chown(int(currentUID), int(currentGID))
			assert.NoError(t, err, "Chown should not return an error")
		})

		t.Run("exception", func(t *testing.T) {
			err := stats.Chown(-1, -1)
			assert.Error(t, err, "Chown should return an error for invalid UID/GID")
		})

		t.Run("exception", func(t *testing.T) {
			err := stats.Chown(8, 8)
			assert.Error(t, err, "Chown should return an error for invalid UID/GID")
		})
	})

	t.Run("IsFile", func(t *testing.T) {
		t.Run("Nominal", func(t *testing.T) {
			tempFile, err := os.CreateTemp("", "testfile")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tempFile.Name())

			s := fs.NewStats(tempFile.Name())
			if !s.IsFile() {
				t.Errorf("Expected IsFile to return true for a regular file, got false")
			}
		})

		t.Run("EdgeCase", func(t *testing.T) {
			sDir := fs.NewStats(".")
			if sDir.IsFile() {
				t.Errorf("Expected IsFile to return false for a directory, got true")
			}
		})
	})

	t.Run("IsDirectory", func(t *testing.T) {
		t.Run("Nominal", func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "testdir")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			s := fs.NewStats(tempDir)
			if !s.IsDirectory() {
				t.Errorf("Expected IsDirectory to return true for a directory, got false")
			}
		})

		t.Run("EdgeCase", func(t *testing.T) {
			sFile := fs.NewStats("/nonexistent")
			if sFile.IsDirectory() {
				t.Errorf("Expected IsDirectory to return false for a non-existent path, got true")
			}
		})
	})

	t.Run("Exists", func(t *testing.T) {
		t.Run("Nominal", func(t *testing.T) {
			tempFile, err := os.CreateTemp("", "testfile")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tempFile.Name())

			s := fs.NewStats(tempFile.Name())
			if !s.Exists() {
				t.Errorf("Expected Exists to return true for existing file, got false")
			}
		})

		t.Run("EdgeCase", func(t *testing.T) {
			sNonExistent := fs.NewStats("/non/existent/path")
			if sNonExistent.Exists() {
				t.Errorf("Expected Exists to return false for non-existent path, got true")
			}
		})
	})

	t.Run("Owner", func(t *testing.T) {
		t.Run("Nominal", func(t *testing.T) {
			tempFile, err := os.CreateTemp("", "testfile")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tempFile.Name())

			s := fs.NewStats(tempFile.Name())
			owner := s.Owner()
			if owner.ID == 0 {
				t.Errorf("Expected valid owner ID, got 0")
			}

			if owner.Name() == "" {
				t.Errorf("Expected valid owner name, got empty string")
			}
		})
	})

	t.Run("Group", func(t *testing.T) {
		t.Run("Nominal", func(t *testing.T) {
			tempFile, err := os.CreateTemp("", "testfile")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tempFile.Name())

			s := fs.NewStats(tempFile.Name())
			group := s.Group()
			if group.ID == 0 {
				t.Errorf("Expected valid group ID, got 0")
			}

			if group.Name() == "" {
				t.Errorf("Expected valid group name, got empty string")
			}
		})
	})

	t.Run("Other", func(t *testing.T) {
		t.Run("Nominal", func(t *testing.T) {
			tempFile, err := os.CreateTemp("", "testfile")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tempFile.Name())

			s := fs.NewStats(tempFile.Name())
			other := s.Other()
			if other.Permissions.Read || other.Permissions.Write || other.Permissions.Exec {
				t.Errorf("Expected other permissions to be false, got true for one or more flags")
			}
		})
	})
}

func BenchmarkStatsRefresh(b *testing.B) {
	tempFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	stats := fs.NewStats(tempFile.Name())
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = stats.Refresh()
	}
}
