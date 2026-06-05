# Aaru 设计文档

## 系统架构

```
┌─────────────────────────────────────────────────────────┐
│  浏览器 (Vanilla JS SPA)                                 │
│  web/js/app.js  web/css/style.css  web/templates/        │
└──────────────────────┬──────────────────────────────────┘
                       │ HTTP / JSON
┌──────────────────────▼──────────────────────────────────┐
│  Gin HTTP Server (cmd/api/main.go)                       │
│  ├─ internal/handler/    路由处理（薄层，委托给 service）  │
│  ├─ internal/middleware/  JWT 认证中间件                  │
│  └─ static file serving (web/)                           │
└──────────────────────┬──────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────┐
│  Service 层                                               │
│  ├─ AuthService        JWT 生成/解析，GitLab OAuth2      │
│  ├─ PermissionService  角色+竖井+环境权限校验             │
│  ├─ ReleaseService     发布生命周期管理                   │
│  ├─ BlueprintService   蓝图 DAG 管理与校验               │
│  └─ DMDBClient         DMDB + Dalaran API 代理            │
└──────────────────────┬──────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────┐
│  Store 层 (GORM)                                         │
│  └─ MySQL                                              │
│  └─ AutoMigrate 所有模型                                 │
└─────────────────────────────────────────────────────────┘
```

## 启动初始化

`initDefaults` 在每次启动时执行，按 `roles` 表是否有数据分两条路径：

### 已有数据（增量修补）

1. 补创缺失的 `viewer` 角色（`*` + `view` 权限）
2. 为 `allowed_silos` 为空的 admin 用户设置 `"*"`
3. 清理废弃的 `approver-*` 环境审批角色

**不会删除或覆盖**已有用户、角色、权限或业务数据。

### 全新安装

创建四个角色及权限：

| 角色 | 权限 |
|------|------|
| admin | deploy, approve, view, manage |
| developer | deploy, view |
| operator | approve, view |
| viewer | view |

### DSN 保障

启动时自动检测 DSN 中是否包含 `parseTime=True`，缺失则自动补充，确保 `time.Time` 字段正常解析。

### 数据迁移

`AutoMigrate` 仅做增量操作（新增表/列），不删除已有数据或修改列类型。

## 数据模型

### User

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| username | string | 唯一，SSO 关联键 |
| email | string | 邮箱 |
| avatar_url | string | 头像 URL |
| gitlab_id | int64 | GitLab 用户 ID（普通索引，允许为空或零值） |
| allowed_silos | string | 可用竖井：`""`/`"*"`/`"silo1,silo2"` |
| allowed_envs | string | 可用环境（仅 operator）：格式同上 |
| roles | []Role | 多对多（user_roles） |

### Role

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| name | string | 唯一角色名 |
| description | string | 描述 |
| permissions | []Permission | 一对多 |

### Permission

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| role_id | uint | 外键 |
| deploy_unit_code | string | DU 代码，`"*"` = 全部 |
| action | string | 操作：`deploy`/`approve`/`view`/`manage` |

### PromotionBlueprint

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| name | string | 唯一名称 |
| description | string | 描述 |

### BlueprintNode

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| blueprint_id | uint | 外键 |
| env_code | string | 环境代码 |
| env_name | string | 环境名称 |
| pos_x, pos_y | int | SVG 布局坐标 |
| gate_type | string | `manual`/`auto`/`api_hook` |
| approve_role_id | *uint | 废弃，已不再使用 |
| webhook_token | string | api_hook 的认证 token |

### BlueprintEdge

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| blueprint_id | uint | 外键 |
| from_node_id | uint | 起始节点 |
| to_node_id | uint | 目标节点 |

### Release

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| title | string | 发布标题 |
| deploy_unit_code | string | DU 代码 |
| deploy_unit_name | string | DU 名称 |
| silo_code | string | 竖井代码（权限校验用） |
| silo_name | string | 竖井名称 |
| system_name | string | 系统名称 |
| version | string | ArtifactVersion |
| blueprint_id | *uint | 外键 |
| changes_json | text | 变更内容 JSON |
| status | string | `draft`/`in_progress`/`completed`/`failed`/`rolled_back` |
| created_by_id | uint | 创建者 |
| stages | []ReleaseStage | 一对多 |

### ReleaseStage

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| release_id | uint | 外键 |
| node_id | *uint | 蓝图节点 ID |
| env_code | string | 环境代码 |
| env_name | string | 环境名称 |
| promotion_order | int | 晋级顺序 |
| gate_type | string | 闸门类型 |
| status | string | 见状态机 |
| approved_by_id | *uint | 审批人 |
| approved_at | *time | 审批时间 |
| comment | string | 审批备注 |

## 状态机

### 发布状态

```
draft ──(StartRelease)──> in_progress ──(全部sink完成)──> completed
                              │
                              ├──(任一stage驳回)──> failed
                              └──(RollbackRelease)──> rolled_back
```

### 阶段状态

```
manual:   pending ──> in_progress ──> approved ──> pushing ──> completed
                            │
                            └──> rejected

auto:     pending ──> in_progress ──> approved ──> pushing ──> completed
           (自动)

api_hook: pending ──> in_progress ──> approved ──> pushing ──> completed
           (webhook)
```

- `pushing` 状态表示正在推送配置到 DMDB，失败时停留在此状态，可通过 `retry-push` 重试

## DAG 晋级逻辑

蓝图使用 Kahn 算法进行拓扑排序：

1. **激活源节点**：无入边的节点在发布开始时自动进入 `in_progress`
2. **激活子节点**：某节点完成后，检查其所有子节点——若某子节点的所有父节点均已完成，则激活该子节点
3. **完成判定**：所有 sink 节点（无出边）完成时，发布状态变为 `completed`

## 权限校验流程

```
操作请求
  │
  ├─ admin 角色? ──> 跳过所有检查，直接放行
  │
  ├─ deploy 操作? ──> CanDeploy(userID, siloCode)
  │     ├─ 角色有 deploy 权限?  ──> 否: 拒绝
  │     └─ allowed_silos 包含 silo? ──> 否: 拒绝
  │
  ├─ approve 操作? ──> CanApprove(userID, siloCode, envCode)
  │     ├─ 角色有 approve 权限? ──> 否: 拒绝
  │     ├─ allowed_silos 包含 silo? ──> 否: 拒绝
  │     └─ allowed_envs 包含 env? ──> 否: 拒绝
  │
  └─ manage 操作? ──> CanAction(userID, "manage") ──> 角色有权限即放行
```

## API 列表

### 公开接口（无需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/init` | 系统初始化（仅无用户时可用） |
| GET | `/auth/login` | 登录页 |
| POST | `/auth/callback` | Mock 登录回调 |
| GET | `/auth/gitlab/callback` | GitLab OAuth2 回调 |
| GET/POST | `/api/hooks/promote/:stageId` | Webhook 晋级（token 认证） |

### 认证接口

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | `/api/current-user` | 当前用户信息 | 登录 |
| POST | `/api/logout` | 退出登录 | 登录 |

### DMDB 数据

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/environments` | 环境列表 |
| GET | `/api/silos` | 竖井列表 |
| GET | `/api/systems` | 系统列表 |
| GET | `/api/deploy-units` | 查询 DU |
| GET | `/api/deploy-units/:code` | DU 详情 |
| GET | `/api/deploy-units/:code/compare` | DU 跨环境对比 |
| GET | `/api/du-list` | 全量 DU 列表（DevOps） |

### 发布管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | `/api/releases` | 创建发布 | deploy + silo |
| POST | `/api/batch-releases` | 批量创建发布 | deploy + silo |
| GET | `/api/releases` | 发布列表 | 登录 |
| GET | `/api/releases/:id` | 发布详情 | 登录 |
| POST | `/api/releases/:id/start` | 开始发布 | deploy + silo |
| POST | `/api/releases/:id/rollback` | 回滚 | manage |

### 阶段操作

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | `/api/stages/:id/approve` | 审批通过 | approve + silo + env |
| POST | `/api/stages/:id/reject` | 驳回 | approve + silo + env |
| POST | `/api/stages/:id/promote` | 手动推进 | deploy + silo |
| POST | `/api/stages/:id/retry-push` | 重试推送 | deploy + silo |
| GET | `/api/approvals/pending` | 待审批列表 | approve（按权限过滤） |

### 蓝图管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | `/api/blueprints` | 蓝图列表 | 登录 |
| POST | `/api/blueprints` | 创建蓝图 | 登录 |
| GET | `/api/blueprints/:id` | 蓝图详情 | 登录 |
| PUT | `/api/blueprints/:id` | 更新蓝图 | 登录 |
| DELETE | `/api/blueprints/:id` | 删除蓝图 | 登录 |

### 管理接口（admin only）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/init` | 系统初始化 |
| GET | `/api/admin/users` | 用户列表 |
| POST | `/api/admin/users/batch` | 批量创建用户 |
| PUT | `/api/admin/users/:id/roles` | 设置用户角色 |
| PUT | `/api/admin/users/:id/access` | 设置用户竖井/环境权限 |
| GET | `/api/admin/roles` | 角色列表 |
| GET | `/api/admin/roles/:id` | 角色详情 |
| POST | `/api/admin/roles` | 创建角色 |
| PUT | `/api/admin/roles/:id/permissions` | 设置角色权限 |

## 外部系统集成

### DMDB — 配置管理数据库

DMDB 维护各环境的部署单元配置数据。Aaru 通过 DMDB API 读取配置、对比差异、推送变更。

| Aaru 方法 | 上游接口 | 说明 |
|-----------|---------|------|
| ListEnvironments | `GET /api/list/env` | 环境列表 |
| ListSilos | `GET /api/list/silo` | 竖井列表 |
| ListSystems | `GET /api/list/system` | 系统列表 |
| QueryDeployUnits | `GET /api/query-du/{env}` | 按环境查询 DU |
| GetDeployUnitByCode | `GET /api/get-du/{env}/{code}` | 获取单个 DU |
| CompareDUConfig | 并发获取所有环境 DU → 扁平化 → 对比 | 跨环境配置对比 |
| UpdateDeployUnit | `POST /api/du-batch-update/{env}` | 批量更新（需 token） |

配置方式：`aaru.yaml` 或环境变量（`AARU_DMDB_URL`、`AARU_DMDB_TOKEN`）

#### 配置推送流程

发布审批通过后，`applyChanges` 构建更新请求：

```json
[{"id": "du-id", "classCode": "du-class", "ArtifactVersion": "v2.3.0", ...}]
```

调用 `POST /api/du-batch-update/{env}` 批量更新接口。响应中每个 item 有 `status` 字段（`updated`/`not_found`/`forbidden` 等）。

#### initDb URL Tag 同步

当 `ArtifactVersion` 变更时，以下字段中的 git blob URL（`/blob/TAG/path`）自动替换为新版本 tag：

- `initDb` — 数据库初始化脚本
- `initDbAuth` — 认证库初始化脚本
- `initDbFinal` — 最终初始化脚本
- `ImportData` — 数据导入配置

前端在预览和提交时预计算替换结果，后端存储后推送至 DMDB。

### Dalaran — 产品树与部署单元管理

Dalaran（端口 8733）维护产品树结构（竖井 → 系统 → 部署单元），提供全局 DU 视图。

| Aaru 方法 | 上游接口 | 说明 |
|-----------|---------|------|
| ListAllDUs | `GET /api/v1/devops/list-du/?silo=&system=` | 全量 DU 列表，支持筛选 |

配置方式：`aaru.yaml` 或环境变量 `AARU_DEVOPS_URL`

#### Dalaran 数据模型

```go
type DevOpsDUItem struct {
    Code   string // DU 唯一编码
    Silo   string // 所属竖井
    System string // 所属系统
    Repo   string // 代码仓库地址
}
```

#### 在 Aaru 中的用途

- **部署单元浏览页**：从 Dalaran 获取 DU 列表（而非逐环境查询 DMDB），支持按竖井/系统筛选
- **权限配置**：竖井下拉选项从 Dalaran 的 DU 列表提取唯一 `silo` 值
- **批量发布**：多 DU 选择时使用 Dalaran 列表作为数据源

#### 与 DMDB 的关系

Dalaran 提供产品树视角（哪些 DU 存在），DMDB 提供环境配置视角（某个 DU 在某环境的具体配置）。Aaru 从 Dalaran 获取 DU 列表，从 DMDB 获取/更新具体配置。

## 认证

### JWT

- 签名算法：HS256
- 有效期：24 小时
- Claims：`user_id`, `username`, `exp`, `iat`
- 传递方式：Cookie `token` 或 Header `Authorization: Bearer <token>`

### GitLab OAuth2

1. 浏览器跳转 `GitLab /oauth/authorize`
2. 用户授权后回调 `/auth/gitlab/callback?code=xxx`
3. 后端用 code 换 token，再用 token 获取用户信息（自动跳过 TLS 证书验证，支持自签名 GitLab）
4. 按 GitLab 返回的 `username` 在 Aaru 中匹配已有用户：
   - **已存在** → 直接登录，保留原有角色/权限，更新 GitLab ID、头像、邮箱
   - **不存在** → 自动创建，分配 `viewer` 角色
5. 生成 JWT，设为 httpOnly Cookie，重定向到首页

管理员可通过"批量导入"预先创建用户并分配角色，SSO 用户登录时自动继承。

## 前端架构

单文件 SPA (`web/js/app.js`)，无框架无构建工具：

- **路由**：`loadPage(page, param)` 基于 switch-case
- **状态**：全局变量（`currentUser`, `crStep`, `crChanges` 等）
- **DAG 编辑器**：SVG + vanilla JS，支持拖拽、Bezier 曲线、Kahn 自动布局
- **颜色编码**：`crAssignValueColors` 为相同值分配相同背景色
- **事件委托**：`.diff-field-link` 全局点击打开 diff 模态框
