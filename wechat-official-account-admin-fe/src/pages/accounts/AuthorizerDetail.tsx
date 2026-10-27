import React from 'react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { DialogFooter } from '@/components/ui/dialog'
import {
  Building2, CheckCircle2, XCircle, QrCode, ShieldCheck, AppWindow,
  Globe2, KeyRound, AlertTriangle, ScrollText,
} from 'lucide-react'

/** 微信开通功能开关字典 (business_info，0 未开通 / 1 已开通) */
const BUSINESS_FLAGS: { key: string; label: string }[] = [
  { key: 'open_pay', label: '微信支付' },
  { key: 'open_shake', label: '微信摇一摇' },
  { key: 'open_scan', label: '扫商品' },
  { key: 'open_card', label: '微信卡券' },
  { key: 'open_store', label: '微信门店' },
]

/** 小程序合法域名字段 (MiniProgramInfo.network) */
const NETWORK_LABELS: Record<string, string> = {
  RequestDomain: 'request 合法域名',
  WsRequestDomain: 'socket 合法域名',
  UploadDomain: 'uploadFile 合法域名',
  DownloadDomain: 'downloadFile 合法域名',
  UDPDomain: 'udp 合法域名',
  TCPDomain: 'tcp 合法域名',
}

/** 单行信息格: 空值统一显示为「未提供」，避免看起来像页面坏了 */
const Cell: React.FC<{ label: string; value?: any; mono?: boolean; danger?: boolean }> = ({
  label, value, mono, danger,
}) => {
  const empty = value === undefined || value === null || value === '' ||
    (Array.isArray(value) && value.length === 0)
  return (
    <div className="p-2.5 rounded-lg border bg-card space-y-0.5 min-w-0">
      <div className="text-[11px] text-muted-foreground">{label}</div>
      {empty ? (
        <div className="text-[12px] text-muted-foreground/70">未提供</div>
      ) : (
        <div className={`text-[12px] break-words ${mono ? 'font-mono text-[11px]' : 'font-semibold'} ${danger ? 'text-red-600' : 'text-foreground'}`}>
          {String(value)}
        </div>
      )}
    </div>
  )
}

const YesNo: React.FC<{ ok: boolean; text: string }> = ({ ok, text }) => (
  <span className={`inline-flex items-center gap-1 text-[11px] ${ok ? 'text-emerald-600' : 'text-muted-foreground'}`}>
    {ok ? <CheckCircle2 className="h-3.5 w-3.5" /> : <XCircle className="h-3.5 w-3.5" />}
    {text}
  </span>
)

/** 类型名称: 微信官方会同时返回 id 与 name，优先展示官方 name */
const typeName = (t: any) => (t?.name && String(t.name).trim()) || (t?.id !== undefined ? `#${t.id}` : '')

/** 微信官方权威档案详情视图 */
export const AuthorizerDetailView: React.FC<{ detail: any; onClose: () => void }> = ({ detail, onClose }) => {
  const acc = detail?.account || {}
  const info = detail?.authorizer_info || {}
  const funcInfo: any[] = detail?.func_info || []
  const biz = info.business_info || {}
  const basic = info.basic_config || {}
  const mp = info.MiniProgramInfo || info.mini_program_info || null
  const isOfficial = detail?.source === 'wechat_official'

  const on = (v: any) => Number(v) === 1

  return (
    <div className="space-y-4 py-1 text-xs">
      {/* 数据来源提示 */}
      {!isOfficial && (
        <div className="p-3 rounded-xl bg-amber-500/10 border border-amber-500/25 text-amber-900 dark:text-amber-200 space-y-1">
          <div className="font-semibold flex items-center gap-1.5">
            <AlertTriangle className="h-4 w-4" />当前展示的是本系统落库档案 (未取到微信官方实时数据)
          </div>
          {detail?.official_error && (
            <div className="break-words leading-relaxed">微信接口返回: {String(detail.official_error)}</div>
          )}
          {detail?.hint && <div className="leading-relaxed break-words">{String(detail.hint)}</div>}
        </div>
      )}

      {/* 基本信息头部 */}
      <div className="p-4 rounded-xl bg-muted/60 border flex items-start gap-4">
        <div className="h-16 w-16 rounded-2xl overflow-hidden shrink-0 border bg-background shadow-sm">
          {info.head_img ? (
            <img src={info.head_img} referrerPolicy="no-referrer" alt="" className="h-full w-full object-cover" />
          ) : (
            <div className="h-full w-full bg-primary/20 text-primary flex items-center justify-center font-bold text-lg">
              {(info.nick_name || acc.name || 'W').slice(0, 1)}
            </div>
          )}
        </div>
        <div className="space-y-1 flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="font-bold text-base text-foreground break-all">
              {info.nick_name || acc.name || '(未取名)'}
            </span>
            <Badge variant={acc.account_type === 'authorized' ? 'success' : 'outline'} className="text-[10px]">
              {acc.account_type === 'authorized' ? '第三方平台托管' : '直连凭据'}
            </Badge>
            <Badge variant="secondary" className="text-[10px]">
              {info.service_type_info?.name || (info.MiniProgramInfo ? '小程序' : '公众号')}
            </Badge>
            {isOfficial ? (
              <Badge variant="outline" className="text-[10px] gap-1">
                <ShieldCheck className="h-3 w-3" />微信官方实时
              </Badge>
            ) : (
              <Badge variant="outline" className="text-[10px]">本地落库</Badge>
            )}
          </div>
          <div className="font-mono text-muted-foreground text-[11px] break-all">AppID: {acc.appid || acc.app_id || '-'}</div>
          {info.user_name && (
            <div className="font-mono text-muted-foreground text-[11px] break-all">原始 ID: {info.user_name}</div>
          )}
          {info.alias && (
            <div className="font-mono text-muted-foreground text-[11px] break-all">微信号: {info.alias}</div>
          )}
        </div>
      </div>

      {/* 认证与主体 */}
      <div className="grid grid-cols-2 gap-2.5">
        <Cell label="认证主体 (principal_name)" value={info.principal_name || '个人 / 未认证主体'} />
        <Cell label="微信认证类型 (verify_type_info)" value={typeName(info.verify_type_info)} />
        <Cell label="账号类型 ID (service_type_info)" value={info.service_type_info?.id !== undefined ? `#${info.service_type_info.id}` : ''} />
        <Cell
          label="刷新令牌 (托管状态)"
          value={acc.authorizer_refresh_token ? '已安全托管，可代调用接口' : ''}
        />
      </div>

      {/* 功能开通情况 */}
      {(BUSINESS_FLAGS.some((b) => biz[b.key] !== undefined) || isOfficial) && (
        <div className="p-3 rounded-lg border bg-card space-y-2">
          <div className="font-semibold text-foreground flex items-center gap-1.5">
            <KeyRound className="h-3.5 w-3.5 text-primary" />功能开通状况 (business_info)
          </div>
          <div className="flex flex-wrap gap-x-4 gap-y-1.5">
            {BUSINESS_FLAGS.map((b) => (
              <YesNo key={b.key} ok={on(biz[b.key])} text={b.label} />
            ))}
            {basic.is_phone_configured !== undefined && (
              <YesNo ok={!!basic.is_phone_configured} text="绑定手机号" />
            )}
            {basic.is_email_configured !== undefined && (
              <YesNo ok={!!basic.is_email_configured} text="绑定邮箱" />
            )}
          </div>
        </div>
      )}

      {/* 功能介绍 */}
      {info.signature && (
        <div className="p-3 rounded-lg border bg-card space-y-1">
          <div className="text-[11px] text-muted-foreground">功能介绍 (signature)</div>
          <div className="text-foreground leading-relaxed break-words">{info.signature}</div>
        </div>
      )}

      {/* 账号二维码 */}
      {(info.qrcode_url || acc.qrcode_url) && (
        <div className="p-3 rounded-lg border bg-card flex items-center gap-3">
          <img
            src={info.qrcode_url || acc.qrcode_url}
            referrerPolicy="no-referrer"
            alt="账号二维码"
            className="h-20 w-20 rounded border bg-background object-contain shrink-0"
          />
          <div className="space-y-1 min-w-0">
            <div className="font-semibold flex items-center gap-1.5">
              <QrCode className="h-3.5 w-3.5 text-primary" />微信官方账号二维码
            </div>
            <div className="text-[11px] text-muted-foreground break-all font-mono">
              {info.qrcode_url || acc.qrcode_url}
            </div>
          </div>
        </div>
      )}

      {/* 小程序附加档案 */}
      {mp && (
        <div className="p-3 rounded-lg border bg-card space-y-2">
          <div className="font-semibold text-foreground flex items-center gap-1.5">
            <AppWindow className="h-3.5 w-3.5 text-primary" />小程序配置信息 (MiniProgramInfo)
          </div>
          {Array.isArray(mp.categories) && mp.categories.length > 0 && (
            <div className="flex flex-wrap gap-1.5">
              {mp.categories.map((c: any, i: number) => (
                <Badge key={i} variant="outline" className="text-[10px]">
                  {[c.first, c.second, c.third].filter(Boolean).join(' / ')}
                </Badge>
              ))}
            </div>
          )}
          {mp.network && (
            <div className="space-y-1.5">
              {Object.entries(mp.network as Record<string, any>).map(([k, v]) =>
                Array.isArray(v) ? (
                  <div key={k} className="min-w-0">
                    <div className="text-[11px] text-muted-foreground">
                      <Globe2 className="h-3 w-3 inline mr-1" />
                      {NETWORK_LABELS[k] || k}: {v.length} 个
                    </div>
                    {v.length > 0 && (
                      <div className="font-mono text-[10px] text-foreground/80 break-all">
                        {(v as string[]).slice(0, 6).join(' , ')}
                        {v.length > 6 ? ` …共 ${v.length} 个` : ''}
                      </div>
                    )}
                  </div>
                ) : null
              )}
            </div>
          )}
          {(info.register_type !== undefined || info.account_status !== undefined) && (
            <div className="text-[11px] text-muted-foreground">
              register_type: {String(info.register_type ?? '-')} · account_status: {String(info.account_status ?? '-')}
            </div>
          )}
        </div>
      )}

      {/* 已授权权限集 */}
      <div className="space-y-1.5 p-3 rounded-lg border bg-card">
        <div className="font-semibold text-foreground flex items-center justify-between gap-2">
          <span>授权给本平台的权限集 (func_info)</span>
          <Badge variant="outline" className="text-[10px] shrink-0">{funcInfo.length} 项</Badge>
        </div>
        {funcInfo.length === 0 ? (
          <div className="text-[11px] text-muted-foreground">
            暂无权限集数据。仅「第三方平台扫码授权托管」的账号才会返回 func_info；若该账号刚完成授权，请重新点击授权同步一次。
          </div>
        ) : (
          <div className="flex flex-wrap gap-1.5 pt-1 max-h-40 overflow-y-auto">
            {funcInfo.map((f: any, idx: number) => {
              const id = f?.funcscope_category?.id
              const cf = f?.confirm_info
              return (
                <Badge key={idx} variant="secondary" className="text-[10px] py-0.5 font-mono">
                  权限集 #{String(id ?? '?')}
                  {cf && cf.need_confirm === 1 && (
                    <span className={cf.already_confirm === 1 ? 'text-emerald-600' : 'text-amber-600'}>
                      {cf.already_confirm === 1 ? ' · 已确认' : ' · 待管理员确认'}
                    </span>
                  )}
                </Badge>
              )
            })}
          </div>
        )}
        <div className="text-[10px] text-muted-foreground leading-relaxed">
          权限集编号即开放平台后台该第三方平台的「权限集」配置项 (例如代公众号群发消息为 7、素材与用户管理等各有独立编号)，具体含义请以开放平台后台勾选的名称为准。
        </div>
      </div>

      {/* 原始响应 (排障用) */}
      {detail?.raw && (
        <details className="rounded-lg border bg-card">
          <summary className="cursor-pointer p-3 text-[11px] font-semibold text-muted-foreground flex items-center gap-1.5">
            <ScrollText className="h-3.5 w-3.5" />微信官方原始响应 (排障用)
          </summary>
          <pre className="px-3 pb-3 text-[10px] font-mono whitespace-pre-wrap break-all max-h-64 overflow-y-auto">
            {JSON.stringify(detail.raw, null, 2)}
          </pre>
        </details>
      )}

      <DialogFooter>
        <Button size="sm" onClick={onClose}>关闭</Button>
      </DialogFooter>
    </div>
  )
}
