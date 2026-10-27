# 微信公众号管理平台 (WeChat Official Account Admin - WOAA Pro)

全栈现代化重构的多微信公众号综合管理平台。

## 🛠️ 重构后技术栈

### 前端技术栈
- **核心框架**: React 18 + TypeScript + Vite
- **UI & 样式库**: Tailwind CSS + shadcn/ui (Radix UI + Lucide-react)
- **数据获取与缓存**: TanStack Query (React Query v5)
- **高级数据表格**: TanStack Table (v8)
- **状态管理**: Zustand (轻量、高效管理 Token、当前公众号租户及深浅主题)
- **表单与校验**: React Hook Form + Zod 模式验证
- **设计体验**: 微信绿主题系统、支持深色/浅色模式切换、高仿真微信手机端自定义菜单设计器

### 后端技术栈
- **语言与 Web 框架**: Go 1.23 + Gin (RESTful API 架构)
- **数据库 ORM**: GORM + PostgreSQL
- **高速缓存**: Redis (AccessToken 分布式缓存与提前续期)
- **后台异步任务**: Asynq (分布式延迟/重试任务队列，处理粉丝批量同步、素材下载等)
- **访问控制**: Casbin (基于 RBAC 的动态角色访问控制，策略持久化至 PostgreSQL)
- **安全认证**: JWT Bearer Token + SSO 单点登录对接支持
- **配置中心**: Viper (支持 YAML 文件与系统环境变量覆盖)
- **结构化日志**: Zap (集成 Gin 请求与全局日志)
- **数据校验**: Validator (go-playground/validator v10)
- **接口文档**: OpenAPI 3.0 / Swagger UI (`/swagger/index.html`)
- **数据库迁移**: Atlas (声明式 Schema 管理与版本迁移)
- **链路观测**: OpenTelemetry (请求追踪、耗时统计与 TraceID 关联审计日志)
- **测试框架**: Testify

---

## 🚀 核心业务功能

1. **多微信公众号管理 (Accounts)**:
   - 快速增删改查公众号凭据 (AppID, AppSecret, Token, EncodingAESKey)
   - 顶部快捷切换当前激活的微信公众号
   - 微信后台服务器配置 (URL 与 Token) 一键复制
   - 设为默认公众号与启停控制

2. **微信自定义菜单设计器 (Menu)**:
   - 高仿真手机视图预览
   - 一级菜单（最多3个）、二级菜单（最多5个）可视化增删与拖拽交互
   - 动作支持：跳转网页 (`view`)、点击事件/自动回复 (`click`)、微信小程序跳转 (`miniprogram`)、扫码推事件等
   - 个性化条件菜单（按粉丝标签、性别、操作系统匹配）
   - 从微信线上拉取同步 & 一键发布至微信服务器

3. **多媒体素材库 (Materials)**:
   - 图片、语音、视频、图文素材管理
   - 支持本地文件上传并同步生成微信 MediaID
   - 本地缓存直连预览与一键复制 MediaID

4. **智能自动回复 (AutoReply)**:
   - 关注公众号后欢迎语回复
   - 收到消息默认应答
   - 关键词回复规则（支持多关键词、精准全匹配 / 模糊半匹配）

5. **粉丝关注者与标签 (Fans & Tags)**:
   - 基于 TanStack Table 的高级表格，支持多维搜索、分页与勾选
   - 修改粉丝专属备注名
   - 微信标签管理（创建、删除、重命名）
   - 批量打标签与批量移除标签
   - Asynq 后台异步全量同步粉丝

6. **带参二维码营销 (Qrcode)**:
   - 临时二维码（带过期倒计时）与永久二维码生成
   - 场景值（数字/字符串）追踪扫码渠道
   - 实时预览与下载二维码图片

7. **请求与链路追踪审计 (Logs)**:
   - 记录每次 API 请求的 Client IP、耗时、状态码、请求体与响应体
   - 自动关联 OpenTelemetry TraceID

8. **系统与安全 (System)**:
   - 管理员与操作员账号维护、密码重置
   - 出口公网 IP 查询（用于配置微信公众号 IP 白名单）
   - 组件健康探测 (`/api/system/health`)

---

## 🏃 快速启动

### 1. 运行依赖
确保本地或 Docker 已运行 PostgreSQL (5432) 和 Redis (6379)。
项目配置文件位于 `config/config.yaml`。

### 2. 数据库迁移 (Atlas)
```bash
atlas schema apply \
  -u "postgres://sc:sc@127.0.0.1:5432/woaa?sslmode=disable" \
  --to "file://atlas/schema.hcl" \
  --auto-approve
```

### 3. 构建并启动
```bash
# 构建前端
cd wechat-official-account-admin-fe
pnpm install
pnpm build
cd ..

# 编译后端
go build -o woaa main.go

# 运行平台
./woaa
```

服务启动后，在浏览器访问 `http://localhost:9305` 即可使用。
- 默认管理员账号：`admin`
- 默认管理员密码：`admin123`
- OpenAPI Swagger 接口文档地址：`http://localhost:9305/swagger/index.html`
