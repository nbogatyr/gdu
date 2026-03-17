package archive

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
)

// CreateTar creates a .tar archive of the given source path at destPath.
// The source can be a file or directory.
func CreateTar(sourcePath, destPath string) error {
	return createArchive(sourcePath, destPath, false)
}

// CreateTarGz creates a gzip-compressed .tar.gz archive of the given source path at destPath.
// The source can be a file or directory.
func CreateTarGz(sourcePath, destPath string) error {
	return createArchive(sourcePath, destPath, true)
}

func createArchive(sourcePath, destPath string, useGzip bool) error {
	outFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	var tw *tar.Writer
	if useGzip {
		gw := gzip.NewWriter(outFile)
		defer gw.Close()
		tw = tar.NewWriter(gw)
	} else {
		tw = tar.NewWriter(outFile)
	}
	defer tw.Close()

	sourceInfo, err := os.Lstat(sourcePath)
	if err != nil {
		return err
	}

	// Use the parent of sourcePath as the base for relative paths,
	// so the archive contains the top-level directory name.
	baseDir := filepath.Dir(sourcePath)

	if !sourceInfo.IsDir() {
		return addFileToTar(tw, sourcePath, baseDir)
	}

	return filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return addFileToTar(tw, path, baseDir)
	})
}

func addFileToTar(tw *tar.Writer, path, baseDir string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}

	relPath, err := filepath.Rel(baseDir, path)
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = relPath

	if info.Mode()&os.ModeSymlink != 0 {
		link, err := os.Readlink(path)
		if err != nil {
			return err
		}
		header.Linkname = link
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	if info.IsDir() || !info.Mode().IsRegular() {
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(tw, f)
	return err
}
