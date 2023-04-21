package utils

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRangeReader(t *testing.T) {
	src := "0123456789"
	tests := []struct {
		name    string
		reader  io.Reader
		offset  int64
		length  int64
		wantErr bool
		wantOut string
	}{
		{
			name:   "basic",
			reader: newReaderFromString(src),
			// rng:     Range{From: 5, To: 10},
			offset:  5,
			length:  5,
			wantErr: false,
			wantOut: src[5:10],
		},
		{
			name:   "basic-seeker",
			reader: newReadSeekerFromString(src),
			// rng:     Range{From: 5, To: 10},
			offset:  5,
			length:  5,
			wantErr: false,
			wantOut: src[5:10],
		},
		{
			name:   "head",
			reader: newReaderFromString(src),
			// rng:     Range{From: 0, To: 3},
			offset:  0,
			length:  3,
			wantErr: false,
			wantOut: src[0:3],
		},
		{
			name:   "head-seeker",
			reader: newReadSeekerFromString(src),
			// rng:     Range{From: 0, To: 3},
			offset:  0,
			length:  3,
			wantErr: false,
			wantOut: src[0:3],
		},
		{
			name:   "tail",
			reader: newReaderFromString(src),
			// rng:     Range{From: 0, To: -2},
			offset:  8,
			length:  2,
			wantErr: false,
			wantOut: src[10-2 : 10],
		},
		{
			name:   "tail-seeker",
			reader: newReadSeekerFromString(src),
			// rng:     Range{From: 0, To: -2},
			offset:  8,
			length:  2,
			wantErr: false,
			wantOut: src[10-2 : 10],
		},
		{
			name:   "mid",
			reader: newReaderFromString(src),
			// rng:     Range{From: 2, To: 6},
			offset:  2,
			length:  4,
			wantErr: false,
			wantOut: src[2:6],
		},
		{
			name:   "mid-seeker",
			reader: newReadSeekerFromString(src),
			// rng:     Range{From: 2, To: 6},
			offset:  2,
			length:  4,
			wantErr: false,
			wantOut: src[2:6],
		},
		{
			name:   "underflow",
			reader: newReaderFromString(src),
			// rng:     Range{From: 999, To: 6},
			offset:  999,
			length:  1,
			wantErr: true,
		},
		{
			name:   "underflow-seeker",
			reader: newReadSeekerFromString(src),
			// rng:     Range{From: 999, To: 6},
			offset:  999,
			length:  1,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := NewRangeReader(tt.reader, tt.offset, tt.length)
			out, err := io.ReadAll(rr)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantOut, string(out))
		})
	}
}

func newReaderFromString(s string) io.Reader {
	return &readerMock{
		buf: []byte(s),
	}
}

func newReadSeekerFromString(s string) io.ReadSeeker {
	return &readSeekerMock{
		readerMock{
			buf: []byte(s),
		},
	}
}

type readerMock struct {
	buf []byte
	pos int64 // current reading index
}

var _ io.Reader = &readerMock{}

// Read implements the io.Reader interface.
func (r *readerMock) Read(b []byte) (n int, err error) {
	if r.pos >= int64(len(r.buf)) {
		return 0, io.EOF
	}
	n = copy(b, r.buf[r.pos:])
	r.pos += int64(n)
	return
}

type readSeekerMock struct {
	readerMock
}

var _ io.ReadSeeker = &readSeekerMock{}

// Seek implements the io.Seeker interface.
func (r *readSeekerMock) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekCurrent:
		abs = r.pos + offset
	default:
		return 0, errors.New("readSeekerMock.Seek: invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("readSeekerMock.Seek: negative position")
	} else if abs > int64(len(r.buf))-1 {
		return 0, errors.New("readSeekerMock.Seek: buffer underflow") // important
	}
	r.pos = abs
	return abs, nil
}
