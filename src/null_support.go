package main

// 参考实现：
//   https://gist.github.com/rsudip90/022c4ef5d98130a224c9239e0a1ab397

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// NullInt64 是 sql.NullInt64 的 JSON 包装类型。
type NullInt64 struct {
	sql.NullInt64
}

// MarshalJSON 将 NullInt64 序列化为 JSON。
func (ni *NullInt64) MarshalJSON() ([]byte, error) {
	if !ni.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(ni.Int64)
}

// NullBool 是 sql.NullBool 的 JSON 包装类型。
type NullBool struct {
	sql.NullBool
}

// MarshalJSON 将 NullBool 序列化为 JSON。
func (nb *NullBool) MarshalJSON() ([]byte, error) {
	if !nb.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(nb.Bool)
}

// NullFloat64 是 sql.NullFloat64 的 JSON 包装类型。
type NullFloat64 struct {
	sql.NullFloat64
}

// MarshalJSON 将 NullFloat64 序列化为 JSON。
func (nf *NullFloat64) MarshalJSON() ([]byte, error) {
	if !nf.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(nf.Float64)
}

// NullString 将 SQL NULL 字符串扫描为空字符串。
type NullString string

// Scan 将数据库值转换为 NullString。
func (s *NullString) Scan(value interface{}) error {
	if value == nil {
		*s = ""
		return nil
	}
	strVal, ok := value.(string)
	if !ok {
		return errors.New("value is not a string")
	}
	*s = NullString(strVal)
	return nil
}

// Value 将 NullString 转换为数据库驱动值。
func (s NullString) Value() (driver.Value, error) {
	if len(s) == 0 { // 为空字符串时写入 SQL NULL。
		return nil, nil
	}
	return string(s), nil
}
