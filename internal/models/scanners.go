package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

func (ct CredentialTypes) Value() (driver.Value, error) {
	if ct == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(ct) // returns []byte
}

func (ct *CredentialTypes) Scan(src interface{}) error {
	if src == nil {
		*ct = nil
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("unsupported type %T for CredentialTypes", src)
	}
	return json.Unmarshal(data, ct)
}

// Implement Valuer/Scanner for ExecutionConfig
func (ec ExecutionConfigs) Value() (driver.Value, error) {
	return json.Marshal(ec)
}

func (ec *ExecutionConfigs) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal ExecutionConfigs: %v", value)
	}
	return json.Unmarshal(bytes, ec)
}

// Implement Valuer/Scanner for ExecutionConfig
func (ec ExecutionConfig) Value() (driver.Value, error) {
	return json.Marshal(ec)
}

func (ec *ExecutionConfig) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal ExecutionConfig: %v", value)
	}
	return json.Unmarshal(bytes, ec)
}

func (m JSONMap) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m *JSONMap) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONMap value: %v", value)
	}
	return json.Unmarshal(bytes, m)
}
