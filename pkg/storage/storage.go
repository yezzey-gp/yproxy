package storage

import (
	"fmt"
	"io"

	"github.com/yezzey-gp/yproxy/config"
)

type StorageReader interface {
	CatFileFromStorage(name string, offset int64) (io.ReadCloser, error)
}

type StorageWriter interface {
	PutFileToDest(name string, r io.Reader) error
	PatchFile(name string, r io.ReadSeeker, startOffset int64) error
}

type StorageLister interface {
	ListPath(prefix string) ([]*ObjectInfo, error)
}

type StorageMover interface {
	MoveObject(from string, to string) error
	DeleteObject(key string) error
}
type StorageInteractor interface {
	StorageReader
	StorageWriter
	StorageLister
	StorageMover
}

func NewStorage(cnf *config.Storage) (StorageInteractor, error) {
	switch cnf.StorageType {
	case "fs":
		return &FileStorageInteractor{
			cnf: cnf,
		}, nil
	case "s3":
		return &S3StorageInteractor{
			pool: NewSessionPool(cnf),
			cnf:  cnf,
		}, nil
	default:
		return nil, fmt.Errorf("wrong storage type " + cnf.StorageType)
	}
}
