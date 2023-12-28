package storage

import (
	"context"
	"io"
	"path"

	"github.com/yezzey-gp/aws-sdk-go/aws"
	"github.com/yezzey-gp/aws-sdk-go/service/s3"
	"github.com/yezzey-gp/aws-sdk-go/service/s3/s3manager"
	"github.com/yezzey-gp/yproxy/config"
	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

type StorageReader interface {
	CatFileFromStorage(name string) (io.Reader, error)
	ListPath(name string) ([]*S3ObjectMeta, error)
}

type StorageWriter interface {
	PutFileToDest(name string, r io.Reader) error
}

type StorageInteractor interface {
	StorageReader
	StorageWriter
}

type S3StorageInteractor struct {
	StorageReader
	StorageWriter

	pool SessionPool
	cnf  *config.Storage
}

func NewStorage(cnf *config.Storage) StorageInteractor {
	return &S3StorageInteractor{
		pool: NewSessionPool(cnf),
		cnf:  cnf,
	}
}

func (s *S3StorageInteractor) CatFileFromStorage(name string) (io.Reader, error) {
	// XXX: fix this
	sess, err := s.pool.GetSession(context.TODO())
	if err != nil {
		ylogger.Zero.Err(err).Msg("failed to acquire s3 session")
		return nil, err
	}

	objectPath := path.Join(s.cnf.StoragePrefix, name)
	input := &s3.GetObjectInput{
		Bucket: &s.cnf.StorageBucket,
		Key:    aws.String(objectPath),
	}

	ylogger.Zero.Debug().Str("key", objectPath).Str("bucket",
		s.cnf.StorageBucket).Msg("requesting external storage")

	object, err := sess.GetObject(input)
	return object.Body, err
}

func (s *S3StorageInteractor) PutFileToDest(name string, r io.Reader) error {
	sess, err := s.pool.GetSession(context.TODO())
	if err != nil {
		ylogger.Zero.Err(err).Msg("failed to acquire s3 session")
		return nil
	}

	objectPath := path.Join(s.cnf.StoragePrefix, name)

	up := s3manager.NewUploaderWithClient(sess, func(uploader *s3manager.Uploader) {
		uploader.PartSize = int64(1 << 24)
		uploader.Concurrency = 1
	})

	_, err = up.Upload(
		&s3manager.UploadInput{

			Bucket:       aws.String(s.cnf.StorageBucket),
			Key:          aws.String(objectPath),
			Body:         r,
			StorageClass: aws.String("STANDARD"),
		},
	)

	return err
}

type S3ObjectMeta struct {
	Path string
	Size int64
}

func (s *S3StorageInteractor) ListPath(prefix string) ([]*S3ObjectMeta, error) {
	sess, err := s.pool.GetSession(context.TODO())
	if err != nil {
		ylogger.Zero.Err(err).Msg("failed to acquire s3 session")
		return nil, err
	}

	prefix = path.Join(s.cnf.StoragePrefix, prefix)
	input := &s3.ListObjectsInput{
		Bucket: &s.cnf.StorageBucket,
		Prefix: aws.String(prefix),
	}

	out, err := sess.ListObjects(input)

	metas := make([]*S3ObjectMeta, 0)
	for _, obj := range out.Contents {
		metas = append(metas, &S3ObjectMeta{
			Path: *obj.Key,
			Size: *obj.Size,
		})
	}

	return metas, nil
}
