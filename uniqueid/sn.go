package uniqueid

import (
	"fmt"
	"time"

	"github.com/scoming-dev/tools/utils"
)

// 生成单号
func GenSn(snPrefix string) string {
	return fmt.Sprintf("%s%s%s", snPrefix, time.Now().Format("20060102150405"), utils.Krand(8, 0))
}
