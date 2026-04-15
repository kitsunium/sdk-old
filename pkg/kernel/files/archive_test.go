package files_test

import (
	"os"
	"testing"

	"github.com/kitsunium/sdk/pkg/kernel/files"
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
			archive := files.NewArchive(tmpDir+"/archive.zip", files.ArchiveOptions{
				ArchiveType:     files.ZIP,
				CompressionType: files.GZIP,
			})
			assert.NotNil(t, archive)
			assert.Equal(t, tmpDir+"/archive.zip", archive.Path())
		})

		t.Run("invalid path", func(t *testing.T) {
			archive := files.NewArchive("", files.ArchiveOptions{
				ArchiveType:     files.ZIP,
				CompressionType: files.GZIP,
			})
			assert.NotNil(t, archive)
			assert.Empty(t, archive.Path(), "Expected path to be empty for invalid archive path")
		})
	})

	t.Run("AddFile", func(t *testing.T) {
		t.Run("nominal valid file", func(t *testing.T) {
			archive := files.NewArchive(tmpDir+"/archive.zip", files.ArchiveOptions{
				ArchiveType:     files.ZIP,
				CompressionType: files.GZIP,
			})
			err := archive.AddFile(tmpFile.Name())
			assert.NoError(t, err, "Adding a valid file to the archive should not return an error")
		})

		t.Run("non-existent file", func(t *testing.T) {
			archive := files.NewArchive(tmpDir+"/archive.zip", files.ArchiveOptions{
				ArchiveType:     files.ZIP,
				CompressionType: files.GZIP,
			})
			err := archive.AddFile("/non/existent/file")
			assert.Error(t, err, "Adding a non-existent file should return an error")
		})

		t.Run("duplicate file", func(t *testing.T) {
			archive := files.NewArchive(tmpDir+"/archive.zip", files.ArchiveOptions{
				ArchiveType:     files.ZIP,
				CompressionType: files.GZIP,
			})
			err := archive.AddFile(tmpFile.Name())
			assert.NoError(t, err, "Adding a valid file to the archive should not return an error")
			err = archive.AddFile(tmpFile.Name())
			assert.NoError(t, err, "Adding a duplicate file should not return an error")
		})

		t.Run("file inside already added directory", func(t *testing.T) {
			archive := files.NewArchive(tmpDir+"/archive.zip", files.ArchiveOptions{
				ArchiveType:     files.ZIP,
				CompressionType: files.GZIP,
			})
			err := archive.AddDirectory(nestedDir)
			assert.NoError(t, err, "Adding a directory should not return an error")
			err = archive.AddFile(nestedFile)
			assert.NoError(t, err, "Adding a file already part of an added directory should not return an error")
		})
	})

	t.Run("AddDirectory", func(t *testing.T) {
		t.Run("nominal valid directory", func(t *testing.T) {
			archive := files.NewArchive(tmpDir+"/archive.zip", files.ArchiveOptions{
				ArchiveType:     files.ZIP,
				CompressionType: files.GZIP,
			})
			err := archive.AddDirectory(tmpDir)
			assert.NoError(t, err, "Adding a valid directory to the archive should not return an error")
		})

		t.Run("non-existent directory", func(t *testing.T) {
			archive := files.NewArchive(tmpDir+"/archive.zip", files.ArchiveOptions{
				ArchiveType:     files.ZIP,
				CompressionType: files.GZIP,
			})
			err := archive.AddDirectory("/non/existent/dir")
			assert.Error(t, err, "Adding a non-existent directory should return an error")
		})

		t.Run("nested directory", func(t *testing.T) {
			archive := files.NewArchive(tmpDir+"/archive.zip", files.ArchiveOptions{
				ArchiveType:     files.ZIP,
				CompressionType: files.GZIP,
			})
			err := archive.AddDirectory(nestedDir)
			assert.NoError(t, err, "Adding a nested directory should not return an error")
		})

		t.Run("directory inside already added parent directory", func(t *testing.T) {
			archive := files.NewArchive(tmpDir+"/archive.zip", files.ArchiveOptions{
				ArchiveType:     files.ZIP,
				CompressionType: files.GZIP,
			})
			err := archive.AddDirectory(tmpDir)
			assert.NoError(t, err, "Adding a directory should not return an error")
			err = archive.AddDirectory(nestedDir)
			assert.NoError(t, err,
				"Adding a nested directory that is already part of an added parent directory should not return an error")
		})

		t.Run("duplicate directory", func(t *testing.T) {
			archive := files.NewArchive(tmpDir+"/archive.zip", files.ArchiveOptions{
				ArchiveType:     files.ZIP,
				CompressionType: files.GZIP,
			})
			err := archive.AddDirectory(nestedDir)
			assert.NoError(t, err, "Adding a valid directory should not return an error")
			err = archive.AddDirectory(nestedDir)
			assert.NoError(t, err, "Adding a duplicate directory should not return an error")
		})
	})

	t.Run("Compress", func(t *testing.T) {
		t.Run("nominal valid archive", func(t *testing.T) {
			archive := files.NewArchive(tmpDir+"/archive.zip", files.ArchiveOptions{
				ArchiveType:     files.ZIP,
				CompressionType: files.GZIP,
			})
			err := archive.Compress()
			assert.NoError(t, err, "Compressing a valid archive should not return an error")
		})

		t.Run("empty archive", func(t *testing.T) {
			emptyArchive := files.NewArchive(tmpDir+"/empty_archive.zip", files.ArchiveOptions{
				ArchiveType:     files.ZIP,
				CompressionType: files.GZIP,
			})
			err := emptyArchive.Compress()
			assert.NoError(t, err, "Compressing an empty archive should not return an error")
		})
	})
}
