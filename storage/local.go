package storage

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"rsync-os/rsync"
)

type Local struct {
	workDir string // Module + Path
}

func NewLocal(module string, path string, topDir string)  (*Local, error) {
	// First, creates a module folder under topDir
	workDir := filepath.Join(topDir, module, path)
	if err := os.MkdirAll(workDir, os.ModePerm); err != nil {
		return nil, err
	}
	return &Local{workDir: workDir}, nil
}

func (l *Local) Put(fileName string, content io.Reader, fileSize int64, metadata rsync.FileMetadata) (written int64, err error) {
	fpath := filepath.Join(l.workDir, fileName)
	// if the file is a folder, ignores content, just creates a folder under the workDir
	if metadata.Mode.IsDIR() {
		return 0, os.Mkdir(fpath, os.ModePerm)
	}

	if metadata.Mode.IsREG() {
		f, err := os.OpenFile(fpath, os.O_CREATE | os.O_EXCL | os.O_WRONLY, metadata.Mode.Convert())
		if err != nil {
			return -1, err
		}
		defer f.Close()

		// Craete a buffer
		fb := bufio.NewWriter(f)
		defer fb.Flush()

		return io.Copy(fb, content)
	}

	return -2, errors.New("Do not support type " + fileName + metadata.Mode.String())
}

func (l *Local) Delete(fileName string, mode rsync.FileMode) error {
	if mode.IsDIR() {
		return os.RemoveAll(fileName)
	}
	return os.Remove(fileName)
}

func (l *Local) List() (rsync.FileList, error) {
	filelist := make(rsync.FileList, 0, 1 << 16)

	if err := filepath.Walk(l.workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		filelist = append(filelist, rsync.FileInfo{
			Path:  []byte(info.Name()),
			Size:  info.Size(),
			Mtime: int32(info.ModTime().Unix()),	// FIXME
			Mode:  rsync.NewFileMode(info.Mode()),
		})

		return nil
	}); err != nil {
		return filelist, err
	}
	return filelist, nil
}



