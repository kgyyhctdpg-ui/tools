// Package money provides precise decimal helpers for monetary values.
package money

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// Decimal parses common numeric types into decimal.Decimal.
func Decimal(value any) (decimal.Decimal, error) {
	switch v := value.(type) {
	case decimal.Decimal:
		return v, nil
	case string:
		return decimal.NewFromString(v)
	case int:
		return decimal.NewFromInt(int64(v)), nil
	case int8:
		return decimal.NewFromInt(int64(v)), nil
	case int16:
		return decimal.NewFromInt(int64(v)), nil
	case int32:
		return decimal.NewFromInt(int64(v)), nil
	case int64:
		return decimal.NewFromInt(v), nil
	case uint:
		return decimal.NewFromUint64(uint64(v)), nil
	case uint8:
		return decimal.NewFromUint64(uint64(v)), nil
	case uint16:
		return decimal.NewFromUint64(uint64(v)), nil
	case uint32:
		return decimal.NewFromUint64(uint64(v)), nil
	case uint64:
		return decimal.NewFromUint64(v), nil
	case float32:
		return decimal.NewFromFloat32(v), nil
	case float64:
		return decimal.NewFromFloat(v), nil
	default:
		return decimal.Zero, fmt.Errorf("money: unsupported value type %T", value)
	}
}

// MustDecimal parses value and panics on error.
func MustDecimal(value any) decimal.Decimal {
	d, err := Decimal(value)
	if err != nil {
		panic(err)
	}
	return d
}

// Add returns a+b.
func Add(a, b any) (decimal.Decimal, error) {
	left, right, err := pair(a, b)
	if err != nil {
		return decimal.Zero, err
	}
	return left.Add(right), nil
}

// Sub returns a-b.
func Sub(a, b any) (decimal.Decimal, error) {
	left, right, err := pair(a, b)
	if err != nil {
		return decimal.Zero, err
	}
	return left.Sub(right), nil
}

// Mul returns a*b.
func Mul(a, b any) (decimal.Decimal, error) {
	left, right, err := pair(a, b)
	if err != nil {
		return decimal.Zero, err
	}
	return left.Mul(right), nil
}

// Div returns a/b.
func Div(a, b any) (decimal.Decimal, error) {
	left, right, err := pair(a, b)
	if err != nil {
		return decimal.Zero, err
	}
	if right.IsZero() {
		return decimal.Zero, fmt.Errorf("money: divide by zero")
	}
	return left.Div(right), nil
}

// Round rounds value to places decimal places.
func Round(value any, places int32) (decimal.Decimal, error) {
	d, err := Decimal(value)
	if err != nil {
		return decimal.Zero, err
	}
	return d.Round(places), nil
}

// Format returns value rounded and formatted with fixed decimal places.
func Format(value any, places int32) (string, error) {
	d, err := Round(value, places)
	if err != nil {
		return "", err
	}
	return d.StringFixed(places), nil
}

// YuanToFen converts yuan to fen using half-up rounding to the nearest fen.
func YuanToFen(yuan any) (int64, error) {
	d, err := Decimal(yuan)
	if err != nil {
		return 0, err
	}
	return d.Mul(decimal.NewFromInt(100)).Round(0).IntPart(), nil
}

// FenToYuan converts fen to yuan.
func FenToYuan(fen int64) decimal.Decimal {
	return decimal.NewFromInt(fen).Div(decimal.NewFromInt(100))
}

// ChineseRMB converts a money amount into uppercase Chinese RMB.
func ChineseRMB(value any) (string, error) {
	d, err := Decimal(value)
	if err != nil {
		return "", err
	}
	negative := d.IsNegative()
	if negative {
		d = d.Neg()
	}

	cents := d.Mul(decimal.NewFromInt(100)).Round(0).IntPart()
	intPart := cents / 100
	frac := cents % 100

	prefix := ""
	if negative {
		prefix = "负"
	}

	result := prefix + chineseInteger(intPart) + "元"
	jiao := frac / 10
	fen := frac % 10
	if jiao == 0 && fen == 0 {
		return result + "整", nil
	}
	if jiao > 0 {
		result += chineseDigits[jiao] + "角"
	} else if intPart > 0 && fen > 0 {
		result += "零"
	}
	if fen > 0 {
		result += chineseDigits[fen] + "分"
	}
	return result, nil
}

func pair(a, b any) (decimal.Decimal, decimal.Decimal, error) {
	left, err := Decimal(a)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	right, err := Decimal(b)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	return left, right, nil
}

var chineseDigits = []string{"零", "壹", "贰", "叁", "肆", "伍", "陆", "柒", "捌", "玖"}

var smallUnits = []string{"仟", "佰", "拾", ""}

var bigUnits = []string{"", "万", "亿", "兆"}

func chineseInteger(num int64) string {
	if num == 0 {
		return "零"
	}

	groups := make([]int64, 0)
	for num > 0 {
		groups = append(groups, num%10000)
		num /= 10000
	}

	result := ""
	needZero := false
	for i := len(groups) - 1; i >= 0; i-- {
		group := groups[i]
		if group == 0 {
			needZero = result != ""
			continue
		}
		if needZero || (result != "" && group < 1000) {
			result += "零"
		}
		result += chineseGroup(group) + bigUnits[i]
		needZero = false
	}
	return result
}

func chineseGroup(num int64) string {
	divisors := []int64{1000, 100, 10, 1}
	result := ""
	zero := false
	for i, divisor := range divisors {
		digit := num / divisor
		num %= divisor
		if digit == 0 {
			if result != "" {
				zero = true
			}
			continue
		}
		if zero {
			result += "零"
			zero = false
		}
		result += chineseDigits[digit] + smallUnits[i]
	}
	return result
}
