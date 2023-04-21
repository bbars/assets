package utils

import (
	"math/rand"
	"strconv"
	"strings"
	"time"
)

const (
	QidTimeOrigin = 167253120 // 2023-01-01T00:00:00Z
)

var (
	qidRandomBytes = []byte("0123456789" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz")
	qidRandomByteLen = len(qidRandomBytes)
	qidRandom        = rand.NewSource(time.Now().UnixNano())
)

func GenerateQid(len int) string {
	if len < 8 {
		panic("qid len is too small, must be in interval [8, 8192]")
	} else if len > (1 << 13) {
		panic("qid len is too big, must be in interval [8, 8192]")
	}
	t := uint64(time.Now().Unix() - QidTimeOrigin)
	res := strings.Builder{}
	res.Grow(len)
	i, err := res.WriteString(strconv.FormatUint(t, 32))
	if err != nil {
		panic(err)
	}
	for i < len {
		err = res.WriteByte(qidRandomBytes[qidRandom.Int63()%int64(qidRandomByteLen)])
		if err != nil {
			panic(err)
		}
		i++
	}
	return res.String()
}
