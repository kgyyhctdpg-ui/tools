package utils

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

var num int64

// 随机字符串 kind 0纯数字,1小写字母,2大写字母,3数字、大小写字母
func Krand(size int, kind int) string {
	ikind, kinds, result := kind, [][]int{[]int{10, 48}, []int{26, 97}, []int{26, 65}}, make([]byte, size)
	is_all := kind > 2 || kind < 0
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < size; i++ {
		if is_all { // random ikind
			ikind = rand.Intn(3)
		}
		scope, base := kinds[ikind][0], kinds[ikind][1]
		result[i] = uint8(base + rand.Intn(scope))
	}
	return string(result)
}

// MD5 md5 encryption
func MD5(value string) string {
	m := md5.New()
	m.Write([]byte(value))

	return hex.EncodeToString(m.Sum(nil))
}

// json压缩
func JsonEncode(data interface{}) string {
	byte, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(byte)
}

// CompressJson 压缩json字符串，去除空格和换行
func CompressJson(jsonStr string) string {
	var dst bytes.Buffer
	if err := json.Compact(&dst, []byte(jsonStr)); err != nil {
		return jsonStr
	}
	return dst.String()
}

// 是否是电话
func IsMobile(mobile string) bool {
	if mobile == "" {
		return false
	}
	ok, _ := regexp.MatchString(`^1[3|4|5|6|7|8|9][0-9]\d{8}$`, mobile)
	return ok
}

// 是否是email
func IsEmail(email string) bool {
	if email == "" {
		return false
	}
	ok, _ := regexp.MatchString(`^([a-zA-Z0-9]+[_|\_|\.]?)*[a-zA-Z0-9]+@([a-zA-Z0-9]+[_|\_|\.]?)*[a-zA-Z0-9]+\.[0-9a-zA-Z]{2,3}$`, email)
	return ok
}

// 按字节截取字符串 utf-8不乱码
func Substr(str string, length int64) string {
	bs := []byte(str)[:length]
	bl := 0
	for i := len(bs) - 1; i >= 0; i-- {
		switch {
		case bs[i] >= 0 && bs[i] <= 127:
			return string(bs[:i+1])
		case bs[i] >= 128 && bs[i] <= 191:
			bl++
		case bs[i] >= 192 && bs[i] <= 253:
			cl := 0
			switch {
			case bs[i]&252 == 252:
				cl = 6
			case bs[i]&248 == 248:
				cl = 5
			case bs[i]&240 == 240:
				cl = 4
			case bs[i]&224 == 224:
				cl = 3
			default:
				cl = 2
			}
			if bl+1 == cl {
				return string(bs[:i+cl])
			}
			return string(bs[:i])
		}
	}
	return ""
}

// 数字转大写
func NumberToChineseNum(num int) string {
	chineseMap := []string{"圆整", "十", "百", "千", "万", "十", "百", "千", "亿", "十", "百", "千"}
	chineseNum := []string{"零", "壹", "贰", "叁", "肆", "伍", "陆", "柒", "捌", "玖"}
	listNum := []int{}
	for ; num > 0; num = num / 10 {
		listNum = append(listNum, num%10)
	}
	n := len(listNum)
	chinese := ""
	//注意这里是倒序的
	for i := n - 1; i >= 0; i-- {
		chinese = fmt.Sprintf("%s%s%s", chinese, chineseNum[listNum[i]], chineseMap[i])
	}
	//注意替换顺序
	for {
		copychinese := chinese
		copychinese = strings.Replace(copychinese, "零万", "万", 1)
		copychinese = strings.Replace(copychinese, "零亿", "亿", 1)
		copychinese = strings.Replace(copychinese, "零十", "零", 1)
		copychinese = strings.Replace(copychinese, "零百", "零", 1)
		copychinese = strings.Replace(copychinese, "零千", "零", 1)
		copychinese = strings.Replace(copychinese, "零零", "零", 1)
		copychinese = strings.Replace(copychinese, "零圆", "圆", 1)

		if copychinese == chinese {
			break
		} else {
			chinese = copychinese
		}
	}

	return chinese
}

// 对长度不足n的数字前面补0
func Sup(i int64, n int) string {
	m := fmt.Sprintf("%d", i)
	for len(m) < n {
		m = fmt.Sprintf("0%s", m)
	}
	return m
}

// 加
func Add(a float64, b float64) float64 {
	a_d := decimal.NewFromFloat(a)
	b_d := decimal.NewFromFloat(b)
	return a_d.Add(b_d).InexactFloat64()
}

// 减
func Sub(a float64, b float64) float64 {
	a_d := decimal.NewFromFloat(a)
	b_d := decimal.NewFromFloat(b)
	return a_d.Sub(b_d).InexactFloat64()
}

// 乘
func Mul(a float64, b float64) float64 {
	a_d := decimal.NewFromFloat(a)
	b_d := decimal.NewFromFloat(b)
	return a_d.Mul(b_d).InexactFloat64()
}

// 除
func Div(a float64, b float64) float64 {
	a_d := decimal.NewFromFloat(a)
	b_d := decimal.NewFromFloat(b)
	return a_d.Div(b_d).InexactFloat64()
}

func GetClientIp(ctx context.Context) string {
	var r *http.Request
	remoteIp := r.WithContext(ctx).RemoteAddr

	if ip := r.WithContext(ctx).Header.Get("X-Real-IP"); ip != "" {
		remoteIp = ip
	} else if ip = r.WithContext(ctx).Header.Get("X-Forwarded-For"); ip != "" {
		remoteIp = ip
	} else {
		remoteIp, _, _ = net.SplitHostPort(remoteIp)
	}

	//本地ip
	if remoteIp == "::1" {
		remoteIp = "127.0.0.1"
	}

	return remoteIp
}
