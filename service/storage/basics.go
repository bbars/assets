package storage

import (
	"io"
)

type Storage interface {
	OpenRead(contentHash string) (rc io.ReadCloser, err error)
	Write(r io.Reader, maxSize int64) (exists bool, contentHash string, size int64, err error)
	Check(contentHash string) (exists bool, err error)
}
