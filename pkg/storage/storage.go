package storage

import (
	"context"
	"fmt"
	"io"
	"path"

	"github.com/yezzey-gp/aws-sdk-go/aws"
	"github.com/yezzey-gp/aws-sdk-go/service/s3"
	"github.com/yezzey-gp/aws-sdk-go/service/s3/s3manager"
	"github.com/yezzey-gp/yproxy/config"
	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

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

type StorageInteractor interface {
	StorageReader
	StorageWriter
	StorageLister
}

type S3StorageInteractor struct {
	StorageInteractor

	pool SessionPool

	cnf *config.Storage
}

func NewStorage(cnf *config.Storage) StorageInteractor {
	return &S3StorageInteractor{
		pool: NewSessionPool(cnf),
		cnf:  cnf,
	}
}

func (s *S3StorageInteractor) CatFileFromStorage(name string, offset int64) (io.ReadCloser, error) {
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
		Range:  aws.String(fmt.Sprintf("bytes=%d-", offset)),
	}

	ylogger.Zero.Debug().Str("key", objectPath).Int64("offset", offset).Str("bucket",
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

func (s *S3StorageInteractor) PatchFile(name string, r io.ReadSeeker, startOffste int64) error {
	sess, err := s.pool.GetSession(context.TODO())
	if err != nil {
		ylogger.Zero.Err(err).Msg("failed to acquire s3 session")
		return nil
	}

	objectPath := path.Join(s.cnf.StoragePrefix, name)

	input := &s3.PatchObjectInput{
		Bucket:       &s.cnf.StorageBucket,
		Key:          aws.String(objectPath),
		Body:         r,
		ContentRange: aws.String(fmt.Sprintf("bytes %d-18446744073709551615", startOffste)),
	}

	_, err = sess.PatchObject(input)

	ylogger.Zero.Debug().Str("key", objectPath).Str("bucket",
		s.cnf.StorageBucket).Msg("modifying file in external storage")

	return err
}

type S3ObjectMeta struct {
	Path string
	Size int64
}

func (s *S3StorageInteractor) ListPath(prefix string) ([]*S3ObjectMeta, error) {
	fmt.Printf("list 1\n")
	sess, err := s.pool.GetSession(context.TODO())
	if err != nil {
		ylogger.Zero.Err(err).Msg("failed to acquire s3 session")
		return nil, err
	}
	fmt.Printf("list 2\n")

	prefix = path.Join(s.cnf.StoragePrefix, prefix)
	input := &s3.ListObjectsInput{
		Bucket: &s.cnf.StorageBucket,
		Prefix: aws.String(prefix),
	}
	fmt.Printf("list 3\n")

	out, err := sess.ListObjects(input)
	fmt.Printf("list 4\n")

	metas := make([]*S3ObjectMeta, 0)
	for _, obj := range out.Contents {
		metas = append(metas, &S3ObjectMeta{
			Path: *obj.Key,
			Size: *obj.Size,
		})
	}
	fmt.Printf("list 5\n")

	return metas, nil
}
