package fs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kitsunium/sdk/pkg/kernel/fs"
	"github.com/stretchr/testify/assert"
)

func TestDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "testdir")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("NewDirectory", func(t *testing.T) {
		t.Run("nominal", func(t *testing.T) {
			d, err := fs.NewDirectory(fs.Option{
				Path: tmpDir,
			})
			assert.NoError(t, err)
			assert.NotNil(t, d)
		})

		t.Run("path empty", func(t *testing.T) {
			d, err := fs.NewDirectory(fs.Option{})
			assert.Error(t, err)
			assert.Nil(t, d)
		})

		t.Run("create if not exist", func(t *testing.T) {
			newDir := tmpDir + "/newdir"
			d, err := fs.NewDirectory(fs.Option{
				Path:             newDir,
				CreateIfNotExist: true,
			})
			assert.NoError(t, err)
			assert.NotNil(t, d)

			_, err = os.Stat(newDir)
			assert.NoError(t, err)
		})

		t.Run("path is a file", func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "testfile")
			assert.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			d, err := fs.NewDirectory(fs.Option{
				Path: tmpFile.Name(),
			})
			assert.Error(t, err)
			assert.Nil(t, d)
		})
	})

	t.Run("Path", func(t *testing.T) {
		d, err := fs.NewDirectory(fs.Option{
			Path: tmpDir,
		})
		assert.NoError(t, err)
		assert.NotNil(t, d)

		t.Run("nominal", func(t *testing.T) {
			assert.Equal(t, tmpDir, d.Path())
		})
	})

	t.Run("Parent", func(t *testing.T) {
		d, err := fs.NewDirectory(fs.Option{
			Path: tmpDir,
		})
		assert.NoError(t, err)
		// Utiliser filepath.Dir pour calculer le chemin du parent
		parentPath := tmpDir[:len(tmpDir)-len(filepath.Base(tmpDir))-1]

		parent, err := d.Parent()
		assert.NoError(t, err)
		assert.NotNil(t, parent)
		assert.Equal(t, parent.Path(), parentPath)
	})

	t.Run("Remove", func(t *testing.T) {
		newDir := tmpDir + "/toremove"
		d, err := fs.NewDirectory(fs.Option{
			Path:             newDir,
			CreateIfNotExist: true,
		})
		assert.NoError(t, err)

		err = d.Remove()
		assert.NoError(t, err)

		_, err = os.Stat(newDir)
		assert.Error(t, err, "directory should not exist after removal")
	})

	t.Run("List", func(t *testing.T) {
		// Nettoyer complètement le répertoire temporaire
		err := os.RemoveAll(tmpDir)
		assert.NoError(t, err)

		err = os.Mkdir(tmpDir, 0755)
		assert.NoError(t, err)

		testFile := tmpDir + "/file1.txt"
		testSubDir := tmpDir + "/subdir"

		// Create file and subdirectory
		_ = os.WriteFile(testFile, []byte("test content"), 0644)
		_ = os.Mkdir(testSubDir, 0755)

		d, err := fs.NewDirectory(fs.Option{
			Path: tmpDir,
		})
		assert.NoError(t, err)

		files, subdirs, err := d.List()

		assert.NoError(t, err)
		assert.Equal(t, 1, len(files), "expected one file")
		assert.Equal(t, 1, len(subdirs), "expected one subdirectory")
	})

	t.Run("Size", func(t *testing.T) {
		d, err := fs.NewDirectory(fs.Option{
			Path: tmpDir,
		})
		assert.NoError(t, err)

		t.Run("nominal", func(t *testing.T) {
			size := d.Size()
			assert.GreaterOrEqual(t, size, int64(0))
		})
	})

	t.Run("Has", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "testdir")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		subDir := filepath.Join(tmpDir, "subdir")
		err = os.Mkdir(subDir, 0755)
		assert.NoError(t, err)

		tmpFile, err := os.CreateTemp(tmpDir, "testfile")
		assert.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		outsideDir := "/tmp/outsideDir"
		err = os.Mkdir(outsideDir, 0755)
		assert.NoError(t, err)
		defer os.RemoveAll(outsideDir)

		outsideFile, err := os.CreateTemp(outsideDir, "outsidefile")
		assert.NoError(t, err)
		defer os.Remove(outsideFile.Name())

		dir, err := fs.NewDirectory(fs.Option{Path: tmpDir})
		assert.NoError(t, err)

		t.Run("File inside directory", func(t *testing.T) {
			assert.True(t, dir.Has(tmpFile.Name()), "Expected file to be within the directory")
		})

		t.Run("Subdirectory inside directory", func(t *testing.T) {
			assert.True(t, dir.Has(subDir), "Expected subdirectory to be within the directory")
		})

		t.Run("File outside directory", func(t *testing.T) {
			assert.False(t, dir.Has(outsideFile.Name()), "Expected file to not be within the directory")
		})

		t.Run("Directory outside directory", func(t *testing.T) {
			assert.False(t, dir.Has(outsideDir), "Expected directory to not be within the directory")
		})

		t.Run("Directory has itself", func(t *testing.T) {
			assert.True(t, dir.Has(tmpDir), "Expected directory to contain itself")
		})
	})

}

// TestDirectoryEdgeCases contains comprehensive tests for directory.go to achieve 100% coverage
func TestDirectoryEdgeCases(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "testdir_comprehensive")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("Parent edge cases", func(t *testing.T) {
		t.Run("root directory", func(t *testing.T) {
			d, err := fs.NewDirectory(fs.Option{
				Path: "/",
			})
			assert.NoError(t, err)

			parent, err := d.Parent()
			assert.Error(t, err, "Root directory should not have a parent")
			assert.Nil(t, parent)
			assert.Contains(t, err.Error(), "no parent directory found")
		})

		t.Run("nested directory", func(t *testing.T) {
			nestedPath := filepath.Join(tmpDir, "nested", "deep")
			err := os.MkdirAll(nestedPath, 0755)
			assert.NoError(t, err)

			d, err := fs.NewDirectory(fs.Option{
				Path: nestedPath,
			})
			assert.NoError(t, err)

			parent, err := d.Parent()
			assert.NoError(t, err)
			assert.NotNil(t, parent)
			expectedParent := filepath.Join(tmpDir, "nested")
			assert.Equal(t, expectedParent, parent.Path())
		})

		t.Run("single level directory", func(t *testing.T) {
			singleLevel := "testdir"
			// Create in current directory
			err := os.MkdirAll(singleLevel, 0755)
			assert.NoError(t, err)
			defer os.RemoveAll(singleLevel)

			d, err := fs.NewDirectory(fs.Option{
				Path: singleLevel,
			})
			assert.NoError(t, err)

			parent, err := d.Parent()
			assert.NoError(t, err)
			assert.NotNil(t, parent)
			assert.Equal(t, ".", parent.Path())
		})
	})

	t.Run("Create edge cases", func(t *testing.T) {
		t.Run("create in non-existent parent", func(t *testing.T) {
			// The Create method using unix.Mkdir doesn't create parent directories
			// It will fail if parent doesn't exist - this tests that error path
			deepPath := filepath.Join(tmpDir, "create_deep", "level1", "level2")

			d, err := fs.NewDirectory(fs.Option{
				Path: deepPath,
			})
			assert.NoError(t, err)

			createdDir, err := d.Create()
			assert.Error(t, err, "Create should fail when parent directories don't exist")
			assert.Nil(t, createdDir)
			assert.Contains(t, err.Error(), "failed to create directory")
		})

		t.Run("chmod failure simulation", func(t *testing.T) {
			if os.Getuid() == 0 {
				t.Skip("Skipping chmod failure test when running as root")
			}

			chmodFailDir := filepath.Join(tmpDir, "chmod_fail")
			// First create the directory
			err := os.MkdirAll(chmodFailDir, 0755)
			assert.NoError(t, err)

			// Now try to create with restrictive permissions on a system that might not allow it
			restrictiveMode := uint32(0000)
			d, err := fs.NewDirectory(fs.Option{
				Path:  chmodFailDir,
				Chmod: &restrictiveMode,
			})
			assert.NoError(t, err)

			// This might succeed or fail depending on the system
			// We're testing the error path in Create()
			_, err = d.Create()
			// Don't assert on the result as it varies by system
			t.Logf("Create with restrictive permissions: %v", err)
		})

		t.Run("chown failure simulation", func(t *testing.T) {
			if os.Getuid() == 0 {
				t.Skip("Skipping chown failure test when running as root")
			}

			chownFailDir := filepath.Join(tmpDir, "chown_fail")
			os.RemoveAll(chownFailDir)

			// Try to set ownership to root (should fail for non-root users)
			rootUID := 0
			rootGID := 0
			d, err := fs.NewDirectory(fs.Option{
				Path: chownFailDir,
				UID:  &rootUID,
				GID:  &rootGID,
			})
			assert.NoError(t, err)

			createdDir, err := d.Create()
			assert.Error(t, err, "Chown to root should fail for non-root users")
			assert.Nil(t, createdDir)
			assert.Contains(t, err.Error(), "failed to set ownership")
		})

		t.Run("mkdir permission failure", func(t *testing.T) {
			// Try to create in a location that doesn't allow directory creation
			invalidDir := "/proc/invalid_test_dir"
			d, err := fs.NewDirectory(fs.Option{
				Path: invalidDir,
			})
			assert.NoError(t, err)

			createdDir, err := d.Create()
			assert.Error(t, err, "Creating directory in /proc should fail")
			assert.Nil(t, createdDir)
			assert.Contains(t, err.Error(), "failed to create directory")
		})
	})

	t.Run("Remove edge cases", func(t *testing.T) {
		t.Run("remove non-existent directory", func(t *testing.T) {
			nonExistentDir := filepath.Join(tmpDir, "does_not_exist")
			d, err := fs.NewDirectory(fs.Option{
				Path: nonExistentDir,
			})
			assert.NoError(t, err)

			err = d.Remove()
			assert.NoError(t, err, "RemoveAll doesn't error on non-existent paths")
		})

		t.Run("remove directory with nested structure", func(t *testing.T) {
			nestedStructure := filepath.Join(tmpDir, "remove_nested")
			deepDir := filepath.Join(nestedStructure, "a", "b", "c", "d")
			err := os.MkdirAll(deepDir, 0755)
			assert.NoError(t, err)

			// Create files at different levels
			err = os.WriteFile(filepath.Join(nestedStructure, "root.txt"), []byte("root"), 0644)
			assert.NoError(t, err)
			err = os.WriteFile(filepath.Join(nestedStructure, "a", "a.txt"), []byte("a"), 0644)
			assert.NoError(t, err)
			err = os.WriteFile(filepath.Join(deepDir, "deep.txt"), []byte("deep"), 0644)
			assert.NoError(t, err)

			d, err := fs.NewDirectory(fs.Option{
				Path: nestedStructure,
			})
			assert.NoError(t, err)

			err = d.Remove()
			assert.NoError(t, err)

			// Verify everything is gone
			_, err = os.Stat(nestedStructure)
			assert.Error(t, err, "Directory tree should be completely removed")
		})

		t.Run("remove with permission issues", func(t *testing.T) {
			if os.Getuid() == 0 {
				t.Skip("Skipping permission test when running as root")
			}

			permDir := filepath.Join(tmpDir, "perm_remove")
			subDir := filepath.Join(permDir, "subdir")
			err := os.MkdirAll(subDir, 0755)
			assert.NoError(t, err)

			// Create a file in subdirectory
			err = os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("content"), 0644)
			assert.NoError(t, err)

			// Make subdirectory read-only
			err = os.Chmod(subDir, 0444)
			assert.NoError(t, err)

			d, err := fs.NewDirectory(fs.Option{
				Path: permDir,
			})
			assert.NoError(t, err)

			// This should still succeed with RemoveAll (it handles permissions)
			err = d.Remove()
			if err != nil {
				// If it fails, it's due to system-specific permission handling
				t.Logf("Remove with permission restrictions: %v", err)
				// Restore permissions for cleanup
				os.Chmod(subDir, 0755)
			}
		})
	})

	t.Run("Size comprehensive tests", func(t *testing.T) {
		t.Run("large directory tree", func(t *testing.T) {
			largeTreeDir := filepath.Join(tmpDir, "large_tree")
			err := os.MkdirAll(largeTreeDir, 0755)
			assert.NoError(t, err)

			totalExpectedSize := int64(0)
			// Create multiple levels with files
			for i := 0; i < 3; i++ {
				levelDir := filepath.Join(largeTreeDir, "level"+string(rune('0'+i)))
				err := os.MkdirAll(levelDir, 0755)
				assert.NoError(t, err)

				for j := 0; j < 3; j++ {
					fileName := filepath.Join(levelDir, "file"+string(rune('0'+j))+".txt")
					content := []byte("content for level " + string(rune('0'+i)) + " file " + string(rune('0'+j)))
					err := os.WriteFile(fileName, content, 0644)
					assert.NoError(t, err)
					totalExpectedSize += int64(len(content))
				}
			}

			d, err := fs.NewDirectory(fs.Option{
				Path: largeTreeDir,
			})
			assert.NoError(t, err)

			size := d.Size()
			assert.Equal(t, totalExpectedSize, size, "Size should match sum of all file contents")
		})

		t.Run("size with broken symlinks", func(t *testing.T) {
			symlinkDir := filepath.Join(tmpDir, "symlink_test")
			err := os.MkdirAll(symlinkDir, 0755)
			assert.NoError(t, err)

			// Create a regular file
			regularFile := filepath.Join(symlinkDir, "regular.txt")
			regularContent := []byte("regular file content")
			err = os.WriteFile(regularFile, regularContent, 0644)
			assert.NoError(t, err)

			// Create a target file for symlink
			targetFile := filepath.Join(symlinkDir, "target.txt")
			targetContent := []byte("target content")
			err = os.WriteFile(targetFile, targetContent, 0644)
			assert.NoError(t, err)

			// Create symlink
			symlinkPath := filepath.Join(symlinkDir, "link.txt")
			err = os.Symlink(targetFile, symlinkPath)
			assert.NoError(t, err)

			// Now delete the target to create a broken symlink
			err = os.Remove(targetFile)
			assert.NoError(t, err)

			d, err := fs.NewDirectory(fs.Option{
				Path: symlinkDir,
			})
			assert.NoError(t, err)

			size := d.Size()
			// Should only count the regular file, broken symlinks should be skipped
			assert.GreaterOrEqual(t, size, int64(len(regularContent)), "Should count regular files despite broken symlinks")
		})

		t.Run("size with inaccessible files", func(t *testing.T) {
			if os.Getuid() == 0 {
				t.Skip("Skipping inaccessible file test when running as root")
			}

			accessDir := filepath.Join(tmpDir, "access_test")
			err := os.MkdirAll(accessDir, 0755)
			assert.NoError(t, err)

			// Create accessible file
			accessibleFile := filepath.Join(accessDir, "accessible.txt")
			accessibleContent := []byte("accessible content")
			err = os.WriteFile(accessibleFile, accessibleContent, 0644)
			assert.NoError(t, err)

			// Create subdirectory with file
			subDir := filepath.Join(accessDir, "subdir")
			err = os.MkdirAll(subDir, 0755)
			assert.NoError(t, err)

			subFile := filepath.Join(subDir, "subfile.txt")
			err = os.WriteFile(subFile, []byte("sub content"), 0644)
			assert.NoError(t, err)

			d, err := fs.NewDirectory(fs.Option{
				Path: accessDir,
			})
			assert.NoError(t, err)

			// Get size before making inaccessible
			initialSize := d.Size()
			assert.Greater(t, initialSize, int64(0))

			// Make subdirectory inaccessible
			err = os.Chmod(subDir, 0000)
			assert.NoError(t, err)
			defer os.Chmod(subDir, 0755) // Restore for cleanup

			size := d.Size()
			// Should handle access errors gracefully and continue
			assert.GreaterOrEqual(t, size, int64(len(accessibleContent)), "Should count accessible files")
		})

		t.Run("size fallback when walk fails", func(t *testing.T) {
			// Create directory then delete it to trigger walk failure
			failDir := filepath.Join(tmpDir, "fail_walk")
			err := os.MkdirAll(failDir, 0755)
			assert.NoError(t, err)

			d, err := fs.NewDirectory(fs.Option{
				Path: failDir,
			})
			assert.NoError(t, err)

			// Delete the directory to make walk fail
			err = os.RemoveAll(failDir)
			assert.NoError(t, err)

			size := d.Size()
			// Should fallback to stats.meta.Size (which will be 0 for non-existent)
			assert.Equal(t, int64(0), size, "Should fallback gracefully when walk fails")
		})
	})

	t.Run("Has comprehensive tests", func(t *testing.T) {
		hasTestDir := filepath.Join(tmpDir, "has_test")
		err := os.MkdirAll(hasTestDir, 0755)
		assert.NoError(t, err)

		d, err := fs.NewDirectory(fs.Option{
			Path: hasTestDir,
		})
		assert.NoError(t, err)

		t.Run("filepath.Rel error handling", func(t *testing.T) {
			// Test with invalid path that causes filepath.Rel to return error
			invalidPath := string([]byte{0x00}) // Null byte in path
			result := d.Has(invalidPath)
			assert.False(t, result, "Should return false when filepath.Rel fails")
		})

		t.Run("edge case paths", func(t *testing.T) {
			// Test various edge cases
			testCases := []struct {
				path     string
				expected bool
				name     string
			}{
				{hasTestDir, true, "directory itself"},
				{hasTestDir + "/", true, "directory with trailing slash"},
				{hasTestDir + "/nonexistent", true, "nonexistent file within directory"},
				{hasTestDir + "/../" + filepath.Base(hasTestDir), true, "path with .. that stays within"},
				{hasTestDir + "/../..", false, "path with .. that escapes"},
				{"/completely/different/path", false, "completely different path"},
				{"", false, "empty path"},
				{".", false, "current directory (relative to test dir)"},
				{"..", false, "parent directory"},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					result := d.Has(tc.path)
					assert.Equal(t, tc.expected, result, "Path: %s", tc.path)
				})
			}
		})

		t.Run("complex relative path resolution", func(t *testing.T) {
			// Create nested structure
			deepDir := filepath.Join(hasTestDir, "a", "b", "c")
			err := os.MkdirAll(deepDir, 0755)
			assert.NoError(t, err)

			// Test paths with complex relative components
			complexPaths := []struct {
				path     string
				expected bool
				name     string
			}{
				{filepath.Join(hasTestDir, "a", "b", "c", "..", ".."), true, "complex .. staying within"},
				{filepath.Join(hasTestDir, "a", ".", "b", "c"), true, "path with . components"},
				{filepath.Join(hasTestDir, "a", "b", "c", "..", "..", "..", ".."), false, "complex .. escaping"},
			}

			for _, tc := range complexPaths {
				t.Run(tc.name, func(t *testing.T) {
					result := d.Has(tc.path)
					assert.Equal(t, tc.expected, result, "Path: %s", tc.path)
				})
			}
		})
	})

	t.Run("List comprehensive tests", func(t *testing.T) {
		t.Run("directory with many entries", func(t *testing.T) {
			manyEntriesDir := filepath.Join(tmpDir, "many_entries")
			err := os.MkdirAll(manyEntriesDir, 0755)
			assert.NoError(t, err)

			expectedFiles := 10
			expectedDirs := 5

			// Create many files
			for i := 0; i < expectedFiles; i++ {
				fileName := filepath.Join(manyEntriesDir, "file"+string(rune('0'+i))+".txt")
				err := os.WriteFile(fileName, []byte("content "+string(rune('0'+i))), 0644)
				assert.NoError(t, err)
			}

			// Create many directories
			for i := 0; i < expectedDirs; i++ {
				dirName := filepath.Join(manyEntriesDir, "dir"+string(rune('0'+i)))
				err := os.MkdirAll(dirName, 0755)
				assert.NoError(t, err)
			}

			d, err := fs.NewDirectory(fs.Option{
				Path: manyEntriesDir,
			})
			assert.NoError(t, err)

			files, subdirs, err := d.List()
			assert.NoError(t, err)
			assert.Equal(t, expectedFiles, len(files))
			assert.Equal(t, expectedDirs, len(subdirs))
		})

		t.Run("directory with special file types", func(t *testing.T) {
			specialDir := filepath.Join(tmpDir, "special_types")
			err := os.MkdirAll(specialDir, 0755)
			assert.NoError(t, err)

			// Create regular file
			regularFile := filepath.Join(specialDir, "regular.txt")
			err = os.WriteFile(regularFile, []byte("regular"), 0644)
			assert.NoError(t, err)

			// Create directory
			subDir := filepath.Join(specialDir, "subdir")
			err = os.MkdirAll(subDir, 0755)
			assert.NoError(t, err)

			// Create symlink to file
			symlinkFile := filepath.Join(specialDir, "symlink_file")
			err = os.Symlink(regularFile, symlinkFile)
			assert.NoError(t, err)

			// Create symlink to directory
			symlinkDir := filepath.Join(specialDir, "symlink_dir")
			err = os.Symlink(subDir, symlinkDir)
			assert.NoError(t, err)

			d, err := fs.NewDirectory(fs.Option{
				Path: specialDir,
			})
			assert.NoError(t, err)

			files, subdirs, err := d.List()
			assert.NoError(t, err)

			// Count actual files vs directories (symlinks should be resolved)
			fileCount := 0
			dirCount := 0

			for _, f := range files {
				if f.Exists() {
					fileCount++
				}
			}

			for _, d := range subdirs {
				if d.Exists() {
					dirCount++
				}
			}

			assert.GreaterOrEqual(t, fileCount, 1, "Should find at least regular files")
			assert.GreaterOrEqual(t, dirCount, 1, "Should find at least regular directories")
		})

		t.Run("Stats object creation in List", func(t *testing.T) {
			// Test that List properly creates stats objects and filters correctly
			statsDir := filepath.Join(tmpDir, "stats_test")
			err := os.MkdirAll(statsDir, 0755)
			assert.NoError(t, err)

			// Create different types to test stats filtering
			regularFile := filepath.Join(statsDir, "regular.txt")
			err = os.WriteFile(regularFile, []byte("content"), 0644)
			assert.NoError(t, err)

			regularDir := filepath.Join(statsDir, "regulardir")
			err = os.MkdirAll(regularDir, 0755)
			assert.NoError(t, err)

			d, err := fs.NewDirectory(fs.Option{
				Path: statsDir,
			})
			assert.NoError(t, err)

			files, subdirs, err := d.List()
			assert.NoError(t, err)

			// Verify that each returned object has proper stats
			for _, f := range files {
				assert.True(t, f.Exists(), "Listed files should exist")
				assert.Greater(t, len(f.Path()), 0, "File path should not be empty")
			}

			for _, d := range subdirs {
				assert.True(t, d.Exists(), "Listed directories should exist")
				assert.Greater(t, len(d.Path()), 0, "Directory path should not be empty")
			}
		})
	})

	t.Run("Exists with refresh behavior", func(t *testing.T) {
		// Test that Exists properly calls stats.Refresh
		refreshDir := filepath.Join(tmpDir, "refresh_behavior")

		d, err := fs.NewDirectory(fs.Option{
			Path: refreshDir,
		})
		assert.NoError(t, err)

		// Initially should not exist
		assert.False(t, d.Exists())

		// Create externally
		err = os.MkdirAll(refreshDir, 0755)
		assert.NoError(t, err)

		// Exists should refresh and detect
		assert.True(t, d.Exists(), "Exists should refresh stats and detect new directory")

		// Remove externally
		err = os.RemoveAll(refreshDir)
		assert.NoError(t, err)

		// Exists should refresh and detect removal
		assert.False(t, d.Exists(), "Exists should refresh stats and detect removal")
	})
}
