package model

import "encoding/json"

// DMDB API 响应模型（与DMDB服务返回的JSON完全一致）

// DMDBListResponse DMDB列表接口通用响应
type DMDBListResponse struct {
	Total   int             `json:"total"`
	Envs    json.RawMessage `json:"envs,omitempty"`
	Silos   json.RawMessage `json:"silos,omitempty"`
	Systems json.RawMessage `json:"systems,omitempty"`
}

// EnvInfo DMDB环境信息（匹配DMDB Environment结构的关键字段）
type EnvInfo struct {
	Id        string `json:"id"`
	ClassCode string `json:"classCode"`
	Env       string `json:"Env"`
	Name      string `json:"name"`
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

// DeployUnitInfo DMDB部署单元信息（匹配DMDB DeployUnit结构）
type DeployUnitInfo struct {
	Id                     string                `json:"id"`
	ClassCode              string                `json:"classCode"`
	BizSerial              string                `json:"biz_serial"`
	Env                    string                `json:"Env"`
	System                 string                `json:"System"`
	SiloCode               string                `json:"SiloCode"`
	Description            string                `json:"desc,omitempty"`
	DuTypeCode             string                `json:"du_type_code,omitempty"`
	DeployType             string                `json:"deploy_type,omitempty"`
	RunAsUser              string                `json:"RunAsUser,omitempty"`
	RunAsGroup             string                `json:"RunAsGroup,omitempty"`
	JvmArgs                string                `json:"JvmArgs,omitempty"`
	AppName                string                `json:"AppName"`
	NodeCount              int                   `json:"NodeCount"`
	MaxPollRecords         int                   `json:"MaxPollRecords"`
	DbStreamEnhancedAudit  string                `json:"dbStreamEnhancedAudit,omitempty"`
	BatchSize              int                   `json:"BatchSize"`
	SystemName             string                `json:"SystemName"`
	KafkaTxTimeoutMs       string                `json:"kafkaTxTimeoutMs"`
	KafkaDeliveryTimeoutMs string                `json:"kafkaDeliveryTimeoutMs"`
	SiloNo                 string                `json:"SiloNo"`
	UseFtp                 string                `json:"UseFtp"`
	RemoteDir              string                `json:"RemoteDir"`
	ExtraConfig            string                `json:"ExtraConfig"`
	MetricPort             int                   `json:"MetricPort"`
	ServiceDatasource      []string              `json:"serviceDatasource"`
	BizSystem              RefObject             `json:"belong_System"`
	ArtifactGroupId        string                `json:"ArtifactGroupId"`
	ArtifactId             string                `json:"ArtifactId"`
	ArtifactVersion        string                `json:"ArtifactVersion"`
	LogLevel               string                `json:"Loglevel"`
	Servers                []RemoteServer        `json:"Servers"`
	FrameworkDatasource    []DatasourceAppConfig `json:"frameworkDatasource"`
	InitDb                 []InitDbCfg           `json:"initDb"`
	InitDbAuth             []InitDbCfg           `json:"initDbAuth"`
	InitDbFinal            []InitDbCfg           `json:"initDbFinal"`
	ImportData             []InitDbCfg           `json:"ImportData"`
	InitKafka              []InitKafka           `json:"initKafka"`
}

// DatasourceAppConfig 数据源应用配置
type DatasourceAppConfig struct {
	Id        string `json:"id"`
	DsName    string `json:"dsName"`
	DbName    string `json:"dbName"`
	Schema    string `json:"schema"`
	ReadOnly  string `json:"readOnly"`
	MaxActive string `json:"maxActive"`
	DbArgs    string `json:"dbArgs,omitempty"`
}

// RemoteServer 远程服务器
type RemoteServer struct {
	Id        string `json:"id"`
	ClassCode string `json:"classCode"`
	Ip        string `json:"ip"`
	Name      string `json:"name"`
	Location  string `json:"location"`
}

// InitDbCfg 数据库初始化配置
type InitDbCfg struct {
	Id     string `json:"id"`
	Type   string `json:"type"`
	Source string `json:"source"`
	DbName string `json:"dbName"`
}

// InitKafka Kafka初始化配置
type InitKafka struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Partitions string `json:"partitions"`
	Replicas   string `json:"replicas"`
}

// DevOpsDUListResponse DevOps API list-du 响应
type DevOpsDUListResponse struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    []DevOpsDUItem `json:"data"`
}

// DevOpsDUItem DevOps API返回的部署单元列表项
type DevOpsDUItem struct {
	Code   string `json:"code"`
	Silo   string `json:"silo"`
	System string `json:"system"`
	Repo   string `json:"repo"`
}

// BatchUpdateResult DMDB批量更新结果
type BatchUpdateResult struct {
	Id         string          `json:"id"`
	ClassCode  string          `json:"classCode"`
	Status     string          `json:"status"`
	DeployUnit *DeployUnitInfo `json:"deploy_unit,omitempty"`
}

// BatchUpdateResponse DMDB批量更新接口响应
type BatchUpdateResponse struct {
	Results []BatchUpdateResult `json:"results"`
}

// DUConfigSnapshot 单个部署单元在某个环境中的完整配置快照（用于跨环境对比）
type DUConfigSnapshot struct {
	Env     string            `json:"env"`
	EnvName string            `json:"env_name"`
	Fields  map[string]string `json:"fields"`
}
