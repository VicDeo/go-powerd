package battery

import (
	"bytes"
	"fmt"
	"log/slog"
)

type HumanReadable interface {
	ToHuman() string
}

type Volt int64

func (v Volt) ToHuman() string      { return fmt.Sprintf("%.3fV", float64(v)/1e6) }
func (v Volt) LogValue() slog.Value { return slog.Float64Value(float64(v) / 1e6) }

type Watt int64

func (w Watt) ToHuman() string      { return fmt.Sprintf("%.3fW", float64(w)/1e6) }
func (w Watt) LogValue() slog.Value { return slog.Float64Value(float64(w) / 1e6) }

type WattHour int64

func (wh WattHour) ToHuman() string      { return fmt.Sprintf("%.3fWh", float64(wh)/1e6) }
func (wh WattHour) LogValue() slog.Value { return slog.Float64Value(float64(wh) / 1e6) }

// parseTo is a universal helper for int based (microunits).
func ParseTo[T ~int | ~int64](raw []byte, dest *T) error {
	cleanRaw := bytes.TrimSpace(raw)
	val, err := atoi64(cleanRaw)
	if err != nil {
		return err
	}
	*dest = T(val)
	return nil
}

func atoi64(b []byte) (int64, error) {
	var res int64
	isNegative := false
	for i := 0; i < len(b); i++ {
		if i == 0 && b[i] == byte('-') {
			isNegative = true
			continue
		}
		if b[i] >= byte('0') && b[i] <= byte('9') {
			res = res*10 + int64(b[i]-'0')
		} else {
			return 0, fmt.Errorf("ascii to int conversion failed. invalid source: %q", b)
		}
	}
	if isNegative {
		res = -res
	}
	return res, nil
}
