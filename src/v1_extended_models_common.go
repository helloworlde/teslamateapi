package main

// ExtendedResponseMeta 描述 v2 响应共享元数据。
type ExtendedResponseMeta struct {
	CarID       int    `json:"car_id,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
	Unit        string `json:"unit,omitempty"`
	GeneratedAt string `json:"generated_at,omitempty"`
	Version     string `json:"version,omitempty"`
}

// ExtendedRange 描述带时区信息的响应时间范围。
type ExtendedRange struct {
	Start    string `json:"start" example:"2026-04-01T00:00:00+08:00"`
	End      string `json:"end" example:"2026-04-30T23:59:59+08:00"`
	Timezone string `json:"timezone" example:"Asia/Shanghai"`
}
