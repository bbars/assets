package utils

import (
	"errors"
	"io"
)

func NewRangeReader(r io.Reader, offset int64, length int64) (rr *rangeReader) {
	rr = &rangeReader{
		offset: offset,
		length: length,
		r:      r,
	}
	return
}

type rangeReader struct {
	offset  int64
	length  int64
	r       io.Reader
	lr      io.Reader
	skipped int64
}

var _ io.Reader = &rangeReader{}

func (rr *rangeReader) Read(p []byte) (n int, err error) {
	if rr.offset < 0 {
		err = errors.New("negative offset")
		return
	}
	if rr.skipped < rr.offset {
		if rs, ok := rr.r.(io.Seeker); ok {
			var initialPos int64
			var resultPos int64
			initialPos, err = rs.Seek(0, io.SeekCurrent)
			if err != nil {
				return
			}
			resultPos, err = rs.Seek(rr.offset, io.SeekCurrent)
			if err != nil {
				if err == io.EOF {
					err = io.ErrUnexpectedEOF
				}
				return
			}
			rr.skipped = resultPos - initialPos
			if rr.skipped < rr.offset {
				err = io.ErrUnexpectedEOF
				return
			}
		} else {
			rr.skipped, err = io.CopyN(io.Discard, rr.r, rr.offset)
			if err != nil {
				if err == io.EOF {
					err = io.ErrUnexpectedEOF
				}
				return
			}
		}
	}
	if rr.lr == nil {
		rr.lr = io.LimitReader(rr.r, rr.length)
	}
	n, err = rr.lr.Read(p)
	return
}

var _ io.Closer = &rangeReader{}

func (rr *rangeReader) Close() error {
	if c, ok := rr.r.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
