package common

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"
)

var objectIDSeq atomic.Uint64

// GenSn generates the timestamp based object key used by OSS upload params.
func GenSn(prefix string) string {
	suffix, err := randomDigits(8)
	if err != nil {
		suffix = fmt.Sprintf("%08d", time.Now().UnixNano()%100000000)
	}
	return fmt.Sprintf("%s%s%s", prefix, time.Now().Format("20060102150405"), suffix)
}

// GenNumericID returns a decimal string suitable for generated object names.
func GenNumericID() string {
	seq := objectIDSeq.Add(1) % 1000
	suffix, err := randomDigits(4)
	if err != nil {
		suffix = fmt.Sprintf("%04d", time.Now().UnixNano()%10000)
	}
	return fmt.Sprintf("%d%03d%s", time.Now().UnixNano(), seq, suffix)
}

func randomDigits(n int) (string, error) {
	if n <= 0 {
		return "", nil
	}
	out := make([]byte, n)
	max := big.NewInt(10)
	for i := range out {
		index, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = byte('0' + index.Int64())
	}
	return string(out), nil
}
