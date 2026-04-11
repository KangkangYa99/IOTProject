# IOTProject

基于 **Go** 与 **Gin** 的物联网设备管理后端：用户注册登录与 JWT 会话、设备绑定与列表、传感器数据上报与历史查询、阈值策略（告警与下行控制）、WebSocket 实时推送。持久化使用 **PostgreSQL**（GORM），会话与锁、限流、策略防抖等使用 **Redis**。

---

## 目录

- [系统能力总览](#系统能力总览)
- [技术栈与工程结构](#技术栈与工程结构)
- [统一响应格式](#统一响应格式)
- [鉴权方式](#鉴权方式)
- [HTTP API 说明](#http-api-说明)
- [WebSocket 说明](#websocket-说明)
- [业务规则补充](#业务规则补充)
- [配置与运行](#配置与运行)
- [其他说明](#其他说明)

---

## 系统能力总览

| 模块 | 能力 |
|------|------|
| 用户 | 注册 / 登录（验证码）、JWT、资料与密码、重置密码、头像、登出；登录限流 |
| 设备 | 绑定、解绑、改名、列表；绑定/解绑/改名等会经 Hub 向在线用户推送事件 |
| 设备数据 | 公开 `POST` 上报 JSON；登录后按设备与传感器类型查历史；上报后异步执行策略并可能向用户推 `device_update`、向设备推控制指令 |
| 策略与告警 | 阈值比较（`> < >= <= ==`）；动作类型 `alert` / `control` / `both`；告警入库、未读统计、待处理列表、标记已处理；告警与控制均有约 6 秒 Redis 防抖 |
| WebSocket | 用户端 `/ws`（先发 JWT 鉴权消息）；硬件 `/ws/device`（URL 带 `device_uid` 与 MD5 Token） |

用户表含 `role_id`，JWT 中间件会写入 `userID`、`roleID`。**当前 HTTP 路由中没有单独的管理员专用接口**；若需后台管理，请自行增加基于角色的中间件与路由。

---

## 技术栈与工程结构

**技术栈**

| 类别 | 技术 |
|------|------|
| 语言与框架 | Go 1.25，Gin |
| 数据库 | PostgreSQL，GORM |
| 缓存 | go-redis |
| 实时 | Gorilla WebSocket |
| 安全 | JWT、密码哈希（`pkg/crypto`）、图形验证码 |

**目录（摘要）**

```
cmd/server/main.go          # 入口
internal/router/router.go   # 全部 HTTP/WS 路由
internal/app/app.go         # 依赖组装与启动
internal/transport/http/    # 各模块 Handler
internal/service/           # 业务逻辑（含策略引擎）
internal/repository/postgres/
internal/domain/
internal/websocket/
pkg/response/               # 统一 JSON 结构
static/                     # 静态文件，HTTP 前缀 /static
```

更细的模块说明见 [docs/项目功能说明.md](docs/项目功能说明.md)。

---

## 统一响应格式

成功与失败均返回 JSON，结构如下（定义见 `pkg/response`）：

| 字段 | 类型 | 说明 |
|------|------|------|
| `code` | int | 业务码；成功一般为 `200` |
| `data` | any | 业务数据，失败时多为 `null` |
| `message` | string | 提示文案；成功时常见为 `"success"` |

HTTP 状态码会随业务错误变化（如未登录/Token 无效多为 `401`，限流 `429`，权限/非设备所有者 `403`，参数与业务错误多为 `400`，服务器错误 `500`）。

---

## 鉴权方式

- **需要登录的接口**：请求头携带 `Authorization: Bearer <access_token>`。
- **Token 校验**：JWT 合法且该 Token 必须在 Redis 集合 `auth:tokens:<user_id>` 中（登录时加入，登出时移除）。
- **登录/注册/重置密码**：需配合图形验证码（先取验证码，再提交 `captcha_id` 与 `code`）。

---

## HTTP API 说明

下列路径均相对于服务根地址，例如 `http://localhost:8888`（端口由 `SERVER_PORT` 决定）。

### 1. 用户模块（`/user`）

#### `POST /user/register` — 用户注册

| 项目 | 说明 |
|------|------|
| 鉴权 | 不需要 |
| 功能 | 创建账号；用户名/手机/邮箱防重用 **Redis 分布式锁**；密码强度校验 |

**请求体 JSON**

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `username` | string | 必填，3–50 字符 | 登录名 |
| `password` | string | 必填，6–50 字符 | 需含小写字母与数字，且不能为弱口令 |
| `phone` | string | 必填，11 位 | 手机号 |
| `email` | string | 必填，邮箱格式 | |
| `device_id` | string | 可选 | 与验证码场景绑定 |
| `code` | string | 必填 | 图形验证码内容 |
| `captcha_id` | string | 必填 | 验证码 ID |

**成功 `data`**：`{ "user_id": number, "username": string }`

---

#### `POST /user/login` — 登录

| 项目 | 说明 |
|------|------|
| 鉴权 | 不需要 |
| 功能 | 校验账号状态与密码；签发 JWT；Token 写入 Redis；**限流：每 IP 每分钟 5 次**（可改 `router` 配置） |

**请求体 JSON**

| 字段 | 类型 | 说明 |
|------|------|------|
| `identity` | string | 用户名、手机号或邮箱 |
| `password` | string | 密码 |
| `device_id` | string | 可选，验证码用 |
| `code` | string | 验证码 |
| `captcha_id` | string | 验证码 ID |

**成功 `data`**：`{ "access_token": string, "expires_in": number }`（`expires_in` 为秒，约 3 天）

---

#### `GET /user/GetCaptcha` — 获取图形验证码

| 项目 | 说明 |
|------|------|
| 鉴权 | 不需要 |
| 功能 | 返回 Base64 图片与 `captcha_id`，供注册/登录/重置密码等使用 |

**Query**

| 参数 | 说明 |
|------|------|
| `device_id` | 必填，客户端设备标识 |
| `action` | 可选，场景（如与 Redis 中验证码存储相关） |

**成功 `data`**：`{ "captcha_id": string, "image": string }`（`image` 为 Base64 数据）

---

#### `POST /user/resetpassword` — 忘记密码重置

| 项目 | 说明 |
|------|------|
| 鉴权 | 不需要 |
| 功能 | 校验用户名+手机+邮箱一致后设置新密码 |

**请求体 JSON**

| 字段 | 类型 | 说明 |
|------|------|------|
| `username` | string | 必填 |
| `phone_number` | string | 必填 |
| `email` | string | 必填 |
| `new_password` | string | 必填，6–50 字符 |
| `device_id` | string | 可选 |
| `code` / `captcha_id` | string | 验证码 |

---

#### `POST /user/Logout` — 登出（无 JWT 中间件）

| 项目 | 说明 |
|------|------|
| 鉴权 | 不强制；若请求头带 `Authorization: Bearer <token>` 且 Context 中已有用户，则从 Redis 移除该 Token |
| 功能 | 退出登录 |

---

#### `POST /user/update` — 修改资料（需登录）

| 项目 | 说明 |
|------|------|
| 鉴权 | 需要 Bearer Token |
| 功能 | 更新手机、邮箱等（未传或空表示不修改） |

**请求体 JSON**

| 字段 | 说明 |
|------|------|
| `phone` | 可选，11 位手机号 |
| `email` | 可选，邮箱格式 |

---

#### `POST /user/changepassword` — 登录态下修改密码（需登录）

**请求体 JSON**

| 字段 | 说明 |
|------|------|
| `old_password` | 旧密码 |
| `new_password` | 新密码，至少 6 位 |
| `verify_phone` | 11 位，用于校验身份 |
| `device_id` / `code` / `captcha_id` | 验证码相关 |

---

#### `GET /user/info` — 当前用户信息（需登录）

**成功 `data`**：用户视图，含 `user_id`、`username`、`role_id`、`phone_number`、`avatar_url`、`email`、`created_at` 等（无密码）。

---

#### `POST /user/logout` — 登出（需登录）

与 `Logout` 类似，走 JWT 中间件，可稳定拿到 `userID` 后移除 Token。

---

#### `POST /user/uploadavatar` — 上传头像（需登录）

| 项目 | 说明 |
|------|------|
| 鉴权 | 需要 Bearer Token |
| 功能 | `multipart/form-data`，字段名 `avatar`；限制约 5MB；校验为真实图片且宽高不超过 4096；保存到 `static/uploads/`，并更新用户 `avatar_url` |

**成功 `data`**：`{ "url": string }`（形如 `/static/uploads/xxx.ext`）

---

### 2. 设备模块（`/device`，均需登录）

#### `POST /device/bindDevice` — 绑定设备

**请求体 JSON**

| 字段 | 说明 |
|------|------|
| `device_uid` | 必填，设备唯一标识 |
| `device_name` | 必填，展示名称 |

`user_id` 由服务端从 Token 注入。成功后向该用户 WebSocket 推送类型为 `bind` 的事件（含设备名、状态等）。

---

#### `POST /device/unbindDevice` — 解绑设备

**请求体 JSON**：`device_uid`（必填）。成功后推送 `unbind` 事件。

---

#### `POST /device/updatedevicename` — 修改设备名称

**请求体 JSON**：`device_uid`、`device_name`。成功后推送 `UpdateDeviceName` 事件。

---

#### `GET /device/getDevicesInfo` — 当前用户设备列表

**成功 `data`**：包含设备总数与设备数组（设备 ID、UID、名称、状态、最后在线时间等，见 `domain.Device` / `DeviceInfo`）。

---

### 3. 设备数据模块（`/devicedata`）

#### `POST /devicedata/uploaddata` — 上报传感器数据

| 项目 | 说明 |
|------|------|
| 鉴权 | **不需要**（便于硬件/网关上送） |
| 功能 | 写入时序数据；异步执行策略引擎；若设备已绑定用户，向该用户 WS 推送 `device_update` |

**请求体 JSON（主要字段，与 `domain.DeviceData` 一致）**

| 字段 | 说明 |
|------|------|
| `device_uid` | 设备唯一标识 |
| `temperature` / `humidity` / `light_intensity` / `noise_level` | 数值类传感器 |
| `flame_detected` | 是否检测到火焰 |
| `carbon_monoxide_level` | 一氧化碳 |
| `fan_on` / `light_on` | 布尔状态 |
| `rgb_red` / `rgb_green` / `rgb_blue` | RGB |
| `data_timestamp` | 可选，时间戳 |

策略匹配时，数值类传感器在库中使用的列名包括：`temperature`、`humidity`、`light_intensity`、`noise_level`、`carbon_monoxide_level` 等（布尔型火焰等需策略引擎支持，当前 `extractValue` 主要覆盖数值列）。

---

#### `GET /devicedata/history` — 历史数据查询（需登录）

**Query**

| 参数 | 说明 |
|------|------|
| `device_uid` | 必填 |
| `sensor_type` | 必填，前端别名（见下表） |
| `limit` | 可选，1–100，默认 100 |

**`sensor_type` 别名 → 数据库列**

| 别名 | 含义 |
|------|------|
| `temp` | 温度 |
| `humi` | 湿度 |
| `light` | 光照 |
| `noise` | 噪音 |
| `fire` | 火焰 |
| `Co` | 一氧化碳 |

仅当设备归属当前用户时允许查询。返回为时间序列点列表（`data_timestamp` + `value`）。

---

### 4. 设备策略与告警（`/devicepolicy`，均需登录）

#### `POST /devicepolicy/createpolicy` — 创建策略

**请求体 JSON**

| 字段 | 说明 |
|------|------|
| `device_uid` | 必填 |
| `sensor_type` | 必填，与入库字段一致（如 `temperature`） |
| `operator` | 必填，`>` `<` `>=` `<=` `==` |
| `threshold_value` | 阈值 |
| `action_type` | 必填，`alert` / `control` / `both` |
| `action_target` | 控制目标，如 `fan`、`light` |
| `action_value` | 控制参数（如风扇/灯状态字符串） |
| `action_message` | 告警文案（告警类使用） |

同一设备下相同传感器、运算符、阈值的组合不可重复。成功后向用户推送 `policy_create` 的 WS 事件。

---

#### `GET /devicepolicy/getepolicy` — 查询某设备下策略列表

**Query**：`device_uid`（必填）。校验设备属于当前用户后返回策略数组。

---

#### `POST /devicepolicy/deleteepolicy` — 删除策略

**请求体 JSON**：`PolicyId`（JSON 字段名，注意大小写与前端约定），必填。

成功后推送 `policy_delete` 事件。

---

#### `GET /devicepolicy/getunreadcount` — 未处理告警数量

返回未处理条数（与告警状态字段相关，见库表 `alert_logs`）。

---

#### `GET /devicepolicy/getpendingaler` — 待处理告警列表

返回当前用户的待处理告警记录列表（`AlertLog` 结构：日志 ID、设备、策略、传感器、当前值、阈值、消息、状态等）。

---

#### `PUT /devicepolicy/markaler/:log_id` — 标记某条告警已处理

**路径参数**：`log_id` 为告警日志 ID。校验归属后更新状态；成功后推送 `alert_handled` 事件。

---

## WebSocket 说明

### `GET /ws` — Web / App 用户连接

1. 建立 WebSocket 后，客户端需发送 JSON：**`{ "type": "auth", "token": "<JWT>" }`**（与 HTTP 使用同一 `access_token`）。
2. 服务端校验 JWT 后将连接登记到 Hub，`user_id` 与在线连接绑定，并回复 `{"type":"auth_success"}`。
3. 未鉴权前除获取验证码等少数消息外，业务消息会返回未授权。
4. 支持 **`{ "type": "get_verify_code", ... }`** 拉取验证码（用于部分流程）。
5. 鉴权后可发 **`{ "type": "control", "device_uid": "...", "fan": "...", "light": "...", "r":0,"g":0,"b":0 }`** 等，将风扇/灯光控制转发到已连接的硬件连接。
6. 服务端可能推送：`device_update`（设备数据）、`alarm`（策略告警）、业务事件 JSON（绑定/策略等经 Hub 广播）。

心跳：服务端定期 Ping；客户端可发 `{"type":"heartbeat"}`。

---

### `GET /ws/device` — 硬件设备连接

**Query 参数**

| 参数 | 说明 |
|------|------|
| `device_uid` | 必填 |
| `token` | 必填，设备令牌：`MD5(device_uid + "23wlw4IOT")` 的十六进制字符串（大小写不敏感） |

用于下发 `fan_ctrl`、`light_ctrl` 及策略触发的控制 JSON。生产环境请修改密钥、`CheckOrigin` 与传输安全（WSS）。

---

## 业务规则补充

- **登录限流**：`POST /user/login`，每 IP 每分钟 5 次（`internal/router/router.go`）。
- **策略防抖**：同一策略在约 **6 秒**内重复满足条件时，告警或控制可能被 Redis 冷却键过滤，避免刷屏或频繁继电器动作。
- **控制下发**：`action_target` 为 `fan` 时下发 `{"type":"fan_ctrl","state":...}`；为 `light` 时下发含 RGB 的 `light_ctrl`（策略里可带默认 RGB）。
- **CORS**：当前为允许所有来源，生产建议收紧。
- **静态资源**：`GET /static/...` 映射本地 `./static`（头像 URL 依赖此路径）。

---

## 配置与运行

**环境变量（节选，默认值见 `internal/config/config.go`）**

| 变量 | 说明 |
|------|------|
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` | PostgreSQL |
| `SERVER_PORT` | 服务端口，默认 `8888` |
| `JWT_SECRET` | JWT 密钥 |
| `REDIS_ADDR` | 如 `localhost:6379` |
| `REDIS_PASSWORD` | 可为空 |

**本地启动**

```bash
git clone https://github.com/KangkangYa99/IOTProject.git
cd IOTProject
go run ./cmd/server/main.go
```

**Docker Compose**

```bash
docker compose up -d
```

详见 `docker-compose.yml` 与 `Dockerfile`。数据库需提前建库且表结构与代码一致（可使用仓库内 SQL 备份若提供）。

---

## 其他说明

- 路由与 Handler 源码以 **`internal/router/router.go`** 为准；接口路径存在历史拼写（如 `getepolicy`、`deleteepolicy`、`markaler`），调用时请与表中路径完全一致。
- 更偏「设计说明」的文档：[docs/项目功能说明.md](docs/项目功能说明.md)。
- 许可证：若仓库根目录存在 `LICENSE` 则以该文件为准。
