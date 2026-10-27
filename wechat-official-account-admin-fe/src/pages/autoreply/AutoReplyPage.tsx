import React, { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { useAppStore } from '@/stores/app'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  MessageSquareReply,
  Plus,
  Trash2,
  Edit,
  Tag,
  CheckCircle,
  HelpCircle,
} from 'lucide-react'
import { toast } from 'sonner'

export const AutoReplyPage: React.FC = () => {
  const queryClient = useQueryClient()
  const { currentApp } = useAppStore()

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<any>(null)

  // 表单状态
  const [ruleName, setRuleName] = useState('')
  const [ruleType, setRuleType] = useState('keyword') // subscribe, default, keyword
  const [replyType, setReplyType] = useState('text') // text, image
  const [content, setContent] = useState('')
  const [keywords, setKeywords] = useState<{ keyword: string; match_type: 'full' | 'half' }[]>([])
  const [inputKw, setInputKw] = useState('')
  const [inputKwMatch, setInputKwMatch] = useState<'full' | 'half'>('full')

  // 获取规则列表
  const { data: rules, isLoading } = useQuery<any[]>({
    queryKey: ['autoreply', currentApp?.appid],
    queryFn: () => api.get('/api/autoreply'),
    enabled: !!currentApp,
  })

  // 保存规则
  const saveMutation = useMutation({
    mutationFn: (payload: any) => api.post('/api/autoreply', payload),
    onSuccess: () => {
      toast.success('自动回复规则保存成功')
      queryClient.invalidateQueries({ queryKey: ['autoreply'] })
      setDialogOpen(false)
    },
  })

  // 删除规则
  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/autoreply/${id}`),
    onSuccess: () => {
      toast.success('规则已删除')
      queryClient.invalidateQueries({ queryKey: ['autoreply'] })
    },
  })

  // 切换开关
  const toggleMutation = useMutation({
    mutationFn: (id: number) => api.put(`/api/autoreply/${id}/toggle`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['autoreply'] })
    },
  })

  const openCreateModal = (type: string = 'keyword') => {
    setEditingRule(null)
    setRuleType(type)
    setRuleName(type === 'subscribe' ? '关注后自动回复' : type === 'default' ? '消息收到默认回复' : '新关键词规则')
    setReplyType('text')
    setContent('')
    setKeywords([])
    setInputKw('')
    setDialogOpen(true)
  }

  const openEditModal = (rule: any) => {
    setEditingRule(rule)
    setRuleType(rule.rule_type)
    setRuleName(rule.rule_name)
    setReplyType(rule.reply_type)
    setContent(rule.content || '')
    try {
      setKeywords(JSON.parse(rule.keywords || '[]'))
    } catch {
      setKeywords([])
    }
    setDialogOpen(true)
  }

  const addKeyword = () => {
    if (!inputKw.trim()) return
    setKeywords([...keywords, { keyword: inputKw.trim(), match_type: inputKwMatch }])
    setInputKw('')
  }

  const removeKeyword = (idx: number) => {
    setKeywords(keywords.filter((_, i) => i !== idx))
  }

  const handleSave = () => {
    if (!ruleName.trim()) {
      toast.error('请输入规则名称')
      return
    }
    if (!content.trim()) {
      toast.error('请输入回复内容')
      return
    }
    if (ruleType === 'keyword' && keywords.length === 0) {
      toast.error('关键词回复至少需要添加一个关键词')
      return
    }

    const payload = {
      id: editingRule?.id || 0,
      rule_name: ruleName,
      rule_type: ruleType,
      reply_type: replyType,
      content,
      keywords: JSON.stringify(keywords),
      enabled: editingRule ? editingRule.enabled : true,
    }

    saveMutation.mutate(payload)
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">智能自动回复</h2>
          <p className="text-sm text-muted-foreground">
            配置关注后欢迎语、收到普通消息默认应答及多关键词智能匹配回复
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button onClick={() => openCreateModal('keyword')} className="gap-2">
            <Plus className="h-4 w-4" />
            <span>新建关键词规则</span>
          </Button>
        </div>
      </div>

      {/* 快捷类型引导卡片 */}
      <div className="grid gap-4 md:grid-cols-2">
        {/* 关注回复卡片 */}
        {(() => {
          const subRule = rules?.find((r) => r.rule_type === 'subscribe')
          return (
            <Card className="border-l-4 border-l-emerald-500">
              <CardHeader className="pb-3 flex flex-row items-center justify-between">
                <div>
                  <CardTitle className="text-base flex items-center gap-2">
                    <span>关注后自动回复 (Subscribe)</span>
                    {subRule && <Badge variant={subRule.enabled ? 'success' : 'secondary'}>{subRule.enabled ? '启用中' : '已停用'}</Badge>}
                  </CardTitle>
                </div>
                {subRule ? (
                  <Button variant="outline" size="sm" onClick={() => openEditModal(subRule)}>编辑</Button>
                ) : (
                  <Button size="sm" onClick={() => openCreateModal('subscribe')}>配置关注回复</Button>
                )}
              </CardHeader>
              <CardContent className="text-xs text-muted-foreground">
                {subRule ? (
                  <div className="p-3 bg-muted/60 rounded-lg text-foreground whitespace-pre-wrap">{subRule.content}</div>
                ) : (
                  <div>用户微信扫码或搜索关注公众号时自动推送的第一条欢迎消息。</div>
                )}
              </CardContent>
            </Card>
          )
        })()}

        {/* 默认回复卡片 */}
        {(() => {
          const defRule = rules?.find((r) => r.rule_type === 'default')
          return (
            <Card className="border-l-4 border-l-blue-500">
              <CardHeader className="pb-3 flex flex-row items-center justify-between">
                <div>
                  <CardTitle className="text-base flex items-center gap-2">
                    <span>收到消息默认回复 (Default)</span>
                    {defRule && <Badge variant={defRule.enabled ? 'success' : 'secondary'}>{defRule.enabled ? '启用中' : '已停用'}</Badge>}
                  </CardTitle>
                </div>
                {defRule ? (
                  <Button variant="outline" size="sm" onClick={() => openEditModal(defRule)}>编辑</Button>
                ) : (
                  <Button size="sm" onClick={() => openCreateModal('default')}>配置默认回复</Button>
                )}
              </CardHeader>
              <CardContent className="text-xs text-muted-foreground">
                {defRule ? (
                  <div className="p-3 bg-muted/60 rounded-lg text-foreground whitespace-pre-wrap">{defRule.content}</div>
                ) : (
                  <div>当粉丝发送的文本消息未命中任何关键词时的统一人工提示或指引。</div>
                )}
              </CardContent>
            </Card>
          )
        })()}
      </div>

      {/* 关键词规则列表 */}
      <div className="space-y-4">
        <h3 className="text-lg font-bold">关键词匹配规则列表</h3>
        {isLoading ? (
          <div className="py-8 text-center text-muted-foreground">加载中...</div>
        ) : rules && rules.filter((r) => r.rule_type === 'keyword').length > 0 ? (
          <div className="grid gap-4">
            {rules
              .filter((r) => r.rule_type === 'keyword')
              .map((rule) => {
                let parsedKw: any[] = []
                try {
                  parsedKw = JSON.parse(rule.keywords || '[]')
                } catch {}

                return (
                  <Card key={rule.id} className="p-4 flex flex-col md:flex-row md:items-center justify-between gap-4">
                    <div className="space-y-2 flex-1">
                      <div className="flex items-center gap-3">
                        <span className="font-bold text-base">{rule.rule_name}</span>
                        <Badge variant="outline" className="text-xs">
                          {rule.reply_type === 'text' ? '纯文本' : '多媒体素材'}
                        </Badge>
                      </div>

                      {/* 关键词标签 */}
                      <div className="flex flex-wrap gap-1.5 items-center">
                        <span className="text-xs text-muted-foreground mr-1">触发关键词:</span>
                        {parsedKw.map((k, idx) => (
                          <Badge key={idx} variant="secondary" className="text-xs gap-1 font-mono">
                            <span>{k.keyword}</span>
                            <span className="text-[10px] text-muted-foreground">
                              ({k.match_type === 'full' ? '精准' : '半匹配'})
                            </span>
                          </Badge>
                        ))}
                      </div>

                      {/* 回复内容摘要 */}
                      <div className="text-xs text-muted-foreground bg-muted/40 p-2.5 rounded-lg">
                        {rule.content}
                      </div>
                    </div>

                    {/* 操作 */}
                    <div className="flex items-center gap-3 shrink-0 self-end md:self-center">
                      <div className="flex items-center gap-2">
                        <span className="text-xs text-muted-foreground">启用</span>
                        <Switch
                          checked={rule.enabled}
                          onCheckedChange={() => toggleMutation.mutate(rule.id)}
                        />
                      </div>
                      <Button variant="ghost" size="icon" onClick={() => openEditModal(rule)}>
                        <Edit className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => {
                          if (confirm('确认删除此规则？')) {
                            deleteMutation.mutate(rule.id)
                          }
                        }}
                        className="text-destructive hover:bg-destructive/10"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </Card>
                )
              })}
          </div>
        ) : (
          <Card className="p-8 text-center text-muted-foreground">
            暂无关键词回复规则，点击上方「新建关键词规则」开始配置
          </Card>
        )}
      </div>

      {/* 新增/编辑 Modal */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>
              {editingRule ? '编辑自动回复规则' : ruleType === 'keyword' ? '新建关键词规则' : '设置系统回复'}
            </DialogTitle>
          </DialogHeader>

          <div className="space-y-4 py-2 text-sm">
            <div className="space-y-1.5">
              <label className="text-xs font-semibold">规则名称 *</label>
              <Input
                value={ruleName}
                onChange={(e) => setRuleName(e.target.value)}
                placeholder="例如: 常见问题咨询、迎新指南"
              />
            </div>

            {ruleType === 'keyword' && (
              <div className="space-y-2">
                <label className="text-xs font-semibold">添加触发关键词 *</label>
                <div className="flex gap-2">
                  <Input
                    value={inputKw}
                    onChange={(e) => setInputKw(e.target.value)}
                    placeholder="输入触发词"
                    className="flex-1"
                    onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), addKeyword())}
                  />
                  <Select value={inputKwMatch} onValueChange={(v: any) => setInputKwMatch(v)}>
                    <SelectTrigger className="w-28">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="full">全匹配(精准)</SelectItem>
                      <SelectItem value="half">半匹配(包含)</SelectItem>
                    </SelectContent>
                  </Select>
                  <Button type="button" variant="secondary" onClick={addKeyword}>
                    添加
                  </Button>
                </div>

                <div className="flex flex-wrap gap-1.5 pt-1 min-h-[32px]">
                  {keywords.map((k, idx) => (
                    <Badge key={idx} variant="secondary" className="gap-1 pr-1 text-xs">
                      <span>{k.keyword}</span>
                      <span className="text-[10px] text-muted-foreground">[{k.match_type === 'full' ? '精准' : '半匹配'}]</span>
                      <button
                        type="button"
                        onClick={() => removeKeyword(idx)}
                        className="hover:text-destructive text-muted-foreground ml-1"
                      >
                        ×
                      </button>
                    </Badge>
                  ))}
                  {keywords.length === 0 && (
                    <span className="text-xs text-muted-foreground">尚未添加任何关键词</span>
                  )}
                </div>
              </div>
            )}

            <div className="space-y-1.5">
              <label className="text-xs font-semibold">回复文本内容 *</label>
              <textarea
                value={content}
                onChange={(e) => setContent(e.target.value)}
                rows={4}
                className="w-full rounded-lg border border-input bg-background p-3 text-sm focus:outline-none focus:ring-1 focus:ring-ring"
                placeholder="输入回复给微信用户的文本内容..."
              />
            </div>

            <DialogFooter className="pt-2">
              <Button variant="outline" onClick={() => setDialogOpen(false)}>取消</Button>
              <Button onClick={handleSave} disabled={saveMutation.isPending}>
                {saveMutation.isPending ? '保存中...' : '确定保存'}
              </Button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
