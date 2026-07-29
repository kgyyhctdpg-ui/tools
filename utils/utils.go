package utils

import (
	"net/http"

	"github.com/scoming-dev/tools/jsonx"
	"github.com/scoming-dev/tools/networkx"
	"github.com/scoming-dev/tools/stringx"
)

// json压缩
func JsonEncode(data interface{}) string {
	return jsonx.MustMarshalString(data)
}

// CompressJson 压缩json字符串，去除空格和换行
func CompressJson(jsonStr string) string {
	return jsonx.Compact(jsonStr)
}

// 按字节截取字符串 utf-8不乱码
func Substr(str string, length int64) string {
	return stringx.SafeByteSubstr(str, int(length))
}

// 对长度不足n的数字前面补0
func Sup(i int64, n int) string {
	return stringx.ZeroPadInt(i, n)
}

func GetClientIp(r *http.Request) string {
	return networkx.ClientIP(r)
}
