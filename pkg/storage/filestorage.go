package storage

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/yezzey-gp/yproxy/config"
)

// Storage prefix uses as path to folder.
// "/path/to/folder/" + "path/to/file.txt"
type FileStorageInteractor struct {
	StorageInteractor
	cnf *config.Storage
}

func (s *FileStorageInteractor) CatFileFromStorage(name string, offset int64) (io.ReadCloser, error) {
	file, err := os.Open(path.Join(s.cnf.StoragePrefix, name))
	if err != nil {
		return nil, err
	}
	_, err = io.CopyN(io.Discard, file, offset)
	return file, err
}
func (s *FileStorageInteractor) ListPath(prefix string) ([]*ObjectInfo, error) {
	var data []*ObjectInfo
	err := filepath.WalkDir(s.cnf.StoragePrefix+prefix, func(path string, d fs.DirEntry, err error) error {
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
		data = append(data, &ObjectInfo{fileinfo.Name(), fileinfo.Size()})
		return nil
	})
	return data, err
}

func (s *FileStorageInteractor) PutFileToDest(name string, r io.Reader) error {
	file, err := os.Create(path.Join(s.cnf.StoragePrefix, name))
	if err != nil {
		return err
	}
	_, err = io.Copy(file, r)
	return err
}

func (s *FileStorageInteractor) PatchFile(name string, r io.ReadSeeker, startOffset int64) error {
	//UNUSED TODO
	return fmt.Errorf("TODO")
}

func (s *FileStorageInteractor) MoveObject(from string, to string) error {
	return os.Rename(path.Join(s.cnf.StoragePrefix, from), path.Join(s.cnf.StoragePrefix, to))
}

func (s *FileStorageInteractor) DeleteObject(key string) error {
	return os.Remove(path.Join(s.cnf.StoragePrefix, key))
}
