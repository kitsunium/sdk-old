package files_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kitsunium/sdk/internal/kernel/files"
	"github.com/stretchr/testify/assert"
)

func TestDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "testdir")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("NewDirectory", func(t *testing.T) {
		t.Run("nominal", func(t *testing.T) {
			d, err := files.NewDirectory(files.Option{
				Path: tmpDir,
			})
			assert.NoError(t, err)
			assert.NotNil(t, d)
		})

		t.Run("path empty", func(t *testing.T) {
			d, err := files.NewDirectory(files.Option{})
			assert.Error(t, err)
			assert.Nil(t, d)
		})

		t.Run("create if not exist", func(t *testing.T) {
			newDir := tmpDir + "/newdir"
			d, err := files.NewDirectory(files.Option{
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

			d, err := files.NewDirectory(files.Option{
				Path: tmpFile.Name(),
			})
			assert.Error(t, err)
			assert.Nil(t, d)
		})
	})

	t.Run("Path", func(t *testing.T) {
		d, err := files.NewDirectory(files.Option{
			Path: tmpDir,
		})
		assert.NoError(t, err)
		assert.NotNil(t, d)

		t.Run("nominal", func(t *testing.T) {
			assert.Equal(t, tmpDir, d.Path())
		})
	})

	t.Run("Parent", func(t *testing.T) {
		d, err := files.NewDirectory(files.Option{
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
		d, err := files.NewDirectory(files.Option{
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

		d, err := files.NewDirectory(files.Option{
			Path: tmpDir,
		})
		assert.NoError(t, err)

		files, subdirs, err := d.List()

		assert.NoError(t, err)
		assert.Equal(t, 1, len(files), "expected one file")
		assert.Equal(t, 1, len(subdirs), "expected one subdirectory")
	})

	t.Run("Size", func(t *testing.T) {
		d, err := files.NewDirectory(files.Option{
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

		dir, err := files.NewDirectory(files.Option{Path: tmpDir})
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
