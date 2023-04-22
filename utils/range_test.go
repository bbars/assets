package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRange(t *testing.T) {
	var size int64 = 10
	tests := []struct {
		name         string
		rangeHeader  string
		wantParseErr bool
		wantRange    Range
		wantNormErr  bool
		wantNormFrom int64
		wantNormTo   int64
		wantLength   int64
		wantHeader   string
	}{
		{
			name:         "normal 3-8",
			rangeHeader:  "bytes=3-8",
			wantParseErr: false,
			wantRange:    Range{3, 9},
			wantNormErr:  false,
			wantNormFrom: 3,
			wantNormTo:   9,
			wantHeader:   "bytes=3-8/10",
		},
		{
			name:         "normal 2-3",
			rangeHeader:  "bytes=2-3",
			wantParseErr: false,
			wantRange:    Range{2, 4},
			wantNormErr:  false,
			wantNormFrom: 2,
			wantNormTo:   4,
			wantHeader:   "bytes=2-3/10",
		},
		{
			name:         "auto offset",
			rangeHeader:  "bytes=-4",
			wantParseErr: false,
			wantRange:    Range{0, -4},
			wantNormErr:  false,
			wantNormFrom: 6,
			wantNormTo:   10,
			wantHeader:   "bytes=6-9/10",
		},
		{
			name:         "auto length",
			rangeHeader:  "bytes=7-",
			wantParseErr: false,
			wantRange:    Range{7, 0},
			wantNormErr:  false,
			wantNormFrom: 7,
			wantNormTo:   10,
			wantHeader:   "bytes=7-9/10",
		},
		{
			name:         "auto length full",
			rangeHeader:  "bytes=0-",
			wantParseErr: false,
			wantRange:    Range{0, 0},
			wantNormErr:  false,
			wantNormFrom: 0,
			wantNormTo:   10,
			wantHeader:   "bytes=0-9/10",
		},
		{
			name:         "full",
			rangeHeader:  "bytes=0-9",
			wantParseErr: false,
			wantRange:    Range{0, 10},
			wantNormErr:  false,
			wantNormFrom: 0,
			wantNormTo:   10,
			wantHeader:   "bytes=0-9/10",
		},
		{
			name:         "length overflow",
			rangeHeader:  "bytes=0-999",
			wantParseErr: false,
			wantRange:    Range{0, 1000},
			wantNormErr:  false,
			wantNormFrom: 0,
			wantNormTo:   10,
			wantHeader:   "bytes=0-9/10",
		},
		{
			name:         "negative offset",
			rangeHeader:  "bytes=-999",
			wantParseErr: false,
			wantRange:    Range{0, -999},
			wantNormErr:  false,
			wantNormFrom: 0,
			wantNormTo:   10,
			wantHeader:   "bytes=0-9/10",
		},
		{
			name:         "overflow-overlap",
			rangeHeader:  "bytes=8-15",
			wantParseErr: false,
			wantRange:    Range{8, 16},
			wantNormFrom: 8,
			wantNormTo:   10,
			wantHeader:   "bytes=8-9/10",
		},
		{
			name:         "offset overflow",
			rangeHeader:  "bytes=10-",
			wantParseErr: false,
			wantRange:    Range{10, 0},
			wantNormErr:  true,
		},
		{
			name:         "swapped",
			rangeHeader:  "bytes=8-3",
			wantParseErr: true,
		},
		{
			name:         "undefined",
			rangeHeader:  "bytes=-",
			wantParseErr: true,
		},
		{
			name:         "unexpected characters",
			rangeHeader:  "bytes=6--9",
			wantParseErr: true,
		},
		{
			name:         "broken header",
			rangeHeader:  "ololo",
			wantParseErr: true,
		},
		{
			name:         "empty header",
			rangeHeader:  "",
			wantParseErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRange, gotParseErr := ParseHttpRangeHeader(tt.rangeHeader)
			if tt.wantParseErr {
				assert.Error(t, gotParseErr)
				return
			}
			assert.NoError(t, gotParseErr)
			assert.Equal(t, tt.wantRange, *gotRange)

			gotNormErr := gotRange.Normalize(size)
			if tt.wantNormErr {
				assert.Error(t, gotNormErr)
				return
			}
			assert.NoError(t, gotNormErr)

			assert.Equal(t, tt.wantHeader, gotRange.HttpHeader(size))
		})
	}
}
