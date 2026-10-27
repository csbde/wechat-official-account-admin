# 微信公众号管理平台 (WeChat Official Account Admin - WOAA Pro)

[![Go Version](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go)](https://golang.org)
[![React](https://img.shields.io/badge/React-18-61DAFB?logo=react)](https://react.dev)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.6-3178C6?logo=typescript)](https://www.typescriptlang.org)
[![Tailwind CSS](https://img.shields.io/badge/Tailwind-3.4-38B2AC?logo=tailwind-css)](https://tailwindcss.com)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

全栈现代化重构的多微信公众号综合管理与运营平台。支持多公众号管理、高仿真手机端自定义菜单设计器、多媒体素材库、智能自动回复、粉丝与标签管理、带参二维码营销、微信开放平台第三方平台接入、多租户行级严格数据隔离以及 OpenTelemetry 链路审计追踪。

前端 SPA 页面通过 Go 标准库 `embed.FS` 完整编译嵌入至单一二进制程序中，实现**零前端静态文件依赖、单文件极速独立分发**。

---

## 🛠️ 重构后技术架构

### 前端技术栈
- **核心框架**: React 18 + TypeScript + Vite
- **UI 组件库**: Tailwind CSS + shadcn/ui (Radix UI + Lucide 图标体系)
- **数据流与缓存**: TanStack Query (React Query v5)
- **高级数据表格**: TanStack Table (v8，支持多选、排序、筛选、分页)
- **全局状态**: Zustand (轻量、高效管理 Token、当前租户、选中的公众号及深浅色主题)
- **表单与强类型校验**: React Hook Form + Zod
- **设计体验**: 微信绿主题系统、支持深色/浅色模式切换、高仿真微信手机底部菜单设计器

### 后端技术栈
- **语言与 Web 框架**: Go 1.26+ + Gin (标准化 RESTful API 架构)
- **数据库 ORM**: GORM (支持 **默认纯 Go SQLite** / **可选 PostgreSQL** 智能双引擎切换)
- **高速缓存**: Redis (微信 AccessToken 分布式缓存与提前续期)
- **分布式任务队列**: Asynq (处理粉丝批量异步同步、素材下载等高并发后台任务)
- **权限与访问控制**: Casbin RBAC (基于角色的动态接口授权)
- **安全认证**: JWT Bearer Token + SSO 单点登录对接支持
- **配置中心**: Viper (支持 YAML 文件与系统环境变量覆盖)
- **结构化日志**: Zap (集成 Gin 请求审计与全局日志输出)
- **数据校验**: Validator (go-playground/validator v10)
- **接口文档**: OpenAPI 3.0 / Swagger UI (`/swagger/index.html`)
- **数据库迁移**: Atlas (声明式 Schema 管理与版本迁移)
- **链路观测**: OpenTelemetry (请求耗时、状态码、TraceID 关联审计日志)

---

## 🌟 核心业务功能

1. **多微信公众号管理 (Accounts)**:
   - 快速增删改查公众号凭据 (AppID, AppSecret, Token, EncodingAESKey)。
   - 顶部一键切换当前激活公众号，全局自动上下文感知。
   - 微信后台服务器配置 (URL 与 Token) 一键复制。
   - 设为默认公众号与启停控制。

2. **微信开放平台第三方平台接入 (WeChat Open Platform Component)**:
   - **免密扫码一键托管**：无需客户公众号提供 AppSecret，客户管理员微信扫码即可将公众号一键委托给本平台代管。
   - **协议端点**：
     - 授权事件接收 URL: `http://<domain>/open/component/notify`
     - 代公众号消息事件接收 URL: `http://<domain>/open/component/$APPID$/callback`
   - **自动化运维**：全自动接收并解析 `component_verify_ticket`，自动维持刷新 `authorizer_access_token`，上层菜单、素材、自动回复、粉丝管理等业务代码完全通用无感透明。

3. **微信自定义菜单设计器 (Menu)**:
   - 高仿真手机视图预览。
   - 一级菜单（最多3个）、二级子菜单（最多5个）可视化增删与交互配置。
   - 动作支持：跳转网页 (`view`)、点击事件/自动回复 (`click`)、微信小程序跳转 (`miniprogram`)、扫码推事件等。
   - 个性化条件菜单（按粉丝标签、性别、手机平台触发）。
   - 从微信线上拉取同步 & 一键发布至微信服务器。

4. **多媒体素材库 (Materials)**:
   - 图片、语音、视频、图文素材管理。
   - 支持本地文件上传并同步生成微信 MediaID。
   - 本地缓存直连预览与一键复制 MediaID。

5. **智能自动回复 (AutoReply)**:
   - 关注公众号后欢迎语回复 (Subscribe)。
   - 收到消息默认应答 (Default)。
   - 关键词智能回复规则（支持多关键词、精准全匹配 / 模糊半匹配）。

6. **粉丝关注者与标签 (Fans & Tags)**:
   - 基于 TanStack Table 的高级表格，支持多维搜索、分页与勾选。
   - 修改粉丝专属备注名。
   - 微信标签管理（创建、删除、重命名）。
   - 批量打标签与批量移除标签。
   - Asynq 后台异步全量同步粉丝。

7. **带参二维码营销 (Qrcode)**:
   - 临时二维码（最长30天时效）与永久二维码生成。
   - 场景值（数字/字符串）追踪扫码渠道与来源。
   - 实时预览与下载二维码图片。

8. **请求与链路追踪审计 (Logs)**:
   - 记录每次 API 请求的 Client IP、耗时、状态码、请求体与响应体。
   - 自动关联 OpenTelemetry TraceID。

9. **多租户严格数据隔离 (Multi-Tenancy & Anti-Privilege-Escalation)**:
   - 租户组织模型 (`Tenant`) 强制隔离所有业务数据。
   - 中间件级防越权拦截：严格校验请求所携带的 `X-App-ID` 是否隶属于当前租户，彻底阻断跨租户水平越权攻击。

---

## 🚀 系统初始化与默认凭据

平台启动时，系统将自动执行自检与初始化流程：
1. **自动迁移数据库**：首次启动自动检测并创建所有数据表，免去繁琐的手动 SQL 导入。
2. **初始主租户**：自动创建默认租户 `ID: 1` (`默认组织`, code: `default`)。
3. **初始超级管理员账号**：
   - 用户名：`admin`
   - 初始密码：`admin123`
   - 归属租户：默认组织 (ID: 1)
   - 角色：`superadmin`

> ⚠️ **安全建议**：首次登录后，请立即进入「系统与权限」页面或通过 API 修改管理员初始密码！

---

## 📦 部署指南

### 部署方式一：极简单二进制部署 (推荐个人/轻量使用，零外部依赖)

系统默认采用纯 Go SQLite 引擎（`github.com/glebarez/sqlite`），无需安装 PostgreSQL 或 Redis，单文件即可开箱即用。

#### 1. 下载发行二进制
直接前往 [GitHub Releases](https://github.com/csbde/wechat-official-account-admin/releases) 页面，下载对应系统的编译产物（如 `woaa-linux-amd64.tar.gz`）。

#### 2. 解压并启动
```bash
# 解压发行包
tar -xzvf woaa-linux-amd64.tar.gz

# 赋予执行权限
chmod +x woaa

# 直接运行 (默认监听 9305 端口，数据文件自动生成在 ./data/woaa.db)
./woaa
```

在浏览器打开 `http://<服务器IP>:9305` 即可直接使用！

---

### 部署方式二：Docker & Docker-Compose 部署 (推荐团队/企业高可用生产环境)

该模式启用 PostgreSQL 16 + Redis 7 + Asynq 分布式队列，适合多租户大规模运营。

#### 1. 获取代码或解压包
```bash
git clone https://github.com/csbde/wechat-official-account-admin.git
cd wechat-official-account-admin
```

#### 2. 一键启动集群
```bash
docker compose up -d
```

#### 3. 查看容器状态与日志
```bash
docker compose ps
docker compose logs -f woaa
```

服务就绪后直接访问 `http://<服务器IP>:9305`。

---

### 部署方式三：Linux Systemd 服务进程守护

在生产服务器上，推荐使用 Systemd 管理守护进程，支持开机自启与异常自动重启。

#### 1. 创建应用目录并放置二进制
```bash
sudo mkdir -p /opt/woaa/data /opt/woaa/files /opt/woaa/config
sudo cp woaa /opt/woaa/
sudo cp config/config.yaml /opt/woaa/config/
sudo chmod +x /opt/woaa/woaa
```

#### 2. 创建 Systemd 服务文件
创建 `/etc/systemd/system/woaa.service`：
```ini
[Unit]
Description=WeChat Official Account Admin Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/woaa
ExecStart=/opt/woaa/woaa
Restart=always
RestartSec=5
LimitNOFILE=65536
Environment="WOAA_SERVER_MODE=release"

[Install]
WantedBy=multi-user.target
```

#### 3. 启动并设置开机自启
```bash
sudo systemctl daemon-reload
sudo systemctl enable woaa
sudo systemctl start woaa
sudo systemctl status woaa
```

---

### 部署方式四：从源码编译构建

系统要求：**Go 1.26+**、**Node.js 22+**、**pnpm 9+**。

```bash
# 1. 克隆仓库
git clone https://github.com/csbde/wechat-official-account-admin.git
cd wechat-official-account-admin

# 2. 构建前端单页应用
cd wechat-official-account-admin-fe
pnpm install
pnpm build
cd ..

# 3. 编译后端程序 (将前端产物静态内嵌至二进制)
go build -ldflags="-s -w" -o woaa main.go

# 4. 运行服务
./woaa
```

---

## 🌐 Nginx 反向代理与 SSL (HTTPS) 配置模板

微信公众平台回调与接口配置强制或推荐使用 HTTPS。以下为标准 Nginx 配置模板：

```nginx
server {
    listen 80;
    server_name wechat.yourdomain.com;
    # 强制重定向至 HTTPS
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name wechat.yourdomain.com;

    # SSL 证书配置
    ssl_certificate /etc/nginx/ssl/wechat.yourdomain.com.crt;
    ssl_certificate_key /etc/nginx/ssl/wechat.yourdomain.com.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # 上传文件大小限制 (适配微信大图与视频素材上传)
    client_max_body_size 50M;

    location / {
        proxy_pass http://127.0.0.1:9305;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # 支持 WebSocket (如有需要)
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

---

## 🔧 微信公众号后台接入配置指引

### 模式 A：普通微信公众号直接接入
1. 登录 [微信公众平台](https://mp.weixin.qq.com)，进入 **设置与开发 -> 基本配置**。
2. 记录 **开发者ID (AppID)** 与 **开发者密码 (AppSecret)**，填入本平台的「添加公众号」表单中。
3. 在微信后台点击 **修改配置**（服务器配置）：
   - **URL (服务器地址)**: 填入本系统生成的地址，格式为 `https://wechat.yourdomain.com/wxmp/<你的AppID>/handler`。
   - **Token (令牌)**: 填入本系统公众号设置中的 Token（如 `mytoken123`）。
   - **EncodingAESKey**: 随机生成或保持一致，消息加解密方式推荐选择 **安全模式** 或 **兼容模式**。
4. 在微信后台的 **IP 白名单** 中，添加本系统所在的公网出口 IP（可在平台仪表盘或「系统管理」一键查看）。
5. 点击提交并启用服务器配置。

### 模式 B：微信开放平台 (第三方平台) 接入
1. 登录 [微信开放平台](https://open.weixin.qq.com)，进入 **管理中心 -> 第三方平台**，创建平台并获取：
   - `ComponentAppID`
   - `ComponentAppSecret`
   - 消息校验 `Token` 与 `EncodingAESKey`
2. 在微信开放平台后台填写以下回调 URL：
   - **授权事件接收配置 URL**: `https://wechat.yourdomain.com/open/component/notify`
   - **代开发公众号消息与事件接收 URL**: `https://wechat.yourdomain.com/open/component/$APPID$/callback`
3. 在本平台进入「公众号管理」->「开放平台第三方设置」，填入上述第三方平台凭证保存。
4. 点击「微信扫码一键授权接入」，微信管理员微信扫码即可一键授权托管，公众号自动同步至平台，无需维护 AppSecret！

---

## ⚙️ 环境配置参数完全手册

系统支持通过 `config/config.yaml` 或系统环境变量进行配置。环境变量具有最高优先级，前缀为 `WOAA_`。

| YAML 配置路径 | 对应环境变量 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| `server.port` | `WOAA_SERVER_PORT` | `9305` | 服务监听端口 |
| `server.mode` | `WOAA_SERVER_MODE` | `debug` | 运行模式 (`debug` / `release`) |
| `database.type` | `WOAA_DATABASE_TYPE` | `sqlite` | 数据库类型 (`sqlite` 或 `postgres`) |
| `database.sqlite_path` | `WOAA_DATABASE_SQLITE_PATH` | `./data/woaa.db` | SQLite 数据文件持久化路径 |
| `database.host` | `WOAA_DATABASE_HOST` | 空 | PostgreSQL 主机地址 (配置时启用) |
| `database.port` | `WOAA_DATABASE_PORT` | `5432` | PostgreSQL 端口 |
| `database.user` | `WOAA_DATABASE_USER` | 空 | PostgreSQL 数据库用户名 |
| `database.password` | `WOAA_DATABASE_PASSWORD` | 空 | PostgreSQL 数据库密码 |
| `database.dbname` | `WOAA_DATABASE_DBNAME` | 空 | PostgreSQL 数据库名称 |
| `redis.addr` | `WOAA_REDIS_ADDR` | `127.0.0.1:6379` | Redis 连接地址 (可选) |
| `redis.password` | `WOAA_REDIS_PASSWORD` | 空 | Redis 连接密码 |
| `redis.db` | `WOAA_REDIS_DB` | `0` | Redis DB 库号 |
| `asynq.concurrency` | `WOAA_ASYNQ_CONCURRENCY` | `5` | Asynq 异步任务并发 Worker 数 |
| `jwt.secret` | `WOAA_JWT_SECRET` | 内置默认密钥 | JWT 签名密钥 (生产环境建议修改) |
| `jwt.expire_hours` | `WOAA_JWT_EXPIRE_HOURS` | `72` | 登录令牌有效期 (小时) |
| `wechat.public_host` | `WOAA_WECHAT_PUBLIC_HOST` | 空 | 外部访问公网域名 (若为空自动取 Host) |

---

## 📖 OpenAPI / Swagger 在线文档

平台内置符合 OpenAPI 3.0 规范的 Swagger UI 交互界面。
服务启动后，可在浏览器中直接访问：
`http://localhost:9305/swagger/index.html`

支持直接在线调试登录认证、公众号增删改查、素材管理、自定义菜单及粉丝接口。

---

## 📄 开源许可证

本项目基于 [MIT 许可证](LICENSE) 发布。欢迎提交 Issue 与 Pull Request 共同改进！
