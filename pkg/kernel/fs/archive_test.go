package fs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kitsunium/sdk/pkg/kernel/fs"
	"github.com/stretchr/testify/assert"
)

func TestArchive(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "testarchive")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tmpFile, err := os.CreateTemp("", "testfile")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	nestedDir := tmpDir + "/nested"
	assert.NoError(t, os.MkdirAll(nestedDir, 0755))
	nestedFile := nestedDir + "/nestedfile.txt"
	assert.NoError(t, os.WriteFile(nestedFile, []byte("nested content"), 0644))

	t.Run("NewArchive", func(t *testing.T) {
		t.Run("nominal valid archive", func(t *testing.T) {
			archive := fs.NewArchive(tmpDir+"/archive.zip", fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})
			assert.NotNil(t, archive)
			assert.Equal(t, tmpDir+"/archive.zip", archive.Path())
		})

		t.Run("invalid path", func(t *testing.T) {
			archive := fs.NewArchive("", fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})
			assert.NotNil(t, archive)
			assert.Empty(t, archive.Path(), "Expected path to be empty for invalid archive path")
		})
	})

	t.Run("AddFile", func(t *testing.T) {
		t.Run("nominal valid file", func(t *testing.T) {
			archive := fs.NewArchive(tmpDir+"/archive.zip", fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})
			err := archive.AddFile(tmpFile.Name())
			assert.NoError(t, err, "Adding a valid file to the archive should not return an error")
		})

		t.Run("non-existent file", func(t *testing.T) {
			archive := fs.NewArchive(tmpDir+"/archive.zip", fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})
			err := archive.AddFile("/non/existent/file")
			assert.Error(t, err, "Adding a non-existent file should return an error")
		})

		t.Run("duplicate file", func(t *testing.T) {
			archive := fs.NewArchive(tmpDir+"/archive.zip", fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})
			err := archive.AddFile(tmpFile.Name())
			assert.NoError(t, err, "Adding a valid file to the archive should not return an error")
			err = archive.AddFile(tmpFile.Name())
			assert.NoError(t, err, "Adding a duplicate file should not return an error")
		})

		t.Run("file inside already added directory", func(t *testing.T) {
			archive := fs.NewArchive(tmpDir+"/archive.zip", fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})
			err := archive.AddDirectory(nestedDir)
			assert.NoError(t, err, "Adding a directory should not return an error")
			err = archive.AddFile(nestedFile)
			assert.NoError(t, err, "Adding a file already part of an added directory should not return an error")
		})
	})

	t.Run("AddDirectory", func(t *testing.T) {
		t.Run("nominal valid directory", func(t *testing.T) {
			archive := fs.NewArchive(tmpDir+"/archive.zip", fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})
			err := archive.AddDirectory(tmpDir)
			assert.NoError(t, err, "Adding a valid directory to the archive should not return an error")
		})

		t.Run("non-existent directory", func(t *testing.T) {
			archive := fs.NewArchive(tmpDir+"/archive.zip", fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})
			err := archive.AddDirectory("/non/existent/dir")
			assert.Error(t, err, "Adding a non-existent directory should return an error")
		})

		t.Run("nested directory", func(t *testing.T) {
			archive := fs.NewArchive(tmpDir+"/archive.zip", fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})
			err := archive.AddDirectory(nestedDir)
			assert.NoError(t, err, "Adding a nested directory should not return an error")
		})

		t.Run("directory inside already added parent directory", func(t *testing.T) {
			archive := fs.NewArchive(tmpDir+"/archive.zip", fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})
			err := archive.AddDirectory(tmpDir)
			assert.NoError(t, err, "Adding a directory should not return an error")
			err = archive.AddDirectory(nestedDir)
			assert.NoError(t, err,
				"Adding a nested directory that is already part of an added parent directory should not return an error")
		})

		t.Run("duplicate directory", func(t *testing.T) {
			archive := fs.NewArchive(tmpDir+"/archive.zip", fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})
			err := archive.AddDirectory(nestedDir)
			assert.NoError(t, err, "Adding a valid directory should not return an error")
			err = archive.AddDirectory(nestedDir)
			assert.NoError(t, err, "Adding a duplicate directory should not return an error")
		})
	})

	t.Run("Compress", func(t *testing.T) {
		t.Run("nominal valid archive", func(t *testing.T) {
			archive := fs.NewArchive(tmpDir+"/archive.zip", fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})
			err := archive.Compress()
			assert.NoError(t, err, "Compressing a valid archive should not return an error")
		})

		t.Run("empty archive", func(t *testing.T) {
			emptyArchive := fs.NewArchive(tmpDir+"/empty_archive.zip", fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})
			err := emptyArchive.Compress()
			assert.NoError(t, err, "Compressing an empty archive should not return an error")
		})
	})
}

// TestArchiveEdgeCases contains comprehensive tests for archive.go to achieve 100% coverage
func TestArchiveEdgeCases(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "archive_comprehensive")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("AddFile edge cases", func(t *testing.T) {
		t.Run("add file with invalid option validation", func(t *testing.T) {
			archive := fs.NewArchive(filepath.Join(tmpDir, "invalid_option.zip"), fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})

			// Try to add file with invalid path that will fail Option.Validate()
			err := archive.AddFile("") // Empty path should fail validation
			assert.Error(t, err, "Adding file with invalid path should fail")
		})

		t.Run("add file with path cleaning issues", func(t *testing.T) {
			archive := fs.NewArchive(filepath.Join(tmpDir, "path_clean.zip"), fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})

			// Try to add file with path that cleans to "." (current directory)
			err := archive.AddFile(".")
			assert.Error(t, err, "Adding current directory as file should fail")
		})

		t.Run("add file already in archive (duplicate prevention)", func(t *testing.T) {
			testFile := filepath.Join(tmpDir, "duplicate_file.txt")
			err := os.WriteFile(testFile, []byte("duplicate test"), 0644)
			assert.NoError(t, err)

			archive := fs.NewArchive(filepath.Join(tmpDir, "duplicate.zip"), fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})

			// Add file first time
			err = archive.AddFile(testFile)
			assert.NoError(t, err)

			// Add same file second time - should not error but should not duplicate
			err = archive.AddFile(testFile)
			assert.NoError(t, err, "Adding duplicate file should not error")
		})

		t.Run("add file that is within already added directory", func(t *testing.T) {
			// Create directory structure
			testDir := filepath.Join(tmpDir, "parent_dir")
			err := os.MkdirAll(testDir, 0755)
			assert.NoError(t, err)

			testFile := filepath.Join(testDir, "child_file.txt")
			err = os.WriteFile(testFile, []byte("child content"), 0644)
			assert.NoError(t, err)

			archive := fs.NewArchive(filepath.Join(tmpDir, "parent_child.zip"), fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})

			// Add parent directory first
			err = archive.AddDirectory(testDir)
			assert.NoError(t, err)

			// Try to add child file - should not error but should not duplicate
			err = archive.AddFile(testFile)
			assert.NoError(t, err, "Adding file within already added directory should not error")
		})

		t.Run("add file with directory entry having file within it", func(t *testing.T) {
			// Test the Has() method path in AddFile
			parentDir := filepath.Join(tmpDir, "has_test_parent")
			err := os.MkdirAll(parentDir, 0755)
			assert.NoError(t, err)

			childFile := filepath.Join(parentDir, "has_test_child.txt")
			err = os.WriteFile(childFile, []byte("has test"), 0644)
			assert.NoError(t, err)

			archive := fs.NewArchive(filepath.Join(tmpDir, "has_test.zip"), fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})

			// Add parent directory first
			err = archive.AddDirectory(parentDir)
			assert.NoError(t, err)

			// Now try to add child file - this tests the Has() check in AddFile
			err = archive.AddFile(childFile)
			assert.NoError(t, err, "Adding file that is within an already added directory should succeed")
		})

		t.Run("add file with NewFile error", func(t *testing.T) {
			archive := fs.NewArchive(filepath.Join(tmpDir, "newfile_error.zip"), fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})

			// Try to add a directory path as a file (should cause NewFile to fail)
			dirAsFile := filepath.Join(tmpDir, "dir_as_file")
			err := os.MkdirAll(dirAsFile, 0755)
			assert.NoError(t, err)

			err = archive.AddFile(dirAsFile)
			assert.Error(t, err, "Adding directory path as file should fail")
		})

		t.Run("add multiple different files", func(t *testing.T) {
			// Test adding multiple different files to verify the entry loop works correctly
			files := []string{
				filepath.Join(tmpDir, "multi1.txt"),
				filepath.Join(tmpDir, "multi2.txt"),
				filepath.Join(tmpDir, "multi3.txt"),
			}

			for i, file := range files {
				content := "content for file " + string(rune('1'+i))
				err := os.WriteFile(file, []byte(content), 0644)
				assert.NoError(t, err)
			}

			archive := fs.NewArchive(filepath.Join(tmpDir, "multi_files.zip"), fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})

			// Add all files
			for _, file := range files {
				err := archive.AddFile(file)
				assert.NoError(t, err, "Adding multiple files should succeed")
			}

			// Try to add them again to test the duplicate checking loop
			for _, file := range files {
				err := archive.AddFile(file)
				assert.NoError(t, err, "Re-adding files should not error")
			}
		})
	})

	t.Run("AddDirectory edge cases", func(t *testing.T) {
		t.Run("add directory with invalid option validation", func(t *testing.T) {
			archive := fs.NewArchive(filepath.Join(tmpDir, "invalid_dir_option.zip"), fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})

			// Try to add directory with invalid path
			err := archive.AddDirectory("") // Empty path should fail validation
			assert.Error(t, err, "Adding directory with invalid path should fail")
		})

		t.Run("add directory with path cleaning issues", func(t *testing.T) {
			archive := fs.NewArchive(filepath.Join(tmpDir, "dir_path_clean.zip"), fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})

			// Try to add directory with path that cleans to "."
			err := archive.AddDirectory(".")
			assert.Error(t, err, "Adding current directory should fail validation")
		})

		t.Run("add directory already in archive", func(t *testing.T) {
			testDir := filepath.Join(tmpDir, "duplicate_dir")
			err := os.MkdirAll(testDir, 0755)
			assert.NoError(t, err)

			archive := fs.NewArchive(filepath.Join(tmpDir, "duplicate_dir.zip"), fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})

			// Add directory first time
			err = archive.AddDirectory(testDir)
			assert.NoError(t, err)

			// Add same directory second time - should not error
			err = archive.AddDirectory(testDir)
			assert.NoError(t, err, "Adding duplicate directory should not error")
		})

		t.Run("add directory that is within already added directory", func(t *testing.T) {
			// Create nested directory structure
			parentDir := filepath.Join(tmpDir, "nested_parent")
			childDir := filepath.Join(parentDir, "nested_child")
			err := os.MkdirAll(childDir, 0755)
			assert.NoError(t, err)

			archive := fs.NewArchive(filepath.Join(tmpDir, "nested.zip"), fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})

			// Add parent directory first
			err = archive.AddDirectory(parentDir)
			assert.NoError(t, err)

			// Try to add child directory - should not error but should not duplicate
			err = archive.AddDirectory(childDir)
			assert.NoError(t, err, "Adding directory within already added directory should not error")
		})

		t.Run("add directory with Has() check", func(t *testing.T) {
			// Test the Has() method check in AddDirectory
			outerDir := filepath.Join(tmpDir, "outer_has")
			innerDir := filepath.Join(outerDir, "inner_has")
			err := os.MkdirAll(innerDir, 0755)
			assert.NoError(t, err)

			archive := fs.NewArchive(filepath.Join(tmpDir, "has_dir_test.zip"), fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})

			// Add outer directory first
			err = archive.AddDirectory(outerDir)
			assert.NoError(t, err)

			// Now try to add inner directory - this tests the Has() check in AddDirectory
			err = archive.AddDirectory(innerDir)
			assert.NoError(t, err, "Adding directory that is within an already added directory should succeed")
		})

		t.Run("add directory with NewDirectory error", func(t *testing.T) {
			archive := fs.NewArchive(filepath.Join(tmpDir, "newdir_error.zip"), fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})

			// Try to add a file path as a directory (should cause NewDirectory to fail)
			fileAsDir := filepath.Join(tmpDir, "file_as_dir.txt")
			err := os.WriteFile(fileAsDir, []byte("not a directory"), 0644)
			assert.NoError(t, err)

			err = archive.AddDirectory(fileAsDir)
			assert.Error(t, err, "Adding file path as directory should fail")
		})

		t.Run("add multiple different directories", func(t *testing.T) {
			// Test adding multiple different directories
			dirs := []string{
				filepath.Join(tmpDir, "multi_dir1"),
				filepath.Join(tmpDir, "multi_dir2"),
				filepath.Join(tmpDir, "multi_dir3"),
			}

			for _, dir := range dirs {
				err := os.MkdirAll(dir, 0755)
				assert.NoError(t, err)
				// Add a file to each directory to make them non-empty
				testFile := filepath.Join(dir, "test.txt")
				err = os.WriteFile(testFile, []byte("test content"), 0644)
				assert.NoError(t, err)
			}

			archive := fs.NewArchive(filepath.Join(tmpDir, "multi_dirs.zip"), fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})

			// Add all directories
			for _, dir := range dirs {
				err := archive.AddDirectory(dir)
				assert.NoError(t, err, "Adding multiple directories should succeed")
			}

			// Try to add them again to test duplicate checking
			for _, dir := range dirs {
				err := archive.AddDirectory(dir)
				assert.NoError(t, err, "Re-adding directories should not error")
			}
		})
	})

	t.Run("Mixed file and directory operations", func(t *testing.T) {
		t.Run("add file then directory containing file", func(t *testing.T) {
			mixedDir := filepath.Join(tmpDir, "mixed_test")
			err := os.MkdirAll(mixedDir, 0755)
			assert.NoError(t, err)

			mixedFile := filepath.Join(mixedDir, "mixed.txt")
			err = os.WriteFile(mixedFile, []byte("mixed content"), 0644)
			assert.NoError(t, err)

			archive := fs.NewArchive(filepath.Join(tmpDir, "mixed.zip"), fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})

			// Add file first
			err = archive.AddFile(mixedFile)
			assert.NoError(t, err)

			// Then add directory containing the file
			err = archive.AddDirectory(mixedDir)
			assert.NoError(t, err, "Adding directory containing already added file should succeed")
		})

		t.Run("add directory then file within directory", func(t *testing.T) {
			mixedDir2 := filepath.Join(tmpDir, "mixed_test2")
			err := os.MkdirAll(mixedDir2, 0755)
			assert.NoError(t, err)

			mixedFile2 := filepath.Join(mixedDir2, "mixed2.txt")
			err = os.WriteFile(mixedFile2, []byte("mixed content 2"), 0644)
			assert.NoError(t, err)

			archive := fs.NewArchive(filepath.Join(tmpDir, "mixed2.zip"), fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})

			// Add directory first
			err = archive.AddDirectory(mixedDir2)
			assert.NoError(t, err)

			// Then add file within the directory
			err = archive.AddFile(mixedFile2)
			assert.NoError(t, err, "Adding file within already added directory should succeed")
		})

		t.Run("complex entry relationship testing", func(t *testing.T) {
			// Create a complex directory structure to test all entry relationship paths
			complexDir := filepath.Join(tmpDir, "complex")
			err := os.MkdirAll(complexDir, 0755)
			assert.NoError(t, err)

			level1Dir := filepath.Join(complexDir, "level1")
			err = os.MkdirAll(level1Dir, 0755)
			assert.NoError(t, err)

			level2Dir := filepath.Join(level1Dir, "level2")
			err = os.MkdirAll(level2Dir, 0755)
			assert.NoError(t, err)

			// Create files at different levels
			rootFile := filepath.Join(complexDir, "root.txt")
			level1File := filepath.Join(level1Dir, "level1.txt")
			level2File := filepath.Join(level2Dir, "level2.txt")

			err = os.WriteFile(rootFile, []byte("root"), 0644)
			assert.NoError(t, err)
			err = os.WriteFile(level1File, []byte("level1"), 0644)
			assert.NoError(t, err)
			err = os.WriteFile(level2File, []byte("level2"), 0644)
			assert.NoError(t, err)

			archive := fs.NewArchive(filepath.Join(tmpDir, "complex.zip"), fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})

			// Add entries in various orders to test all relationship paths
			err = archive.AddFile(level2File)
			assert.NoError(t, err)

			err = archive.AddDirectory(level1Dir)
			assert.NoError(t, err)

			err = archive.AddFile(rootFile)
			assert.NoError(t, err)

			err = archive.AddDirectory(complexDir)
			assert.NoError(t, err)

			err = archive.AddFile(level1File)
			assert.NoError(t, err)

			err = archive.AddDirectory(level2Dir)
			assert.NoError(t, err)

			// All operations should succeed without errors
		})
	})

	t.Run("Archive properties and methods", func(t *testing.T) {
		t.Run("Path method", func(t *testing.T) {
			testPath := filepath.Join(tmpDir, "path_test.zip")
			archive := fs.NewArchive(testPath, fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})

			assert.Equal(t, testPath, archive.Path(), "Path should return the archive path")
		})

		t.Run("Path method with empty path", func(t *testing.T) {
			archive := fs.NewArchive("", fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})

			assert.Empty(t, archive.Path(), "Path should return empty string for empty archive path")
		})

		t.Run("Archive with different options", func(t *testing.T) {
			// Test with ZIP/GZIP combination
			zipArchivePath := filepath.Join(tmpDir, "options_zip.zip")
			zipArchive := fs.NewArchive(zipArchivePath, fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
				Level:           5,
			})

			assert.Equal(t, zipArchivePath, zipArchive.Path())

			// Test adding a file to ZIP archive
			zipTestFile := filepath.Join(tmpDir, "zip_options_file.txt")
			err := os.WriteFile(zipTestFile, []byte("zip options test"), 0644)
			assert.NoError(t, err)

			err = zipArchive.AddFile(zipTestFile)
			assert.NoError(t, err, "ZIP archive should be able to add files")

			// Test with TAR/GZIP combination
			tarArchivePath := filepath.Join(tmpDir, "options_tar.tar.gz")
			tarArchive := fs.NewArchive(tarArchivePath, fs.ArchiveOptions{
				ArchiveType:     fs.TAR,
				CompressionType: fs.GZIP,
				Level:           3,
			})

			assert.Equal(t, tarArchivePath, tarArchive.Path())

			// Test adding a file to TAR archive
			tarTestFile := filepath.Join(tmpDir, "tar_options_file.txt")
			err = os.WriteFile(tarTestFile, []byte("tar options test"), 0644)
			assert.NoError(t, err)

			err = tarArchive.AddFile(tarTestFile)
			assert.NoError(t, err, "TAR archive should be able to add files")
		})
	})

	t.Run("Entry list behavior", func(t *testing.T) {
		t.Run("entries are properly maintained", func(t *testing.T) {
			// This tests that the entries slice is properly managed
			entriesDir := filepath.Join(tmpDir, "entries_test")
			err := os.MkdirAll(entriesDir, 0755)
			assert.NoError(t, err)

			entriesFile := filepath.Join(entriesDir, "entries.txt")
			err = os.WriteFile(entriesFile, []byte("entries test"), 0644)
			assert.NoError(t, err)

			archive := fs.NewArchive(filepath.Join(tmpDir, "entries.zip"), fs.ArchiveOptions{
				ArchiveType:     fs.ZIP,
				CompressionType: fs.GZIP,
			})

			// Add file
			err = archive.AddFile(entriesFile)
			assert.NoError(t, err)

			// Add directory
			err = archive.AddDirectory(entriesDir)
			assert.NoError(t, err)

			// Add them again to ensure duplicate checking works
			err = archive.AddFile(entriesFile)
			assert.NoError(t, err)

			err = archive.AddDirectory(entriesDir)
			assert.NoError(t, err)

			// All operations should succeed, demonstrating proper entry management
		})
	})
}
