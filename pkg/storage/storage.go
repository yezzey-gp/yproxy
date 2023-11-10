package storage

import (
	"io"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
)

type SessionPool struct {
}

type StorageReader interface {
}

type S3StorageReader struct {
}

func NewStorage(bucket, name string) StorageReader {
	sess, err := createSession(bucket, settings)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new session")
	}
	client := s3.New(sess)

	return &S3StorageReader{}
}

func (s *S3StorageReader) CatFileFromStorage(name string) io.Reader {

	return nil
}
