package uniqueid

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

// 生成单号
func GenSn(snPrefix string) string {
	suffix, err := randomDigits(8)
	if err != nil {
		suffix = fmt.Sprintf("%08d", time.Now().UnixNano()%100000000)
	}
	return fmt.Sprintf("%s%s%s", snPrefix, time.Now().Format("20060102150405"), suffix)
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
