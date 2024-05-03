package proc

import (
	"fmt"
	"io"
	"time"

	"github.com/yezzey-gp/yproxy/pkg/storage"
	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

type RestartReader interface {
	io.ReadCloser
	Restart(offsetStart int64) error
}

type YRestartReader struct {
	underlying io.ReadCloser
	s          storage.StorageInteractor
	name       string
}

// Close implements RestartReader.
func (y *YRestartReader) Close() error {
	return y.underlying.Close()
}

// Read implements RestartReader.
func (y *YRestartReader) Read(p []byte) (n int, err error) {
	return y.underlying.Read(p)
}

func NewRestartReader(s storage.StorageInteractor,
	name string) RestartReader {

	return &YRestartReader{
		s:    s,
		name: name,
	}
}

func (y *YRestartReader) Restart(offsetStart int64) error {
	if y.underlying != nil {
		_ = y.underlying.Close()
	}
	ylogger.Zero.Debug().Str("object-path", y.name).Int64("offset", offsetStart).Msg("cat object with offset")
	r, err := y.s.CatFileFromStorage(y.name, offsetStart)
	if err != nil {
		return err
	}

	y.underlying = r

	return nil
}

type YproxyRetryReader struct {
	io.ReadCloser
	underlying RestartReader

	bytesWrite    int64
	retryLimit    int
	needReacquire bool
}

// Close implements io.ReadCloser.
func (y *YproxyRetryReader) Close() error {
	err := y.underlying.Close()
	if err != nil {
		ylogger.Zero.Error().Err(err).Msg("encounter close error")
	}
	return err
}

// Read implements io.ReadCloser.
func (y *YproxyRetryReader) Read(p []byte) (int, error) {
	//fmt.Printf("start read:\n")
	for retry := 0; retry < y.retryLimit; retry++ {
		//fmt.Printf("retry %d:\n", retry)
		if y.needReacquire {

			err := y.underlying.Restart(y.bytesWrite)

			if err != nil {
				// log error and continue.
				// Try to mitigate overload problems with random sleep
				fmt.Printf("some err: %v\n", err)
				ylogger.Zero.Error().Err(err).Int("offset reached", int(y.bytesWrite)).Int("retry count", int(retry)).Msg("failed to reacquire external storage connection, wait and retry")

				time.Sleep(time.Second)
				continue
			}

			y.needReacquire = false
		}

		n, err := y.underlying.Read(p)
		if err == io.EOF {
			return n, err
		}
		if err != nil || n < 0 {
			fmt.Printf("some n: %d or err2: %v\n", n, err)
			ylogger.Zero.Error().Err(err).Int("offset reached", int(y.bytesWrite)).Int("retry count", int(retry)).Msg("encounter read error")

			// what if close failed?
			_ = y.underlying.Close()

			// try to reacquire connection to external storage and continue read
			// from previously reached point

			y.needReacquire = true
			continue
		} else {
			y.bytesWrite += int64(n)

			return n, err
		}
	}
	return -1, fmt.Errorf("failed to unpload within retries")
}

const (
	defaultRetryLimit = 100
)

func NewYRetryReader(r RestartReader) io.ReadCloser {
	return &YproxyRetryReader{
		underlying:    r,
		retryLimit:    defaultRetryLimit,
		bytesWrite:    0,
		needReacquire: true,
	}
}

var _ io.ReadCloser = &YproxyRetryReader{}
