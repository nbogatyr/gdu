package archive

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create nested structure:
	// dir/
	//   top/
	//     file1.txt (13 bytes)
	//     nested/
	//       file2.txt (12 bytes)
	topDir := filepath.Join(dir, "top")
	nestedDir := filepath.Join(topDir, "nested")
	assert.NoError(t, os.MkdirAll(nestedDir, 0o755))
	assert.NoError(t, os.WriteFile(filepath.Join(topDir, "file1.txt"), []byte("hello world\n"), 0o644))
	assert.NoError(t, os.WriteFile(filepath.Join(nestedDir, "file2.txt"), []byte("nested file\n"), 0o644))

	return dir
}

func listTarEntries(t *testing.T, tarPath string) []string {
	t.Helper()
	f, err := os.Open(tarPath)
	assert.NoError(t, err)
	defer f.Close()

	tr := tar.NewReader(f)
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		assert.NoError(t, err)
		names = append(names, hdr.Name)
	}
	return names
}

func listTarGzEntries(t *testing.T, tgzPath string) []string {
	t.Helper()
	f, err := os.Open(tgzPath)
	assert.NoError(t, err)
	defer f.Close()

	gr, err := gzip.NewReader(f)
	assert.NoError(t, err)
	defer gr.Close()

	tr := tar.NewReader(gr)
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		assert.NoError(t, err)
		names = append(names, hdr.Name)
	}
	return names
}

func TestCreateTarDirectory(t *testing.T) {
	dir := createTestDir(t)
	destPath := filepath.Join(dir, "archive.tar")

	err := CreateTar(filepath.Join(dir, "top"), destPath)
	assert.NoError(t, err)
	assert.FileExists(t, destPath)

	entries := listTarEntries(t, destPath)
	assert.Contains(t, entries, "top")
	assert.Contains(t, entries, filepath.Join("top", "file1.txt"))
	assert.Contains(t, entries, filepath.Join("top", "nested"))
	assert.Contains(t, entries, filepath.Join("top", "nested", "file2.txt"))
}

func TestCreateTarGzDirectory(t *testing.T) {
	dir := createTestDir(t)
	destPath := filepath.Join(dir, "archive.tgz")

	err := CreateTarGz(filepath.Join(dir, "top"), destPath)
	assert.NoError(t, err)
	assert.FileExists(t, destPath)

	entries := listTarGzEntries(t, destPath)
	assert.Contains(t, entries, "top")
	assert.Contains(t, entries, filepath.Join("top", "file1.txt"))
	assert.Contains(t, entries, filepath.Join("top", "nested"))
	assert.Contains(t, entries, filepath.Join("top", "nested", "file2.txt"))
}

func TestCreateTarSingleFile(t *testing.T) {
	dir := createTestDir(t)
	destPath := filepath.Join(dir, "single.tar")

	err := CreateTar(filepath.Join(dir, "top", "file1.txt"), destPath)
	assert.NoError(t, err)
	assert.FileExists(t, destPath)

	entries := listTarEntries(t, destPath)
	assert.Contains(t, entries, "file1.txt")
	assert.Len(t, entries, 1)
}

func TestCreateTarNonExistentSource(t *testing.T) {
	dir := t.TempDir()
	destPath := filepath.Join(dir, "archive.tar")

	err := CreateTar(filepath.Join(dir, "nonexistent"), destPath)
	assert.Error(t, err)
}

func TestCreateTarPermissionDeniedDest(t *testing.T) {
	dir := createTestDir(t)
	destPath := filepath.Join("/", "root", "noperm.tar")

	err := CreateTar(filepath.Join(dir, "top"), destPath)
	assert.Error(t, err)
}

func TestCreateTarGzIsSmaller(t *testing.T) {
	dir := createTestDir(t)

	// Create a larger file to make compression noticeable
	largeContent := make([]byte, 10000)
	for i := range largeContent {
		largeContent[i] = 'A'
	}
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "top", "large.txt"), largeContent, 0o644))

	tarPath := filepath.Join(dir, "archive.tar")
	tgzPath := filepath.Join(dir, "archive.tgz")

	assert.NoError(t, CreateTar(filepath.Join(dir, "top"), tarPath))
	assert.NoError(t, CreateTarGz(filepath.Join(dir, "top"), tgzPath))

	tarInfo, err := os.Stat(tarPath)
	assert.NoError(t, err)
	tgzInfo, err := os.Stat(tgzPath)
	assert.NoError(t, err)

	assert.Less(t, tgzInfo.Size(), tarInfo.Size())
}
