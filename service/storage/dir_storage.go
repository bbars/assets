package storage

import (
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const PathChunkLen = 2

//goland:noinspection GoNameStartsWithPackageName
type DirStorage struct {
	Dir       string
	PathDepth uint8
	DirPerm   os.FileMode
	FilePerm  os.FileMode
}

var _ Storage = &DirStorage{}

func (storage *DirStorage) OpenRead(contentHash string) (rc io.ReadCloser, err error) {
	exists, path, err := storage.dig(contentHash, false)
	if err != nil {
		return
	}
	if !exists {
		err = os.ErrNotExist
		return
	}

	rc, err = os.Open(path)
	return
}

func (storage *DirStorage) Write(r io.Reader) (exists bool, contentHash string, size int64, err error) {
	tempPath, contentHash, size, err := storage.storeTemp(r)
	if err != nil {
		err = errors.Wrap(err, "store temporary")
		return
	}
	defer func() {
		if tempPath != "" {
			rmTempErr := os.Remove(tempPath)
			if rmTempErr != nil && err == nil {
				err = errors.Wrapf(rmTempErr, "remove temp file %+q", tempPath)
				return
			}
		}
	}()

	exists, path, err := storage.dig(contentHash, true)
	if err != nil {
		err = errors.Wrapf(err, "prepare persistent storage for contentHash=%s", contentHash)
		return
	}
	if exists {
		return
	}

	err = os.Rename(tempPath, path)
	if err != nil {
		err = errors.Wrapf(err, "move temp file %+q to %+q", tempPath, path)
		return
	}
	tempPath = ""
	return
}

func (storage *DirStorage) Check(contentHash string) (exists bool, err error) {
	exists, _, err = storage.dig(contentHash, false)
	return
}

func (storage *DirStorage) dig(contentHash string, prepare bool) (exists bool, path string, err error) {
	contentHashLen := len([]rune(contentHash))
	if contentHashLen > 512 {
		err = errors.New("contentHash must be shorter than 512 characters")
		return
	}
	if uint8(contentHashLen) < storage.PathDepth*PathChunkLen {
		err = errors.New("contentHash is too short to build full-depth path")
		return
	}
	dir := strings.Builder{}
	dir.Grow(len(contentHash) * 3)
	dir.WriteString(storage.Dir)
	var fi os.FileInfo
	for i := uint8(0); ; i++ {
		fi, err = os.Stat(dir.String())

		if os.IsNotExist(err) {
			if i == 0 {
				err = fmt.Errorf("root directory %+q does not exist", dir)
			}
			err = nil
			for ; i < storage.PathDepth; i++ {
				dir.WriteRune(filepath.Separator)
				dir.WriteString(contentHash[i*PathChunkLen : i*PathChunkLen+PathChunkLen])
			}
			break
		}
		if !fi.IsDir() {
			err = fmt.Errorf("file %+q is not a directory", dir)
			return
		}

		path = filepath.Join(dir.String(), contentHash)
		fi, err = os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				err = nil
				if i < storage.PathDepth {
					dir.WriteRune(filepath.Separator)
					dir.WriteString(contentHash[i*PathChunkLen : i*PathChunkLen+PathChunkLen])
					continue
				}
			}
			break
		}
		if fi.IsDir() {
			err = fmt.Errorf("file %+q is a directory", path)
			return
		}

		exists = true
		break
	}

	path = filepath.Join(dir.String(), contentHash)

	if prepare && !exists {
		err = os.MkdirAll(dir.String(), storage.DirPerm)
		if err != nil {
			return
		}
	}

	return
}

func (storage *DirStorage) storeTemp(r io.Reader) (path string, contentHash string, size int64, err error) {
	f, err := os.CreateTemp(storage.Dir, "asset")
	if err != nil {
		err = errors.Wrap(err, "create temp file for asset")
		return
	}
	defer func() {
		closeErr := f.Close()
		if closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	path = f.Name()

	md5Calc := md5.New()
	sha1Calc := sha1.New()

	teeMd5 := io.TeeReader(r, md5Calc)
	teeSha1 := io.TeeReader(teeMd5, sha1Calc)

	_, err = io.Copy(f, teeSha1)
	if err != nil {
		err = errors.Wrap(err, "reading asset data stream and calculating hashes")
		return
	}

	contentHash = fmt.Sprintf(
		"%x%x",
		md5Calc.Sum(nil),
		sha1Calc.Sum(nil),
	)

	fi, err := f.Stat()
	if err != nil {
		err = errors.Wrapf(err, "stat file %+q", path)
		return
	}
	size = fi.Size()

	return
}
