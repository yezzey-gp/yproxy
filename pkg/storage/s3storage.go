package storage

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/yezzey-gp/aws-sdk-go/aws"
	"github.com/yezzey-gp/aws-sdk-go/service/s3"
	"github.com/yezzey-gp/aws-sdk-go/service/s3/s3manager"
	"github.com/yezzey-gp/yproxy/config"
	"github.com/yezzey-gp/yproxy/pkg/message"
	"github.com/yezzey-gp/yproxy/pkg/object"
	"github.com/yezzey-gp/yproxy/pkg/tablespace"
	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

type S3StorageInteractor struct {
	StorageInteractor

	pool SessionPool

	cnf *config.Storage

	bucketMap map[string]string
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

func (s *S3StorageInteractor) PutFileToDest(name string, r io.Reader, settings []message.PutSettings) error {
	sess, err := s.pool.GetSession(context.TODO())
	if err != nil {
		ylogger.Zero.Err(err).Msg("failed to acquire s3 session")
		return err
	}

	objectPath := path.Join(s.cnf.StoragePrefix, name)

	up := s3manager.NewUploaderWithClient(sess, func(uploader *s3manager.Uploader) {
		uploader.PartSize = int64(1 << 24)
		uploader.Concurrency = 1
	})

	storageClass := ResolveStorageSetting(settings, message.StorageClassSetting, "STANDARD")
	tableSpace := ResolveStorageSetting(settings, message.TableSpaceSetting, tablespace.DefaultTableSpace)

	bucket, ok := s.bucketMap[tableSpace]
	if !ok {
		ylogger.Zero.Err(err).Msg(fmt.Sprintf("failed to match tablespace %s to s3 bucket.", tableSpace))
		return err
	}

	_, err = up.Upload(
		&s3manager.UploadInput{
			Bucket:       aws.String(bucket),
			Key:          aws.String(objectPath),
			Body:         r,
			StorageClass: aws.String(storageClass),
		},
	)

	return err
}

func (s *S3StorageInteractor) PatchFile(name string, r io.ReadSeeker, startOffset int64) error {
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
		ContentRange: aws.String(fmt.Sprintf("bytes %d-18446744073709551615", startOffset)),
	}

	_, err = sess.PatchObject(input)

	ylogger.Zero.Debug().Str("key", objectPath).Str("bucket",
		s.cnf.StorageBucket).Msg("modifying file in external storage")

	return err
}

func (s *S3StorageInteractor) ListPath(prefix string) ([]*object.ObjectInfo, error) {
	sess, err := s.pool.GetSession(context.TODO())
	if err != nil {
		ylogger.Zero.Err(err).Msg("failed to acquire s3 session")
		return nil, err
	}

	var continuationToken *string
	prefix = path.Join(s.cnf.StoragePrefix, prefix)
	metas := make([]*object.ObjectInfo, 0)

	for {
		input := &s3.ListObjectsV2Input{
			Bucket:            &s.cnf.StorageBucket,
			Prefix:            aws.String(prefix),
			ContinuationToken: continuationToken,
		}

		out, err := sess.ListObjectsV2(input)
		if err != nil {
			fmt.Printf("list error: %v\n", err)
		}

		for _, obj := range out.Contents {
			metas = append(metas, &object.ObjectInfo{
				Path: *obj.Key,
				Size: *obj.Size,
			})
		}

		if !*out.IsTruncated {
			break
		}

		continuationToken = out.NextContinuationToken
	}
	return metas, nil
}

func (s *S3StorageInteractor) DeleteObject(key string) error {
	sess, err := s.pool.GetSession(context.TODO())
	if err != nil {
		ylogger.Zero.Err(err).Msg("failed to acquire s3 session")
		return err
	}
	ylogger.Zero.Debug().Msg("aquired session")

	if !strings.HasPrefix(key, s.cnf.StoragePrefix) {
		key = path.Join(s.cnf.StoragePrefix, key)
	}

	input2 := s3.DeleteObjectInput{
		Bucket: &s.cnf.StorageBucket,
		Key:    aws.String(key),
	}

	_, err = sess.DeleteObject(&input2)
	if err != nil {
		ylogger.Zero.Err(err).Msg("failed to delete old object")
		return err
	}
	ylogger.Zero.Debug().Msg("deleted object")
	return nil
}
