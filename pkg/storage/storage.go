package storage

import (
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/yezzey-gp/yproxy/config"
)

type StorageReader interface {
	CatFileFromStorage(name string) (io.Reader, error)
}

type S3StorageReader struct {
	pool SessionPool
	cnf  config.Storage
}

func NewStorage() StorageReader {
	return &S3StorageReader{}
}

func (s *S3StorageReader) CatFileFromStorage(name string) (io.Reader, error) {
	sess := s.pool.GetSession()
	objectPath := name
	input := &s3.GetObjectInput{
		Bucket: &s.cnf.StorageBucket,
		Key:    aws.String(objectPath),
	}

	object, err := sess.GetObject(input)
	return object.Body, err
}
