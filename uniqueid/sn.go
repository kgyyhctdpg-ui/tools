package uniqueid

import (
	"fmt"
	"time"

	toolscrypto "github.com/scoming-dev/tools/crypto"
)

// 生成单号
func GenSn(snPrefix string) string {
	suffix, err := toolscrypto.RandomDigits(8)
	if err != nil {
		suffix = fmt.Sprintf("%08d", time.Now().UnixNano()%100000000)
	}
	return fmt.Sprintf("%s%s%s", snPrefix, time.Now().Format("20060102150405"), suffix)
}
