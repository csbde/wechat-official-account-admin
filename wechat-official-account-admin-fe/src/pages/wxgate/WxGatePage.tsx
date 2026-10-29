import React, { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { useAppStore } from '@/stores/app'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  Zap,
  Globe,
  Share2,
  Copy,
  Trash2,
  Activity,
  Layers,
  Code2,
  CheckCircle2,
  Laptop,
  Compass,
  QrCode,
  Smartphone,
  RefreshCw,
  ShieldAlert,
  ShieldCheck,
  Check,
  User,
  HeartPulse,
  ExternalLink,
} from 'lucide-react'
import { toast } from 'sonner'

export const WxGatePage: React.FC = () => {
  // 提取纯域名供微信后台配置网页授权域名使用 (严禁带协议和端口)
  const currentDomainOnly =
    typeof window !== 'undefined'
      ? window.location.hostname
      : 'your-domain.com'

  const queryClient = useQueryClient()
  const { currentApp } = useAppStore()

  // JSSDK 测试
  const [testUrl, setTestUrl] = useState(
    typeof window !== 'undefined' ? window.location.href.split('#')[0] : 'https://your-domain.com'
  )
  const [jssdkResult, setJssdkResult] = useState<any>(null)
  const [loadingJSSDK, setLoadingJSSDK] = useState(false)

  // 目标业务系统实际回调地址 (自适应当前真实域名，不再生硬写死 localhost)
  const initialRedirect =
    typeof window !== 'undefined' && !window.location.hostname.includes('localhost')
      ? `${window.location.origin}/oauth/callback`
      : 'https://app.yourcompany.com/login/callback'

  const [targetRedirect, setTargetRedirect] = useState(initialRedirect)
  const [authScope, setAuthScope] = useState('snsapi_userinfo')

  // 扫码调试模式: 是否使用内置沙箱回显
  const [useDebugEcho, setUseDebugEcho] = useState(true)
  const [debugSessionId, setDebugSessionId] = useState('')
  const [scannedUser, setScannedUser] = useState<any>(null)
  const [isPolling, setIsPolling] = useState(false)

  // 健康诊断
  const [diagnoseLoading, setDiagnoseLoading] = useState(false)
  const [diagnoseResult, setDiagnoseResult] = useState<any>(null)

  // 获取网关运行统计与监控
  const { data: gateStats } = useQuery<any>({
    queryKey: ['wxgate-stats'],
    queryFn: () => api.get('/api/wxgate/stats'),
    refetchInterval: 5000,
  })

  // 获取 JSSDK 配置
  const handleFetchJSSDK = async () => {
    try {
      setLoadingJSSDK(true)
      const appQuery = currentApp?.appid ? `&appid=${currentApp.appid}` : ''
      const res: any = await api.get(`/api/jssdk/config?url=${encodeURIComponent(testUrl)}${appQuery}`)
      setJssdkResult(res?.data || res)
      toast.success('JSSDK 签名生成成功！')
    } catch (e: any) {
      toast.error(e.message || '获取配置失败')
    } finally {
      setLoadingJSSDK(false)
    }
  }

  // 清除指定应用缓存
  const clearCacheMutation = useMutation({
    mutationFn: (appId: string) => api.delete(`/api/wxgate/cache/${appId}`),
    onSuccess: (data: any) => {
      toast.success(data?.message || '缓存已清空')
      queryClient.invalidateQueries({ queryKey: ['wxgate-stats'] })
    },
  })

  const copyText = (txt: string, label: string) => {
    navigator.clipboard.writeText(txt)
    toast.success(`已复制 ${label} 到剪贴板`)
  }

  // 动态生成 OAuth2 授权中转链接 (自适应当前网关域名)
  const gatewayBase = typeof window !== 'undefined' ? window.location.origin : 'https://your-gateway.com'
  const effectiveRedirect = useDebugEcho
    ? `${gatewayBase}/oauth/debug`
    : targetRedirect

  const generatedOAuthURL = currentApp
    ? `${gatewayBase}/connect/oauth2/authorize?appid=${currentApp.appid}&redirect_uri=${encodeURIComponent(
        effectiveRedirect
      )}&response_type=code&scope=${authScope}&state=STATE#wechat_redirect`
    : '请先在顶部选择一个微信公众号'

  // 二维码图片链接
  const oauthQRCodeURL = currentApp
    ? `/api/wxgate/qrcode?text=${encodeURIComponent(generatedOAuthURL)}`
    : ''

  // 轮询检查扫码登录测试结果
  useEffect(() => {
    let timer: any = null
    if (isPolling) {
      timer = setInterval(async () => {
        try {
          const res: any = await api.get('/api/wxgate/stats')
          // 从网关最近完成的会话或通过 session 匹配获取
        } catch (e) {}
      }, 2000)
    }
    return () => {
      if (timer) clearInterval(timer)
    }
  }, [isPolling])

  // 执行接口与白名单体检
  const handleDiagnose = async () => {
    if (!currentApp) {
      toast.error('请先选择一个公众号')
      return
    }
    try {
      setDiagnoseLoading(true)
      const res: any = await api.get(`/api/wxgate/diagnose?appid=${currentApp.appid}`)
      setDiagnoseResult(res?.data || res)
      toast.success('体检诊断完成！')
    } catch (e: any) {
      toast.error(e.message || '体检诊断失败')
    } finally {
      setDiagnoseLoading(false)
    }
  }

  return (
    <div className="space-y-4 sm:space-y-6">
      {/* 顶部标题与核心特性 */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold tracking-tight flex items-center gap-2">
            <span>微信认证与接口代理网关 (WXGate)</span>
            <Badge variant="success" className="text-xs">高性能缓存与单飞防穿透</Badge>
          </h2>
          <p className="text-sm text-muted-foreground">
            统一全局 Token 凭证中心（彻底告别子系统互相踢 Token）、突破单授权域名限制（支持任意子域名与 localhost 联调）及 JSSDK 极速配置
          </p>
        </div>
      </div>

      {/* 核心指标统计 */}
      <div className="grid gap-3 sm:gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Zap className="h-4 w-4 text-amber-500" />
              <span>Token / Ticket 两级高速缓存</span>
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-1">
            <div className="text-2xl font-bold font-mono">
              {(gateStats?.cache?.l1_memory_items ?? gateStats?.data?.cache?.l1_memory_items) ?? 0} 个凭证活跃
            </div>
            <p className="text-xs text-muted-foreground">
              L1 本地内存读写锁 + L2 Redis: {(gateStats?.cache?.l2_redis_online ?? gateStats?.data?.cache?.l2_redis_online) ? '已联网' : '内存自适应'}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Globe className="h-4 w-4 text-emerald-600" />
              <span>突破域名限制 (OAuth 会话)</span>
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-1">
            <div className="text-2xl font-bold font-mono">
              {(gateStats?.sessions?.active_sessions ?? gateStats?.data?.sessions?.active_sessions) ?? 0}
            </div>
            <p className="text-xs text-muted-foreground">支持任意外部 redirect_uri 授权中转与信息直传</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Activity className="h-4 w-4 text-primary" />
              <span>单飞合并防击穿 (SingleFlight)</span>
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-1">
            <div className="text-2xl font-bold text-primary font-mono">Active</div>
            <p className="text-xs text-muted-foreground">万级高并发下仅发起单次微信网络请求，提前 300s 续期</p>
          </CardContent>
        </Card>
      </div>

      {/* 功能区块：场景一 与 场景二 */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* 场景一：微信网页授权登录代理 */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <Globe className="h-5 w-5 text-emerald-600" />
              <span>场景一：网页授权登录中转（突破单域名限制）</span>
            </CardTitle>
            <CardDescription className="text-xs">
              只需在微信公众平台配置当前网关域名，所有子系统、测试环境甚至本地 localhost 均可正常发起授权并自动脱敏带回用户信息！
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3.5 text-xs">
            {/* 微信 10003 域名配置指引提示条 */}
            <div className="p-3 rounded-xl bg-amber-500/10 border border-amber-500/20 text-xs text-amber-900 dark:text-amber-200 space-y-2">
              <div className="flex items-center justify-between font-semibold">
                <span className="flex items-center gap-1.5">
                  <span className="inline-block h-2 w-2 rounded-full bg-amber-500"></span>
                  <span>微信后台必须填写的【网页授权域名】(仅填网关域名一次即可):</span>
                </span>
                <button
                  type="button"
                  onClick={() => copyText(currentDomainOnly, '授权域名')}
                  className="text-primary hover:underline text-[11px] font-mono font-bold flex items-center gap-0.5"
                >
                  <Copy className="h-3 w-3" />
                  <span>复制域名: {currentDomainOnly}</span>
                </button>
              </div>

              <div className="text-[11px] text-muted-foreground leading-relaxed">
                {currentApp?.account_type === 'authorized' ? (
                  <span>
                    👉 <strong>当前为第三方代管号</strong>：请前往 <strong>微信开放平台后台</strong> -&gt; 第三方平台详情 -&gt; <strong>开发配置</strong> -&gt; <strong>公众号开发域名</strong> 中填入上方的纯域名 <code>{currentDomainOnly}</code>。
                  </span>
                ) : (
                  <span>
                    👉 <strong>当前为普通直连号</strong>：请前往 <strong>微信公众平台后台</strong> -&gt; 设置与开发 -&gt; <strong>接口权限</strong> -&gt; 网页服务 -&gt; <strong>网页账号 (网页授权获取用户基本信息)</strong> -&gt; 点击修改填入上方的纯域名 <code>{currentDomainOnly}</code>。
                  </span>
                )}
                <br />
                <span className="text-destructive font-semibold">⚠️ 提示：若出现 10003 报错，99.9%% 是因为微信后台上述位置尚未填写该域名，或填了端口号/协议头！</span>
              </div>
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <label className="font-semibold text-foreground">测试模式与回调目标 (redirect_uri):</label>
                <div className="flex items-center gap-1 bg-muted p-0.5 rounded-lg text-xs">
                  <button
                    type="button"
                    onClick={() => setUseDebugEcho(true)}
                    className={`px-2 py-0.5 rounded text-[11px] font-semibold transition-all ${
                      useDebugEcho ? 'bg-background text-primary shadow-sm' : 'text-muted-foreground hover:text-foreground'
                    }`}
                  >
                    ✨ 内置沙箱测试回显
                  </button>
                  <button
                    type="button"
                    onClick={() => setUseDebugEcho(false)}
                    className={`px-2 py-0.5 rounded text-[11px] font-semibold transition-all ${
                      !useDebugEcho ? 'bg-background text-primary shadow-sm' : 'text-muted-foreground hover:text-foreground'
                    }`}
                  >
                    外部业务系统回调
                  </button>
                </div>
              </div>

              {useDebugEcho ? (
                <div className="p-2.5 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-[11px] text-emerald-800 dark:text-emerald-300 space-y-1">
                  <div className="font-semibold flex items-center gap-1">
                    <CheckCircle2 className="h-3.5 w-3.5 text-emerald-600" />
                    <span>已启用沙箱调试回显 (推荐测试): <code>{gatewayBase}/oauth/debug</code></span>
                  </div>
                  <div>手机微信扫下方二维码授权后，页面会直接展示扫码微信的昵称、头像、OpenID、地区与授权凭证，无需搭建任何前端！</div>
                </div>
              ) : (
                <>
                  <div className="flex items-center justify-end gap-1.5 text-[11px] text-muted-foreground">
                    <span>快捷填入:</span>
                    <button
                      type="button"
                      onClick={() => setTargetRedirect(`${window.location.origin}/oauth/callback`)}
                      className="text-primary hover:underline flex items-center gap-0.5"
                    >
                      <Compass className="h-3 w-3" />当前站点
                    </button>
                    <span>|</span>
                    <button
                      type="button"
                      onClick={() => setTargetRedirect('https://app.yourcompany.com/login/callback')}
                      className="text-primary hover:underline"
                    >
                      子域名
                    </button>
                    <span>|</span>
                    <button
                      type="button"
                      onClick={() => setTargetRedirect('http://localhost:3000/callback')}
                      className="text-muted-foreground hover:text-foreground flex items-center gap-0.5"
                    >
                      <Laptop className="h-3 w-3" />本地 localhost
                    </button>
                  </div>
                  <Input
                    value={targetRedirect}
                    onChange={(e) => setTargetRedirect(e.target.value)}
                    placeholder="例如: https://sub.yourcompany.com/callback 或 http://localhost:3000/callback"
                    className="font-mono text-xs"
                  />
                </>
              )}
              <p className="text-[11px] text-muted-foreground">
                说明：此地址是您自己的具体业务系统接收登录回调的 URL。网关会在此 URL 后面自动拼接 <code>code</code>、<code>state</code> 与脱敏用户信息 <code>user_info</code>。
              </p>
            </div>

            <div className="space-y-1">
              <label className="font-semibold text-foreground">授权作用域 (Scope):</label>
              <div className="grid grid-cols-2 gap-2">
                <Button
                  type="button"
                  variant={authScope === 'snsapi_userinfo' ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => setAuthScope('snsapi_userinfo')}
                  className="text-xs"
                >
                  snsapi_userinfo (弹窗/带完整用户信息)
                </Button>
                <Button
                  type="button"
                  variant={authScope === 'snsapi_base' ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => setAuthScope('snsapi_base')}
                  className="text-xs"
                >
                  snsapi_base (静默获取 OpenID)
                </Button>
              </div>
            </div>

            <div className="space-y-1 pt-1">
              <div className="flex items-center justify-between">
                <span className="font-semibold text-foreground">业务端发起跳转的网关授权链接:</span>
                {currentApp && (
                  <button
                    onClick={() => copyText(generatedOAuthURL, '授权链接')}
                    className="text-primary hover:underline inline-flex items-center gap-1 font-semibold"
                  >
                    <Copy className="h-3 w-3" />
                    <span>复制链接</span>
                  </button>
                )}
              </div>
              <div className="p-3 bg-muted/70 rounded-lg font-mono text-[11px] select-all break-all border text-foreground">
                {generatedOAuthURL}
              </div>
            </div>

            {/* 📱 微信扫码授权真机测试沙箱 (核心亮点功能) */}
            {currentApp && (
              <div className="p-3.5 rounded-xl border bg-card space-y-3">
                <div className="flex items-center justify-between">
                  <div className="font-semibold text-foreground flex items-center gap-1.5">
                    <QrCode className="h-4 w-4 text-emerald-600" />
                    <span>手机微信扫码直测 (获取当前微信信息):</span>
                  </div>
                  <Badge variant="outline" className="text-[10px] text-emerald-600 border-emerald-500/30">
                    无需自建页面
                  </Badge>
                </div>

                <div className="flex flex-col sm:flex-row items-center gap-4 bg-muted/40 p-3 rounded-lg">
                  <div className="relative h-32 w-32 shrink-0 bg-background rounded-lg border p-1 shadow-sm flex items-center justify-center">
                    <img
                      src={oauthQRCodeURL}
                      alt="OAuth QR Code"
                      className="h-full w-full object-contain"
                    />
                  </div>
                  <div className="space-y-1.5 text-[11px] leading-relaxed">
                    <div className="font-semibold text-foreground flex items-center gap-1">
                      <Smartphone className="h-3.5 w-3.5 text-primary" />
                      <span>使用微信 App 扫描左侧二维码</span>
                    </div>
                    <p className="text-muted-foreground">
                      授权完成后将重定向到回调地址。若使用沙箱回显，请在微信中确认授权后，页面会直接展示您的 <code>OpenID</code>、<code>Nickname</code>、<code>头像</code> 与 <code>UnionID</code>。
                    </p>
                    <div className="pt-1 flex flex-wrap gap-2">
                      <a
                        href={generatedOAuthURL}
                        target="_blank"
                        rel="noreferrer"
                        className="inline-flex items-center gap-1 text-primary hover:underline font-semibold"
                      >
                        <ExternalLink className="h-3 w-3" />
                        <span>电脑端在浏览器中直接模拟点击</span>
                      </a>
                    </div>
                  </div>
                </div>
              </div>
            )}

            <div className="p-3 bg-emerald-500/10 border border-emerald-500/20 rounded-lg text-emerald-800 dark:text-emerald-300 space-y-1">
              <div className="font-semibold">✨ WXGate 特色直通能力:</div>
              <div>
                微信授权成功后，网关在将用户重定向回业务地址时，会在 URL 中直接附带脱敏后的安全用户信息 <code>user_info</code>（包含 openid, nickname, headimgurl 等），前端无需二次向后端请求微信换 Token，直接展示登录态！
              </div>
            </div>
          </CardContent>
        </Card>

        {/* 场景二：前端极速 JSSDK 签名与分享 */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <Share2 className="h-5 w-5 text-purple-600" />
              <span>场景二：前端极速 JSSDK 签名与分享</span>
            </CardTitle>
            <CardDescription className="text-xs">
              自动调度全局缓存的 access_token 与 jsapi_ticket，前端只需调用一次接口即可立即完成 wx.config 注入
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3.5 text-xs">
            <div className="space-y-1">
              <label className="font-semibold text-foreground">当前前端页面完整 URL:</label>
              <div className="flex gap-2">
                <Input
                  value={testUrl}
                  onChange={(e) => setTestUrl(e.target.value)}
                  placeholder="https://your-page.com/article/1"
                  className="font-mono text-xs flex-1"
                />
                <Button size="sm" onClick={handleFetchJSSDK} disabled={loadingJSSDK}>
                  {loadingJSSDK ? '计算中...' : '测试获取配置'}
                </Button>
              </div>
            </div>

            {jssdkResult && (
              <div className="space-y-1.5 p-3 rounded-lg bg-muted/60 border font-mono text-[11px]">
                <div className="font-sans font-semibold text-foreground flex items-center gap-1 text-xs">
                  <CheckCircle2 className="h-3.5 w-3.5 text-emerald-600" />
                  <span>生成的 JSSDK 签名数据:</span>
                </div>
                <div>appId: {jssdkResult.appId}</div>
                <div>timestamp: {jssdkResult.timestamp}</div>
                <div>nonceStr: {jssdkResult.nonceStr}</div>
                <div className="truncate">signature: {jssdkResult.signature}</div>
              </div>
            )}

            <div className="space-y-1">
              <div className="flex items-center justify-between">
                <span className="font-semibold text-foreground">前端页面极速接入代码:</span>
                <button
                  onClick={() =>
                    copyText(
                      `// 1. 引入网关加速代理的 JSSDK\n<script src="${window.location.origin}/jweixin.js"></script>\n\n// 2. 向网关获取配置并初始化\nconst currentUrl = encodeURIComponent(window.location.href.split('#')[0]);\nconst res = await fetch('${window.location.origin}/api/jssdk/config?url=' + currentUrl);\nconst { data } = await res.json();\nwx.config({\n  appId: data.appId,\n  timestamp: data.timestamp,\n  nonceStr: data.nonceStr,\n  signature: data.signature,\n  jsApiList: data.jsApiList\n});`,
                      'JSSDK 示例代码'
                    )
                  }
                  className="text-primary hover:underline inline-flex items-center gap-1 font-semibold"
                >
                  <Code2 className="h-3 w-3" />
                  <span>复制代码</span>
                </button>
              </div>
              <pre className="p-3 bg-muted rounded-lg font-mono text-[10.5px] overflow-x-auto max-h-36">
{`<!-- 1. 引入网关代理脚本 -->
<script src="${window.location.origin}/jweixin.js"></script>

<!-- 2. 获取配置注入 wx.config -->
const url = encodeURIComponent(window.location.href.split('#')[0]);
const res = await fetch('${window.location.origin}/api/jssdk/config?url=' + url);
const { data } = await res.json();
wx.config({ ...data });`}
              </pre>
            </div>

            {/* 缓存清理与管理 */}
            {currentApp && (
              <div className="pt-2 border-t flex items-center justify-between">
                <span className="text-muted-foreground">当前公众号缓存管理:</span>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => clearCacheMutation.mutate(currentApp.appid)}
                  className="text-destructive hover:bg-destructive/10 text-xs gap-1 h-8"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                  <span>彻底清空该公众号凭证缓存</span>
                </Button>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* 账号能力体检与白名单诊断箱 */}
      <Card>
        <CardHeader className="pb-3">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
            <div>
              <CardTitle className="text-base flex items-center gap-2">
                <HeartPulse className="h-5 w-5 text-rose-500" />
                <span>公众号接口健康度 & 白名单诊断体检箱</span>
              </CardTitle>
              <CardDescription className="text-xs">
                一键排查当前公众号 AccessToken 有效性、服务器外网出口 IP（用于配置微信后台 IP 白名单）、服务号/订阅号能力匹配
              </CardDescription>
            </div>
            <Button
              type="button"
              variant="default"
              size="sm"
              onClick={handleDiagnose}
              disabled={diagnoseLoading || !currentApp}
              className="gap-1.5 text-xs h-8"
            >
              <RefreshCw className={`h-3.5 w-3.5 ${diagnoseLoading ? 'animate-spin' : ''}`} />
              <span>{diagnoseLoading ? '体检中...' : '开始接口体检'}</span>
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-3 text-xs">
          {diagnoseResult ? (
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              <div className="p-3 bg-muted/60 rounded-xl space-y-1">
                <div className="text-muted-foreground text-[11px]">服务器外网出口 IP:</div>
                <div className="font-mono text-sm font-bold flex items-center justify-between">
                  <span>{diagnoseResult.outbound_ip || '未知'}</span>
                  <button
                    onClick={() => copyText(diagnoseResult.outbound_ip, '出口 IP')}
                    className="text-primary hover:underline text-xs flex items-center gap-0.5"
                  >
                    <Copy className="h-3 w-3" />
                    <span>复制</span>
                  </button>
                </div>
                <p className="text-[10.5px] text-muted-foreground">请将此 IP 填入微信后台「IP 白名单」</p>
              </div>

              <div className="p-3 bg-muted/60 rounded-xl space-y-1">
                <div className="text-muted-foreground text-[11px]">AccessToken 凭证连通性:</div>
                <div className="text-sm font-bold flex items-center gap-1">
                  {diagnoseResult.token_status === 'ok' ? (
                    <span className="text-emerald-600 flex items-center gap-1">
                      <CheckCircle2 className="h-4 w-4" /> 正常有效
                    </span>
                  ) : (
                    <span className="text-destructive flex items-center gap-1">
                      <ShieldAlert className="h-4 w-4" /> 获取失败
                    </span>
                  )}
                </div>
                {diagnoseResult.token_error && (
                  <p className="text-[10px] text-destructive truncate" title={diagnoseResult.token_error}>
                    {diagnoseResult.token_error}
                  </p>
                )}
              </div>

              <div className="p-3 bg-muted/60 rounded-xl space-y-1">
                <div className="text-muted-foreground text-[11px]">网页授权 (OAuth2) 权限:</div>
                <div className="text-sm font-bold">
                  {diagnoseResult.features?.can_oauth_userinfo ? (
                    <span className="text-emerald-600">已开通 (认证服务号)</span>
                  ) : (
                    <span className="text-amber-600">受限 (需微信认证服务号)</span>
                  )}
                </div>
                <p className="text-[10.5px] text-muted-foreground">普通订阅号仅支持静默基础授权</p>
              </div>

              <div className="p-3 bg-muted/60 rounded-xl space-y-1">
                <div className="text-muted-foreground text-[11px]">公众号模板消息权限:</div>
                <div className="text-sm font-bold">
                  {diagnoseResult.features?.can_template_msg ? (
                    <span className="text-emerald-600">支持服务号模板</span>
                  ) : (
                    <span className="text-muted-foreground">仅支持订阅消息</span>
                  )}
                </div>
                <p className="text-[10.5px] text-muted-foreground">订阅号建议使用一次性订阅消息</p>
              </div>
            </div>
          ) : (
            <div className="p-4 rounded-xl border border-dashed text-center text-muted-foreground">
              点击上方「开始接口体检」按钮，一键快速诊断当前公众号各项能力与网络连通性。
            </div>
          )}
        </CardContent>
      </Card>

      {/* 原生接口代理端点指引 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <Layers className="h-5 w-5 text-blue-600" />
            <span>场景三：子系统原生微信 API 代理（统一凭证中心）</span>
          </CardTitle>
          <CardDescription className="text-xs">
            其它业务后端系统直接请求网关地址，网关内置智能缓存与自动刷新，彻底解决不同系统重复获取 Token 互相踢下线的致命问题
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3 text-xs">
          <div className="grid gap-3 sm:grid-cols-3">
            <div className="p-3 bg-muted/60 rounded-xl space-y-1">
              <div className="font-semibold text-foreground">全局 AccessToken 代理</div>
              <div className="font-mono text-[11px] text-primary select-all">/cgi-bin/token</div>
              <p className="text-[11px] text-muted-foreground">带两级缓存，提前 5 分钟换新，完全兼容微信官方入参及返回</p>
            </div>

            <div className="p-3 bg-muted/60 rounded-xl space-y-1">
              <div className="font-semibold text-foreground">稳定版 Token 代理</div>
              <div className="font-mono text-[11px] text-primary select-all">/cgi-bin/stable_token</div>
              <p className="text-[11px] text-muted-foreground">微信官方稳定版 Token 协议透明代理与自动缓存</p>
            </div>

            <div className="p-3 bg-muted/60 rounded-xl space-y-1">
              <div className="font-semibold text-foreground">JSAPI Ticket 代理</div>
              <div className="font-mono text-[11px] text-primary select-all">/cgi-bin/ticket/getticket</div>
              <p className="text-[11px] text-muted-foreground">自动调度底层 Token 并维护 Ticket 有效期</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
