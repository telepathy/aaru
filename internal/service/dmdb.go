package service

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"aaru/internal/model"
	"github.com/go-resty/resty/v2"
)

type DMDBClient struct {
	baseURL    string
	devopsURL  string
	client     *resty.Client
}

func NewDMDBClient(baseURL, devopsURL string) *DMDBClient {
	return &DMDBClient{
		baseURL:   baseURL,
		devopsURL: devopsURL,
		client:    resty.New(),
	}
}

func (d *DMDBClient) ping() error {
	resp, err := d.client.R().Get(d.baseURL + "/ping")
	if err != nil {
		return err
	}
	if resp.StatusCode() != 200 {
		return fmt.Errorf("DMDB not reachable, status=%d", resp.StatusCode())
	}
	return nil
}

func (d *DMDBClient) get(path string, result interface{}) error {
	resp, err := d.client.R().SetResult(result).Get(d.baseURL + path)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	if resp.StatusCode() != 200 {
		return fmt.Errorf("GET %s: status=%d", path, resp.StatusCode())
	}
	return nil
}

// ListEnvironments 获取环境列表
func (d *DMDBClient) ListEnvironments() ([]model.EnvInfo, error) {
	var resp model.DMDBListResponse
	if err := d.get("/api/list/env", &resp); err != nil {
		return nil, err
	}
	var envs []model.EnvInfo
	if err := json.Unmarshal(resp.Envs, &envs); err != nil {
		return nil, fmt.Errorf("unmarshal envs: %w", err)
	}
	return envs, nil
}

// ListSilos 获取竖井列表
func (d *DMDBClient) ListSilos() ([]model.SiloInfo, error) {
	var resp model.DMDBListResponse
	if err := d.get("/api/list/silo", &resp); err != nil {
		return nil, err
	}
	var silos []model.SiloInfo
	if err := json.Unmarshal(resp.Silos, &silos); err != nil {
		return nil, fmt.Errorf("unmarshal silos: %w", err)
	}
	return silos, nil
}

// ListSystems 获取业务系统列表
func (d *DMDBClient) ListSystems() ([]model.SystemInfo, error) {
	var resp model.DMDBListResponse
	if err := d.get("/api/list/system", &resp); err != nil {
		return nil, err
	}
	var systems []model.SystemInfo
	if err := json.Unmarshal(resp.Systems, &systems); err != nil {
		return nil, fmt.Errorf("unmarshal systems: %w", err)
	}
	return systems, nil
}

// QueryDeployUnits 查询部署单元
func (d *DMDBClient) QueryDeployUnits(env, system, silo string) ([]model.DeployUnitInfo, error) {
	path := "/api/query-du/" + env
	if system != "" {
		path += "/" + system
	}
	if silo != "" {
		path += "/" + silo
	}
	var dus []model.DeployUnitInfo
	if err := d.get(path, &dus); err != nil {
		return nil, err
	}
	return dus, nil
}

// GetDeployUnitByCode 获取单个部署单元
func (d *DMDBClient) GetDeployUnitByCode(env, code string) (*model.DeployUnitInfo, error) {
	var du model.DeployUnitInfo
	if err := d.get("/api/get-du/"+env+"/"+code, &du); err != nil {
		return nil, err
	}
	if du.BizSerial == "" {
		return nil, fmt.Errorf("deploy unit %s/%s not found", env, code)
	}
	return &du, nil
}

// GetEnvDetail 获取环境详情
func (d *DMDBClient) GetEnvDetail(code string) (*model.EnvInfo, error) {
	var env model.EnvInfo
	if err := d.get("/api/query-env/"+code, &env); err != nil {
		return nil, err
	}
	if env.Env == "" {
		// 尝试从完整结构中提取
		var raw json.RawMessage
		if err := d.get("/api/query-env/"+code, &raw); err != nil {
			return nil, err
		}
		var partial struct {
			Env  string `json:"Env"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(raw, &partial); err == nil {
			env.Env = partial.Env
			env.Name = partial.Name
		}
	}
	if env.Env == "" {
		return nil, fmt.Errorf("env %s not found", code)
	}
	return &env, nil
}

// ListAllDUs 从DevOps API获取所有部署单元列表，支持按竖井和系统筛选
func (d *DMDBClient) ListAllDUs(silo, system string) ([]model.DevOpsDUItem, error) {
	path := "/api/v1/devops/list-du/"
	params := make(map[string]string)
	if silo != "" {
		params["silo"] = silo
	}
	if system != "" {
		params["system"] = system
	}
	var resp model.DevOpsDUListResponse
	if err := d.getFromDevops(path, params, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// getRawDU 获取单个DU的完整原始JSON数据
func (d *DMDBClient) getRawDU(env, code string) (map[string]interface{}, error) {
	var raw map[string]interface{}
	if err := d.get("/api/get-du/"+env+"/"+code, &raw); err != nil {
		return nil, err
	}
	if raw["biz_serial"] == nil || raw["biz_serial"] == "" {
		return nil, fmt.Errorf("deploy unit %s/%s not found", env, code)
	}
	return raw, nil
}

// flattenFields 将map[string]interface{}扁平化为map[string]string，嵌套值序列化为JSON
func flattenFields(raw map[string]interface{}) map[string]string {
	fields := make(map[string]string)
	for k, v := range raw {
		switch val := v.(type) {
		case string:
			fields[k] = val
		case float64:
			fields[k] = fmt.Sprintf("%v", val)
		case bool:
			if val {
				fields[k] = "true"
			} else {
				fields[k] = "false"
			}
		case nil:
			fields[k] = ""
		default:
			// 嵌套结构（数组、对象等）→ JSON字符串
			b, err := json.Marshal(val)
			if err != nil {
				fields[k] = fmt.Sprintf("%v", val)
			} else {
				fields[k] = string(b)
			}
		}
	}
	return fields
}

// CompareDUConfig 获取某个DU在所有DMDB环境中的完整配置，用于跨环境对比
func (d *DMDBClient) CompareDUConfig(duCode string) ([]model.DUConfigSnapshot, error) {
	envs, err := d.ListEnvironments()
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}

	var snapshots []model.DUConfigSnapshot
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, env := range envs {
		wg.Add(1)
		go func(envCode, envName string) {
			defer wg.Done()
			raw, err := d.getRawDU(envCode, duCode)
			if err != nil {
				return
			}
			fields := flattenFields(raw)
			mu.Lock()
			snapshots = append(snapshots, model.DUConfigSnapshot{
				Env:     envCode,
				EnvName: envName,
				Fields:  fields,
			})
			mu.Unlock()
		}(env.Env, env.Name)
	}
	wg.Wait()
	return snapshots, nil
}

func (d *DMDBClient) getFromDevops(path string, params map[string]string, result interface{}) error {
	req := d.client.R().SetResult(result)
	for k, v := range params {
		req.SetQueryParam(k, v)
	}
	resp, err := req.Get(d.devopsURL + path)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	if resp.StatusCode() != 200 {
		return fmt.Errorf("GET %s: status=%d", path, resp.StatusCode())
	}
	return nil
}

// GetAllDeployUnits 查询所有环境的部署单元
func (d *DMDBClient) GetAllDeployUnits() ([]model.DeployUnitInfo, error) {
	envs, err := d.ListEnvironments()
	if err != nil {
		return nil, err
	}
	var mu sync.Mutex
	var all []model.DeployUnitInfo
	var wg sync.WaitGroup
	for _, env := range envs {
		wg.Add(1)
		go func(code string) {
			defer wg.Done()
			dus, err := d.QueryDeployUnits(code, "", "")
			if err != nil {
				log.Printf("query dus for env %s: %v", code, err)
				return
			}
			mu.Lock()
			all = append(all, dus...)
			mu.Unlock()
		}(env.Env)
	}
	wg.Wait()
	return all, nil
}
