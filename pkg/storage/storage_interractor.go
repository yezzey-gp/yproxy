package storage

import "io"

//go:generate mockgen -destination=pkg/mock/storage.go -package=mock
type StorageInteractor interface {
	StorageReader
	StorageWriter
	StorageLister
	StorageMover
}

type StorageReader interface {
	CatFileFromStorage(name string, offset int64) (io.ReadCloser, error)
	ListPath(name string) ([]*S3ObjectMeta, error)
}

type StorageWriter interface {
	PutFileToDest(name string, r io.Reader) error
	PatchFile(name string, r io.ReadSeeker, startOffste int64) error
}

type StorageLister interface {
	ListPath(prefix string) ([]*S3ObjectMeta, error)
}

type StorageMover interface {
	MoveObject(from string, to string) error
	DeleteObject(key string) error
}
