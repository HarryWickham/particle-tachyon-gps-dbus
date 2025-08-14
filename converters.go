package main

import (
	"fmt"
	"strconv"

	"github.com/godbus/dbus/v5"
)

func ParseFloatVariant(v dbus.Variant) (float64, error) {
	switch val := v.Value().(type) {
	case float64:
		return val, nil
	case string:
		return strconv.ParseFloat(val, 64)
	case int32:
		return float64(val), nil
	default:
		return 0, fmt.Errorf("unexpected type for GNSS value: %T", val)
	}
}

func ToInt8(val any) int8 {
	switch v := val.(type) {
	case int8:
		return v
	case uint8:
		return int8(v)
	case int32:
		return int8(v)
	case uint32:
		return int8(v)
	case int:
		return int8(v)
	case uint:
		return int8(v)
	default:
		return 0
	}
}

func ToInt32(val any) int32 {
	switch v := val.(type) {
	case int8:
		return int32(v)
	case uint8:
		return int32(v)
	case int32:
		return v
	case uint32:
		return int32(v)
	case int:
		return int32(v)
	case uint:
		return int32(v)
	default:
		return 0
	}
}

func ToUint8(val any) uint8 {
	switch v := val.(type) {
	case int8:
		return uint8(v)
	case uint8:
		return v
	case int32:
		return uint8(v)
	case uint32:
		return uint8(v)
	case int:
		return uint8(v)
	case uint:
		return uint8(v)
	default:
		return 0
	}
}
