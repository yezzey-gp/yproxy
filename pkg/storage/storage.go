package storage

import (
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/yezzey-gp/yproxy/config"
	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

type StorageReader interface {
	CatFileFromStorage(name string) (io.Reader, error)
}

type S3StorageReader struct {
	pool SessionPool
	cnf  *config.Storage
}

func NewStorage(cnf *config.Storage) StorageReader {
	return &S3StorageReader{
		pool: NewSessionPool(cnf),
		cnf:  cnf,
	}
}

func (s *S3StorageReader) CatFileFromStorage(name string) (io.Reader, error) {
	sess, err := s.pool.GetSession()
	if err != nil {
		ylogger.Zero.Err(err).Msg("failed to acquire s3 session")
		return nil, err
	}

	objectPath := name
	input := &s3.GetObjectInput{
		Bucket: &s.cnf.StorageBucket,
		Key:    aws.String(objectPath),
	}

	ylogger.Zero.Debug().Str("key", objectPath).Str("bucket", s.cnf.StorageBucket).Msg("requesting external storage")
	object, err := sess.GetObject(input)
	return object.Body, err
}
