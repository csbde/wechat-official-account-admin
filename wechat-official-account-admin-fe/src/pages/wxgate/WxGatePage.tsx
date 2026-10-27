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
  const generatedOAuthURL = currentApp
    ? `${gatewayBase}/connect/oauth2/authorize?appid=${currentApp.appid}&redirect_uri=${encodeURIComponent(
        targetRedirect
      )}&response_type=code&scope=${authScope}&state=STATE#wechat_redirect`
    : '请先在顶部选择一个微信公众号'

  return (
    <div className="space-y-6">
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
      <div className="grid gap-4 md:grid-cols-3">
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

            <div className="space-y-1.5">
              <div className="flex items-center justify-between">
                <label className="font-semibold text-foreground">目标业务系统实际回调地址 (redirect_uri):</label>
                <div className="flex items-center gap-1.5">
                  <button
                    type="button"
                    onClick={() => setTargetRedirect(`${window.location.origin}/oauth/callback`)}
                    className="text-[11px] text-primary hover:underline flex items-center gap-0.5"
                  >
                    <Compass className="h-3 w-3" />
                    <span>当前站点</span>
                  </button>
                  <span className="text-muted-foreground/40">|</span>
                  <button
                    type="button"
                    onClick={() => setTargetRedirect('https://app.yourcompany.com/login/callback')}
                    className="text-[11px] text-primary hover:underline"
                  >
                    子域名系统
                  </button>
                  <span className="text-muted-foreground/40">|</span>
                  <button
                    type="button"
                    onClick={() => setTargetRedirect('http://localhost:3000/callback')}
                    className="text-[11px] text-muted-foreground hover:text-foreground flex items-center gap-0.5"
                  >
                    <Laptop className="h-3 w-3" />
                    <span>本地联调</span>
                  </button>
                </div>
              </div>

              <Input
                value={targetRedirect}
                onChange={(e) => setTargetRedirect(e.target.value)}
                placeholder="例如: https://sub.yourcompany.com/callback 或 http://localhost:3000/callback"
                className="font-mono text-xs"
              />
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
