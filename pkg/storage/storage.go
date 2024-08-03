package storage

import (
	"io"

	"github.com/yezzey-gp/yproxy/config"
)

type StorageReader interface {
	CatFileFromStorage(name string, offset int64) (io.ReadCloser, error)
}

type StorageWriter interface {
	PutFileToDest(name string, r io.Reader) error
	PatchFile(name string, r io.ReadSeeker, startOffste int64) error
}

type StorageLister interface {
	ListPath(prefix string) ([]*ObjectInfo, error)
}

type StorageInteractor interface {
	StorageReader
	StorageWriter
	StorageLister
}

func NewStorage(cnf *config.Storage) StorageInteractor {
	if cnf.IsLocal == "True" {
		return &FileStorageInteractor{
			cnf: cnf,
		}
	} else {
		return &S3StorageInteractor{
			pool: NewSessionPool(cnf),
			cnf:  cnf,
		}
	}
}
