package storage

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/yezzey-gp/yproxy/config"
)

type FileStorageInteractor struct {
	StorageInteractor
	cnf *config.Storage
}

func (s *FileStorageInteractor) CatFileFromStorage(name string, offset int64) (io.ReadCloser, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	_, err = io.CopyN(io.Discard, file, offset)
	return file, err
}
func (s *FileStorageInteractor) ListPath(prefix string) ([]*FileInfo, error) {
	var data []*FileInfo
	err := filepath.WalkDir(s.cnf.PathToFolder, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		fileinfo, err := file.Stat()
		if err != nil {
			return err
		}
		data = append(data, &FileInfo{fileinfo.Name(), fileinfo.Size()})
		return nil
	})
	return data, err
}

func (s *FileStorageInteractor) PutFileToDest(name string, r io.Reader) error {
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, r)
	return err
}

func (s *FileStorageInteractor) PatchFile(name string, r io.ReadSeeker, startOffste int64) error {
	//UNUSED TODO
	return fmt.Errorf("TODO")
}
