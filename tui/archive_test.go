package tui

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/dundee/gdu/v5/internal/testanalyze"
	"github.com/dundee/gdu/v5/internal/testapp"
	"github.com/dundee/gdu/v5/pkg/analyze"
	"github.com/dundee/gdu/v5/pkg/fs"
	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
)

func setupArchiveUI(t *testing.T) (
	*UI, *analyze.Dir, *analyze.Dir, string,
) {
	t.Helper()
	dir := t.TempDir()
	subDir := filepath.Join(dir, "sub")
	assert.NoError(t, os.MkdirAll(subDir, 0o755))
	assert.NoError(t, os.WriteFile(
		filepath.Join(subDir, "file.txt"), []byte("hello"), 0o644,
	))

	parentDir := &analyze.Dir{
		File: &analyze.File{
			Name: filepath.Base(dir),
		},
		BasePath: filepath.Dir(dir),
		Files:    make([]fs.Item, 0, 1),
	}
	childDir := &analyze.Dir{
		File: &analyze.File{
			Name:   "sub",
			Size:   5,
			Usage:  8192,
			Parent: parentDir,
		},
		Files: make([]fs.Item, 0),
	}
	parentDir.Files = fs.Files{childDir}

	return nil, parentDir, childDir, dir
}

func createArchiveUI(t *testing.T) (*UI, *analyze.Dir, string) {
	t.Helper()
	_, parentDir, _, dir := setupArchiveUI(t)

	simScreen := testapp.CreateSimScreen()
	t.Cleanup(simScreen.Fini)

	app := testapp.CreateMockedApp(true)
	ui := CreateUI(app, simScreen, &bytes.Buffer{}, false, true, false, false)
	ui.Analyzer = &testanalyze.MockedAnalyzer{}
	ui.currentDir = parentDir
	ui.topDir = parentDir
	ui.topDirPath = parentDir.GetPath()
	ui.showDir()
	ui.table.Select(0, 0)

	return ui, parentDir, dir
}

func TestConfirmArchive(t *testing.T) {
	ui, _, _ := createArchiveUI(t)

	form := ui.confirmArchive()

	assert.True(t, ui.pages.HasPage("archive"))
	assert.NotNil(t, form)
	assert.Equal(t, "sub.tar", ui.archiveName)
	assert.False(t, ui.archiveGzip)
}

func TestConfirmArchiveEsc(t *testing.T) {
	ui, _, _ := createArchiveUI(t)

	form := ui.confirmArchive()
	formInputFn := form.GetInputCapture()

	assert.True(t, ui.pages.HasPage("archive"))

	formInputFn(tcell.NewEventKey(tcell.KeyEsc, 0, 0))

	assert.False(t, ui.pages.HasPage("archive"))
}

func TestArchiveItem(t *testing.T) {
	ui, _, dir := createArchiveUI(t)
	ui.done = make(chan struct{})

	ui.archiveName = "sub.tar"
	ui.archiveGzip = false
	ui.archiveItem()

	assert.True(t, ui.pages.HasPage("archiving"))

	<-ui.done

	assert.FileExists(t, filepath.Join(dir, "sub.tar"))

	for _, f := range ui.app.(*testapp.MockedApp).GetUpdateDraws() {
		f()
	}
}

func TestArchiveItemTgz(t *testing.T) {
	ui, _, dir := createArchiveUI(t)
	ui.done = make(chan struct{})

	ui.archiveName = "sub.tgz"
	ui.archiveGzip = true
	ui.archiveItem()

	assert.True(t, ui.pages.HasPage("archiving"))

	<-ui.done

	assert.FileExists(t, filepath.Join(dir, "sub.tgz"))

	for _, f := range ui.app.(*testapp.MockedApp).GetUpdateDraws() {
		f()
	}
}

func TestArchiveInArchive(t *testing.T) {
	simScreen := testapp.CreateSimScreen()
	defer simScreen.Fini()

	app := testapp.CreateMockedApp(true)
	ui := CreateUI(app, simScreen, &bytes.Buffer{}, false, true, false, false)
	ui.Analyzer = &testanalyze.MockedAnalyzer{}
	ui.currentDir = &analyze.TarDir{}

	ui.handleArchive()

	assert.True(t, ui.pages.HasPage("error"))
}

func TestArchiveWithNoDelete(t *testing.T) {
	ui, _, _ := createArchiveUI(t)
	ui.SetNoDelete()

	ui.handleArchive()

	assert.False(t, ui.pages.HasPage("archive"))
}

func TestArchiveWithNilCurrentDir(t *testing.T) {
	simScreen := testapp.CreateSimScreen()
	defer simScreen.Fini()

	app := testapp.CreateMockedApp(true)
	ui := CreateUI(app, simScreen, &bytes.Buffer{}, false, true, false, false)
	ui.Analyzer = &testanalyze.MockedAnalyzer{}
	ui.currentDir = nil

	ui.handleArchive()

	assert.False(t, ui.pages.HasPage("archive"))
}

func TestArchiveItemError(t *testing.T) {
	simScreen := testapp.CreateSimScreen()
	defer simScreen.Fini()

	parentDir := &analyze.Dir{
		File: &analyze.File{
			Name: "parent",
		},
		BasePath: "/nonexistent_base",
		Files:    make([]fs.Item, 0, 1),
	}
	childDir := &analyze.Dir{
		File: &analyze.File{
			Name:   "sub",
			Size:   5,
			Usage:  8192,
			Parent: parentDir,
		},
		Files: make([]fs.Item, 0),
	}
	parentDir.Files = fs.Files{childDir}

	app := testapp.CreateMockedApp(true)
	ui := CreateUI(app, simScreen, &bytes.Buffer{}, false, true, false, false)
	ui.done = make(chan struct{})
	ui.Analyzer = &testanalyze.MockedAnalyzer{}
	ui.currentDir = parentDir
	ui.topDir = parentDir
	ui.topDirPath = parentDir.GetPath()
	ui.showDir()
	ui.table.Select(0, 0)

	ui.archiveName = "sub.tar"
	ui.archiveGzip = false
	ui.archiveItem()

	<-ui.done

	for _, f := range ui.app.(*testapp.MockedApp).GetUpdateDraws() {
		f()
	}

	assert.True(t, ui.pages.HasPage("error"))
}

func TestArchiveItemPathTraversal(t *testing.T) {
	ui, _, _ := createArchiveUI(t)

	ui.archiveName = "../../../tmp/evil.tar"
	ui.archiveGzip = false
	ui.archiveItem()

	assert.True(t, ui.pages.HasPage("error"))
	assert.False(t, ui.pages.HasPage("archiving"))
}

func TestArchiveItemInsideSource(t *testing.T) {
	ui, _, _ := createArchiveUI(t)

	// Try to create the archive inside the directory being archived
	ui.archiveName = "sub/backup.tar"
	ui.archiveGzip = false
	ui.archiveItem()

	assert.True(t, ui.pages.HasPage("error"))
	assert.False(t, ui.pages.HasPage("archiving"))
}
