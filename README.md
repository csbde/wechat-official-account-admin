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

9. **多租户严格数据隔离与租户管理 (Multi-Tenancy & Anti-Privilege-Escalation)**:
   - 租户组织模型 (`Tenant`) 强制隔离所有业务数据（公众号/小程序、菜单、素材、自动回复、粉丝与标签、带参二维码、群发与模板消息、第三方平台凭证、日志）。
   - 「系统与权限 → 租户与组织」支持超级管理员**新建 / 编辑 / 停用 / 删除租户**，并可一键为该租户创建初始管理员（角色固定 `admin`，不可自举超级管理员）。
   - **租户视角切换**：超级管理员通过请求头 `X-Tenant-ID` 显式切换到某个已启用租户（顶栏下拉即可操作），切换后新增的公众号与全部业务数据都归属该租户；普通用户携带该头会被无条件忽略。
   - 中间件级防越权拦截：`TenantAppID` 对**所有角色（含超级管理员）**校验 `X-App-ID` 是否隶属"当前租户视角"，跨租户访问一律 403 并写入告警日志。
   - 数据面收口：账号/用户/第三方平台配置的读写一律以「当前租户视角」为过滤条件；删除租户前必须先清空该租户下的微信账号与用户，避免产生孤立数据。
   - 同一个微信号 (AppID) 不允许被两个租户重复接入，避免多组织互相干扰（超级管理员可确认后用 `?force=1` 强制录入）。
   - OIDC 单点登录为**系统级配置**：仅超级管理员可修改，客户端密钥对非超级管理员自动脱敏；自动建号可指定归属租户，防止外部人员落入他租户。


10. **微信小程序代开发与快速注册 (Mini Program Lifecycle)**:
    - **免 300 元认证费通道**：依托第三方平台资质，企业与个体工商户无需向腾讯支付认证审核费，微信直接向法人微信号下发人脸核身通知。
    - **合法域名代配置**：无需进入微信公众平台，直接代小程序配置合法 Request / Socket / Upload / Download 及业务 Webview 域名。
    - **免密登录网关 (Code2Session)**：集成 WXGate 网关能力，外部或移动端直接调用后端换取 OpenID 与 SessionKey，无需在移动端暴露 AppSecret。

11. **公众号关联小程序管理 (Linked Mini Programs — 公众号维度)**:
    - **归属**：这是**公众号**的能力（使用公众号 `authorizer_access_token` 调用），管理的是"当前选中的公众号关联了哪些小程序"。
    - **微信官方通道**：调用 `cgi-bin/wxopen/wxamplinkget` 实时拉取已关联小程序列表，包含头像、昵称、原始ID、关联状态 (1 已关联 / 2 待小程序管理员确认 / 3 管理员拒绝 / 12 待公众号管理员确认) 与绑定邮箱。
    - **一键关联与解绑**：支持向小程序发起绑定申请 (`wxamplink`，可选 `notify_users`、`show_profile` 是否在公众号资料页展示入口) 并支持随时解绑 (`wxampunlink`)。
    - **仅第三方托管可用**：微信官方「适用范围」限定本组接口只能由**第三方平台代调用 (权限集 id=33)**。「直连凭据」公众号调用必返回 `48001 api unauthorized`，因此后端会先判定账号类型并返回 **400 + 明确处置指引** (改用「微信扫码一键授权接入」托管，或到微信公众平台后台手动管理)；前端在选中直连账号 / 小程序账号时直接展示说明横幅并**不发起请求**，不再出现莫名其妙的报错弹窗。

12. **小程序代码模板库与草稿箱 (Code Template Management — 小程序 / 开放平台维度)**:
    - **归属**：这是**小程序**（代码发布包）的能力，且属于**微信开放平台第三方平台账号级全局资源**（使用 `component_access_token` 调用），既不隶属公众号，也不隶属单个小程序，一次入库可分发给任意已授权小程序。
    - **草稿箱一键入库**：直接拉取小程序开发者工具上传的代码草稿箱 (`gettemplatedraftlist`)，并支持一键转化为正式代码模板 (`addtotemplate`)。
    - **模板库管理**：统一管理已收录的小程序代码模板 (`gettemplatelist`)，支持按类型过滤并支持删除过期模板 (`deltemplate`)。
    - **易混淆提示**：此"代码模板库"与微信的"模板消息 (`template_id` 消息通知)"完全是两套体系，本模块仅涉及小程序代码版本模板。

13. **微信官方权威档案与权限集详情 (Authorizer Detail)**:
    - 公众号列表操作栏提供「查看微信官方权威档案」按钮，调用 `api_get_authorizer_info` 实时获取微信官方原始数据。
    - 完整呈现昵称/头像/微信号 (`alias`)/原始ID、**认证主体全称**、**认证类型与账号类型的官方名称**、功能开通状况 (`business_info`：微信支付/摇一摇/扫商品/卡券/门店)、手机与邮箱绑定状态 (`basic_config`)、**账号二维码**、功能介绍，以及授权给本平台的**权限集清单** (`func_info`，含需管理员确认的 `confirm_info` 状态)。
    - **小程序授权账号**额外展开 `MiniProgramInfo`：服务类目、六类合法域名 (request/socket/uploadFile/downloadFile/udp/tcp) 清单，以及 `register_type` / `account_status`。
    - **绝不空白**：直连凭据账号或官方接口调用失败时，自动降级返回本系统落库档案 + 授权时的权限集快照，并在弹窗内明确标注原因 (`official_error` / `hint`)；弹窗底部另附「微信官方原始响应」折叠面板便于排障。
    - 授权回调时会把官方档案中的原始ID、账号二维码、功能介绍一并落库，减少重复调用与离线不可见问题。



14. **小程序代码管理与发布 (Code Management — 小程序 / 开放平台维度)**:
    - 覆盖微信官方「代码管理」全链路接口，全部使用目标小程序的 `authorizer_access_token`，并由 `TenantAppID` 中间件强制校验小程序归属租户，杜绝跨租户越权代管他人小程序代码。
    - **提交代码 `wxa/commit`**：选择代码模板库中的 template_id，填写版本号与描述即可为任意已授权小程序上传开发版本；`ext_json` 留空时自动按目标小程序生成 (支持手工覆盖 `requiredPrivateInfos` 等高级配置)。
    - **体验版 `wxa/get_qrcode`**：在线获取体验版二维码 (可指定落地页 path 与参数)，配合体验者绑定 (`wxa/bind_tester`、`wxa/unbind_tester`、`wxa/memberauth`) 直接扫码体验。
    - **提交审核 `wxa/submit_audit`**：可视化填写至多 5 个审核项 (页面地址 + 类目 ID/名称 + 标签)，页面地址可从 `wxa/get_page` 返回的代码包页面列表一键填入；支持 `privacy_api_not_use` 声明规避 61040。
    - **审核跟踪与撤回**：`wxa/get_latest_auditstatus` 总览最近一次提审、`wxa/get_auditstatus` 按单号精查、`wxa/undocodeaudit` 一键撤回；并提供 `wxa/security/get_code_privacy_info` 隐私接口检测 (避免 61039 检测未完成即提审)。
    - **发布与回退**：`wxa/release` 全量发布、`wxa/revertcoderelease` 查看历史线上版本并回退到指定版本、`wxa/getvisitstatus` / `wxa/change_visitstatus` 控制服务可见性 (暂停 / 恢复)。
    - **基础库与类目**：`getweappsupportversion` / `setweappsupportversion` 管理最低基础库版本；`wxopen/getcategory`、`wxopen/addcategory`、`wxopen/deletecategory` 管理服务类目及类目审核状态与配额。


15. **模板消息与订阅消息 (Template & Subscribe Message)**:
    - 侧边栏「群发与模板消息」页面，随顶部账号切换自动携带 `X-App-ID` 并由租户中间件鉴权。
    - **公众号模板消息**：查询/设置所属行业 (`template/get_industry`、`api_set_industry`)、从公共模板库选用模板 (`api_add_template`，支持指定关键词组合)、查看与删除我的模板库 (`get_all_private_template`、`del_private_template`)、发送模板消息 (`message/template/send`，支持 `url`、跳转小程序 `miniprogram`、防重入 `client_msg_id` 与字体颜色)、一次性订阅下发 (`message/template/subscribe`)、查询被拦截的模板消息 (`wxa/sec/queryblocktmplmsg`)。
    - **订阅消息 (小程序 · 服务号)**：拉取类目 (`wxaapi/newtmpl/getcategory`) → 分页浏览公共模板库 (`getpubtemplatetitles`) → 查看关键词 (`getpubtemplatekeywords`) → 勾选 2~5 个关键词加入我的模板库 (`addtemplate` 返回 `priTmplId`) → 我的订阅模板列表与删除 (`gettemplate`、`deltemplate`) → 发送 (`message/subscribe/send`，可选 `formal/trial/developer` 版本与跳转页) 或服务号一次性订阅下发 (`message/subscribe/bizsend`)。
    - **智能表单**：自动解析模板 `content` 中的 `{{xxx.DATA}}` 占位，动态生成关键词输入框，并以模板 `example` 作为填写示例，避免 47003 (参数格式不符) 类错误。
    - **加载状态可诊断**：类目列表、公共模板库、我的模板库、模板关键词、行业信息等每一处拉取都区分「加载中 / 失败原因 / 空数据」三态，失败时直接显示微信返回的中文处置建议并提供「重试」按钮，不会再出现一直卡在"正在拉取…"却查不到原因的界面。

16. **群发消息与服务提醒卡片 (Mass Messaging & Service Card)**:
    - 同页面新增「群发消息」「服务提醒卡片」两个标签页，接口统一位于 `/api/notify/mass/*` 与 `/api/notify/sub/service-card*`，全部经过 `TenantAppID` 租户校验 (跨租户 appid 直接 403)，参数类错误返回 400、微信侧错误返回 500。
    - **群发内容**：支持 `text` / `mpnews` (图文) / `image` (贴图，多张 `media_ids`) / `voice` / `mpvideo` / `wxcard` 六类消息，扁平化入参由后端组装微信要求的结构，并附带 `clientmsgid` (24 小时内防重复群发) 与 `send_ignore_reprint` (原创转载策略)。
    - **接收者三选一**：全部用户 (`sendall` + `is_to_all=true`)、指定粉丝标签 (`sendall` + `tag_id`，标签下拉来自「粉丝与标签管理」)、指定 openid 列表 (`send`，2~10000 个，支持多行粘贴或从粉丝列表点选)；另提供 `preview` 预览接口 (`touser` 或 `towxname` 微信号预览，不占用群发次数)。
    - **任务管理**：`message/mass/get` 按 `msg_id` 查询发送状态 (`SEND_SUCCESS`/`SENDING`/`SEND_FAIL`/`DELETE` 中文化展示)、`message/mass/delete` 按 `msg_id` 或图文 `url` + `article_idx` 撤回未发送内容、`message/mass/speed/get|set` 查看与设置群发速率档位 (0=80 万/分 ~ 4=10 万/分)。
    - **图文素材准备 (群发前置)**：`media/uploadimg` 上传图文正文图片并直接返回可用 URL (不占用素材库配额)、`material/add_material?type=image` 上传永久图片素材返回真实 `thumb_media_id`、`media/uploadnews` 一键新建图文素材并回填群发所需的 `media_id`，彻底解决"素材型图文群发没有 media_id"的断链问题。
    - **服务提醒卡片**：`wxa/set_user_notify` 激活与更新卡片 (`notify_type` + `notify_code` + `content_json`，微信支付订单号作 code 时可填 `check_json`)，`wxa/get_user_notify` 查询卡片状态 (`code_state` 0 正常 / 1 有风险 / 2 异常 / 10 用户拒收) 并可一键回填上次推送内容；内置 1001、2001 卡片示例便于快速联调。
    - **频次与资质提示**：订阅号每天 1 次、认证服务号每月 4 次群发 (超出返回 44004)；第三方平台代群发需勾选权限集 **id=7**，素材上传需 **id=11 或 100**，服务卡片需权限集 **id=18** (详见下文「权限集勾选清单」)；若账号开启「API 群发保护」，群发全部用户需管理员在 30 分钟内确认。
    - **粉丝标签同步修正**：新建标签现在会真实调用 `cgi-bin/tags/create` 取回微信分配的 `tag_id` 后再落库 (官方创建失败则拒绝保存本地假标签)，保证批量打标签与「按标签群发」可用。

---


---

## ⚡ 微信认证与接口代理网关 (WXGate 功能集成)

本项目已深度融合 **WXGate** 的高并发凭证中心与授权网关能力，解决中大型团队多系统协同开发时的四大核心痛点：

### 1. 统一凭证中心（彻底告别 Token 互相踢下线与限频）
- **痛点**：多个子系统、不同业务线各自分别调用微信获取 `access_token` 或 `jsapi_ticket`，会导致旧 Token 被新 Token 覆盖失效，频繁出现“Token invalid”报警。
- **网关解决方案**：
  - 子系统只需将原本请求微信的域名改为本网关地址：
    - `GET/POST /cgi-bin/token?grant_type=client_credential&appid=APPID&secret=SECRET`
    - `POST /cgi-bin/stable_token`
    - `GET/POST /cgi-bin/ticket/getticket?access_token=TOKEN&type=jsapi`
  - **L1 本地内存读写锁 + L2 Redis 双级缓存**，自动提前 300 秒续期。
  - **SingleFlight 单飞并发防击穿合并锁**：万级高并发下同时失效时，仅向微信官方发起 1 次网络换取，其他请求共享结果，从容承载极高并发。

### 2. 突破微信单一授权回调域名限制（OAuth2 代理中转）
- **痛点**：微信公众号后台仅允许配置 1~2 个授权回调域名。多个不同子域名、测试环境、本地 `localhost` 无法完成微信授权登录。
- **网关解决方案**：
  - 只有本网关的域名需要在微信后台配置一次；
  - 任意业务系统或前端直接发起跳转网关：
    ```
    https://<你的网关域名>/connect/oauth2/authorize?appid=YOUR_APPID&redirect_uri=https://any-domain.com/callback&scope=snsapi_userinfo&state=STATE#wechat_redirect
    ```
  - **自动脱敏回传 `user_info`**：授权成功后，网关在服务端自动替业务端换取 Token 并拉取脱敏后的用户信息 `user_info`（包含 `openid`, `nickname`, `headimgurl`, `sex` 等），直接附带在 URL Query 重定向回业务原本的 `redirect_uri`，业务端无需再走二次换 Token 流程！

### 3. 前端极速 JSSDK 签名与脚本代理
- **一行配置，前端 2 分钟搞定微信分享**：
  ```html
  <!-- 1. 引入网关加速代理的 JSSDK -->
  <script src="https://<你的网关域名>/jweixin.js"></script>

  <script>
  async function initShare() {
    // 2. 向网关获取动态签名 (网关自动调度全局 token 和 ticket)
    const currentUrl = encodeURIComponent(window.location.href.split("#")[0]);
    const res = await fetch(`https://<你的网关域名>/api/jssdk/config?url=${currentUrl}`);
    const { data } = await res.json();

    // 3. 注入配置
    wx.config({
      appId: data.appId,
      timestamp: data.timestamp,
      nonceStr: data.nonceStr,
      signature: data.signature,
      jsApiList: data.jsApiList
    });
  }
  initShare();
  </script>
  ```
- **网关管理监控与一键清理**：
  - 查看监控与缓存统计：`GET /api/wxgate/stats`
  - 彻底清空指定应用凭证缓存：`DELETE /api/wxgate/cache/:app_id`

## 🚀 系统初始化与默认凭据

平台启动时，系统将自动执行自检与初始化流程：
1. **自动迁移数据库**：首次启动自动检测并创建所有数据表，免去繁琐的手动 SQL 导入。
2. **初始主租户**：自动创建默认租户 `ID: 1` (`默认组织`, code: `default`)。
3. **初始超级管理员账号**：
   - 用户名：`admin`
   - 初始密码：`admin123`
   - 归属租户：默认组织 (ID: 1)
   - 角色：`superadmin`

### 多租户（组织）快速上手

平台天生多租户：**一个租户 = 一个独立组织（分公司 / 客户 / 代运营主体）**，租户之间数据完全不可见。

1. **创建租户**：超级管理员进入「系统与权限 → 租户与组织 → 新建租户」，填写组织名称与唯一标识 (code，字母开头，如 `east-branch`)，并勾选"同时创建初始管理员"，设置其用户名/密码/绑定邮箱。
   - API 方式：
     ```bash
     curl -X POST http://127.0.0.1:8080/api/system/tenants \
       -H "Authorization: Bearer <超级管理员Token>" -H 'Content-Type: application/json' \
       -d '{"name":"华东分公司","code":"east-branch","description":"华东区代运营",
            "admin_username":"east_admin","admin_password":"east123456","admin_email":"east@company.com"}'
     ```
2. **为该租户接入公众号**（两种等价方式）：
   - 直接用刚创建的 `east_admin` 登录，在「公众号与开放平台」添加或扫码授权接入 —— 数据自动归属华东分公司；
   - 或超级管理员在顶栏「租户视角」下拉切换到「华东分公司」后再录入（页面顶部会显示当前租户视角与代管警示）。
3. **成员与权限**：`系统与权限 → 后台用户` 中，超级管理员可指定成员所属租户（也可开启"跨租户总览"查看所有成员）；普通管理员只能维护本租户成员。角色含义：
   - `superadmin` 超级管理员：可管理租户、OIDC 等系统级配置，可切换租户视角；
   - `admin` 租户管理员：本租户全量业务权限（含查看/修改本租户公众号凭据）；
   - `operator` / `viewer`：本租户业务操作 / 只读。
4. **停用与删除**：停用租户后其成员无法登录、也无法被切换视角（列表内该行的"切换视角"按钮会自动禁用）；删除租户要求该租户下**没有任何微信账号与用户**，且不允许删除/停用你自己所属的租户。
5. **隔离边界与审计**：跨租户请求返回 `403 越权访问被拒绝…`，并在服务端日志中记录 `username / tenant_id / home_tenant_id / appid / client_ip`；「请求与追踪日志」页可按本租户审计。

> 说明：租户数据在同表内通过 `tenant_id` 逻辑隔离，所有查询与写入都由中间件与 handler 双重强制附加租户条件；如需物理隔离部署（每租户独立库），请使用多实例部署方案。

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
5. **微信错误码自动中文化 (全局)**：`[40001] invalid credential`、`[48001] api unauthorized`、`[61005] 授权失效`、`[89503] 需管理员确认` 等常见错误，系统会在原始返回码之后自动追加「处置建议」中文说明 (`internal/service/wx_error.go`)，避免只看到一句英文 errmsg 无从下手。
6. **权限集勾选清单** (授权时决定哪些能力可代调用，缺项会导致对应页面报 `48001 api unauthorized`)，以下 ID 均取自微信官方各接口页的「该接口所属的权限集 id」：
   - 群发消息 (`cgi-bin/message/mass/send`、`sendall`、`get`、`delete`、`preview`、`speed/*`)：**id=7**
   - 群发图文与素材 (`media/uploadnews`、`material/add_material`)：**id=1、7、8、9、11、100** 中任一 (建议同时勾选 7 与 11)
   - 图文正文图片 (`media/uploadimg`)：**id=1、8、9、11、18、37、100** 中任一
   - 服务提醒卡片 (`wxa/set_user_notify`、`wxa/get_user_notify`)：**id=18**
   - 公众号关联小程序 (`cgi-bin/wxopen/wxamplinkget`、`wxamplink`、`wxampunlink`)：**id=33** (「小程序关联管理」权限集，直连凭据账号无法调用)
   - 自定义菜单、粉丝与标签、带参二维码等按对应页面提示补齐

---

## ⚙️ 环境配置参数完全手册

系统支持通过 `config/config.yaml` 或系统环境变量进行配置。环境变量具有最高优先级，前缀为 `WOAA_`。

| YAML 配置路径 | 对应环境变量 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| `server.host` | `WOAA_SERVER_HOST` | `0.0.0.0` | 监听主机/IP (支持指定 127.0.0.1 仅本地、内网具体网卡IP、或 0.0.0.0) |
| `server.port` | `WOAA_SERVER_PORT` | `9305` | 服务监听端口 |
| `server.listen_addr` | `WOAA_SERVER_LISTEN_ADDR` | 空 | 完整监听地址 (设置时优先于 HOST+PORT，例如: 127.0.0.1:9305) |
| `server.trusted_proxies` | `WOAA_SERVER_TRUSTED_PROXIES` | 常见内网段+回环 | 信任的上级反向代理列表 (Nginx/Cloudflare)，反代场景必须信任以防IP被误判为127.0.0.1 |
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
| `oidc.enabled` | `WOAA_OIDC_ENABLED` | `false` | 是否启用 OIDC 单点登录 (仅超级管理员可在后台可视化修改) |
| `oidc.issuer` | `WOAA_OIDC_ISSUER` | 空 | OIDC Issuer，如 `https://auth.example.com/realms/master` |
| `oidc.client_id` / `oidc.client_secret` | `WOAA_OIDC_CLIENT_ID` / `WOAA_OIDC_CLIENT_SECRET` | 空 | SSO 客户端凭据 (密钥对非超级管理员自动脱敏) |
| `oidc.redirect_uri` | `WOAA_OIDC_REDIRECT_URI` | 空 | 回调地址，留空自动按 `public_host`/反向代理协议生成 |
| `oidc.scopes` | `WOAA_OIDC_SCOPES` | `openid profile email` | 授权范围 (务必包含 `email` 才能按邮箱匹配绑定) |
| `oidc.auto_register` | `WOAA_OIDC_AUTO_REGISTER` | `false` | 未录入邮箱的 OIDC 用户是否自动建号 (推荐保持关闭) |
| `oidc.default_role` | `WOAA_OIDC_DEFAULT_ROLE` | `operator` | 自动建号角色 (仅 `operator` / `viewer`，禁止自动提权) |
| `oidc.default_tenant_id` | `WOAA_OIDC_DEFAULT_TENANT_ID` | `0` | 自动建号归属租户 ID；`0` 表示沿用该 OIDC 配置所属租户，最终兜底默认租户 1 |
| `wechat.public_host` | `WOAA_WECHAT_PUBLIC_HOST` | 空 | 外部访问公网域名 (若为空自动取 Host) |

### 租户与组织相关接口

| 方法 | 路径 | 权限 | 说明 |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/system/tenants` | 超级管理员=全部；其他=仅本租户(只读) | 租户列表，附带微信账号数/用户数与视角标记 |
| `GET` | `/api/system/tenants/options` | 全部(按可见范围) | 租户下拉选项 (分配成员归属时使用) |
| `POST` | `/api/system/tenants` | **仅超级管理员** | 新建租户，可同时创建该租户初始管理员 (`admin`) |
| `PUT` | `/api/system/tenants/:id` | **仅超级管理员** | 修改名称/描述/启停 (唯一标识 code 不可改) |
| `DELETE` | `/api/system/tenants/:id` | **仅超级管理员** | 删除空租户 (仍有账号或用户时返回 400) |
| `GET/POST/PUT/DELETE` | `/api/system/users` | 当前租户视角内 | 成员管理；超级管理员可带 `tenant_id` 跨租户分配/迁移，`?all=1` 跨租户总览 |

> 请求头约定：`X-Tenant-ID: <id>` 仅对超级管理员生效（用于切换租户视角），后端会校验该租户存在且已启用，否则返回 403；普通用户携带该头将被无条件忽略。

---

## 📖 OpenAPI / Swagger 在线文档

平台内置符合 OpenAPI 3.0 规范的 Swagger UI 交互界面。
服务启动后，可在浏览器中直接访问：
`http://localhost:9305/swagger/index.html`

支持直接在线调试登录认证、公众号增删改查、素材管理、自定义菜单及粉丝接口。

---

## 📄 开源许可证

本项目基于 [MIT 许可证](LICENSE) 发布。欢迎提交 Issue 与 Pull Request 共同改进！
