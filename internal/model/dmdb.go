package model

import "encoding/json"

// DMDB API 响应模型（与DMDB服务返回的JSON完全一致）

// DMDBListResponse DMDB列表接口通用响应
type DMDBListResponse struct {
	Total   int               `json:"total"`
	Envs    json.RawMessage   `json:"envs,omitempty"`
	Silos   json.RawMessage   `json:"silos,omitempty"`
	Systems json.RawMessage   `json:"systems,omitempty"`
}

// EnvInfo DMDB环境信息（匹配DMDB Environment结构的关键字段）
type EnvInfo struct {
	Id        string `json:"id"`
	ClassCode string `json:"classCode"`
	Env       string `json:"Env"`       // 环境代码，如 "dev"
	Name      string `json:"name"`      // 环境名称，如 "开发环境"
}

// SiloInfo DMDB竖井信息
type SiloInfo struct {
	Id        string     `json:"id"`
	ClassCode string     `json:"classCode"`
	BizSerial string     `json:"biz_serial"`
	Name      string     `json:"name"`
	BizSystem SystemInfo `json:"biz_system"`
}

// SystemInfo DMDB业务系统信息
type SystemInfo struct {
	Id        string `json:"id"`
	ClassCode string `json:"classCode,omitempty"`
	Code      string `json:"code,omitempty"`
	Name      string `json:"name"`
}

// RefObject DMDB引用对象（如 belong_System）
type RefObject struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

// DeployUnitInfo DMDB部署单元信息（匹配 DeployUnit 核心字段）
type DeployUnitInfo struct {
	Id              string    `json:"id"`
	ClassCode       string    `json:"classCode"`
	BizSerial       string    `json:"biz_serial"`
	Env             string    `json:"Env"`
	System          string    `json:"System"`
	SiloCode        string    `json:"SiloCode"`
	Description     string    `json:"desc,omitempty"`
	DuTypeCode      string    `json:"du_type_code,omitempty"`
	AppName         string    `json:"AppName"`
	ArtifactVersion string    `json:"ArtifactVersion"`
	ArtifactId      string    `json:"ArtifactId"`
	SystemName      string    `json:"SystemName,omitempty"`
	BizSystem       RefObject `json:"belong_System"`
}
