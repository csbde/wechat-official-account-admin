import React, { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { useAppStore } from '@/stores/app'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
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
  Bot,
  Sparkles,
  Zap,
  Send,
  Search,
  CheckCircle2,
} from 'lucide-react'
import { toast } from 'sonner'

export const AutoReplyPage: React.FC = () => {
  const queryClient = useQueryClient()
  const { currentApp } = useAppStore()

  const [activeTab, setActiveTab] = useState<'rules' | 'ai'>('rules')
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<any>(null)
  const [searchKw, setSearchKw] = useState('')

  // 规则表单状态
  const [ruleName, setRuleName] = useState('')
  const [ruleType, setRuleType] = useState('keyword')
  const [replyType, setReplyType] = useState('text')
  const [content, setContent] = useState('')
  const [keywords, setKeywords] = useState<{ keyword: string; match_type: 'full' | 'half' }[]>([])
  const [inputKw, setInputKw] = useState('')
  const [inputKwMatch, setInputKwMatch] = useState<'full' | 'half'>('full')

  // AI 助手配置表单状态
  const [aiEnabled, setAiEnabled] = useState(false)
  const [aiProvider, setAiProvider] = useState('openai')
  const [aiBaseUrl, setAiBaseUrl] = useState('https://api.openai.com/v1')
  const [aiApiKey, setAiApiKey] = useState('')
  const [aiModel, setAiModel] = useState('gpt-4o-mini')
  const [aiSystemPrompt, setAiSystemPrompt] = useState('你是该公众号的官方客服智能助手，请用简洁、亲切的中文回答用户咨询。')
  const [aiMatchMode, setAiMatchMode] = useState('fallback')
  const [aiTimeout, setAiTimeout] = useState(4)

  // AI 调试测试
  const [testPrompt, setTestPrompt] = useState('请问你们公众号主要提供哪些服务？')
  const [testResponse, setTestResponse] = useState('')
  const [testingAI, setTestingAI] = useState(false)

  // 获取规则列表
  const { data: rules, isLoading } = useQuery<any[]>({
    queryKey: ['autoreply', currentApp?.appid],
    queryFn: () => api.get('/api/autoreply'),
    enabled: !!currentApp,
  })

  // 获取 AI 配置
  const { data: aiSetting } = useQuery<any>({
    queryKey: ['autoreply-ai', currentApp?.appid],
    queryFn: () => api.get('/api/autoreply/ai'),
    enabled: !!currentApp,
  })

  useEffect(() => {
    if (aiSetting) {
      setAiEnabled(!!aiSetting.enabled)
      setAiProvider(aiSetting.provider || 'openai')
      setAiBaseUrl(aiSetting.base_url || 'https://api.openai.com/v1')
      setAiApiKey(aiSetting.api_key || '')
      setAiModel(aiSetting.model || 'gpt-4o-mini')
      setAiSystemPrompt(aiSetting.system_prompt || '')
      setAiMatchMode(aiSetting.match_mode || 'fallback')
      setAiTimeout(aiSetting.timeout_sec || 4)
    }
  }, [aiSetting])

  // 保存规则
  const saveMutation = useMutation({
    mutationFn: (payload: any) => api.post('/api/autoreply', payload),
    onSuccess: () => {
      toast.success('自动回复规则保存成功')
      queryClient.invalidateQueries({ queryKey: ['autoreply'] })
      setDialogOpen(false)
    },
  })

  // 保存 AI 配置
  const saveAiMutation = useMutation({
    mutationFn: (payload: any) => api.post('/api/autoreply/ai', payload),
    onSuccess: () => {
      toast.success('AI 智能助手配置已保存！')
      queryClient.invalidateQueries({ queryKey: ['autoreply-ai'] })
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

  // 服务商预设切换
  const handleProviderChange = (provider: string) => {
    setAiProvider(provider)
    switch (provider) {
      case 'deepseek':
        setAiBaseUrl('https://api.deepseek.com/v1')
        setAiModel('deepseek-chat')
        break
      case 'qwen':
        setAiBaseUrl('https://dashscope.aliyuncs.com/compatible-mode/v1')
        setAiModel('qwen-plus')
        break
      case 'moonshot':
        setAiBaseUrl('https://api.moonshot.cn/v1')
        setAiModel('moonshot-v1-8k')
        break
      case 'ollama':
        setAiBaseUrl('http://localhost:11434/v1')
        setAiModel('llama3.2')
        break
      case 'openai':
      default:
        setAiBaseUrl('https://api.openai.com/v1')
        setAiModel('gpt-4o-mini')
        break
    }
  }

  // 连通性测试
  const handleTestAI = async () => {
    if (!testPrompt.trim()) {
      toast.error('请输入测试问题')
      return
    }
    try {
      setTestingAI(true)
      const res: any = await api.post('/api/autoreply/ai/test', {
        base_url: aiBaseUrl,
        api_key: aiApiKey,
        model: aiModel,
        prompt: testPrompt,
      })
      setTestResponse(res?.reply || '测试成功，但 AI 未返回文字')
      toast.success('AI 接口连通正常！')
    } catch (e: any) {
      toast.error(e.message || '测试失败，请检查 BaseURL 或 APIKey')
    } finally {
      setTestingAI(false)
    }
  }

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

  // 过滤关键词
  const filteredKeywordRules = (rules || [])
    .filter((r) => r.rule_type === 'keyword')
    .filter((r) => {
      if (!searchKw.trim()) return true
      return r.rule_name.includes(searchKw) || r.keywords.includes(searchKw) || r.content.includes(searchKw)
    })

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold tracking-tight flex items-center gap-2">
            <span>智能自动回复 & AI 机器人</span>
            <Badge variant="outline" className="text-xs">ChatGPT 兼容</Badge>
          </h2>
          <p className="text-sm text-muted-foreground">
            配置关注欢迎语、关键词规则回复，并可一键接入大语言模型 (ChatGPT / DeepSeek 等)
          </p>
        </div>

        {activeTab === 'rules' && (
          <Button onClick={() => openCreateModal('keyword')} className="gap-2 bg-emerald-600 hover:bg-emerald-700">
            <Plus className="h-4 w-4" />
            <span>新建关键词规则</span>
          </Button>
        )}
      </div>

      {/* Tabs 分组 */}
      <Tabs value={activeTab} onValueChange={(v: any) => setActiveTab(v)}>
        <TabsList className="grid grid-cols-2 max-w-xs">
          <TabsTrigger value="rules" className="text-xs gap-1.5">
            <MessageSquareReply className="h-3.5 w-3.5" />
            <span>固定规则回复</span>
          </TabsTrigger>
          <TabsTrigger value="ai" className="text-xs gap-1.5">
            <Bot className="h-3.5 w-3.5 text-purple-600" />
            <span>AI 智能助手 (ChatGPT)</span>
          </TabsTrigger>
        </TabsList>

        {/* 规则回复标签页 */}
        <TabsContent value="rules" className="mt-6 space-y-6">
          {/* 快捷引导卡片 */}
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
                        {subRule && (
                          <Badge variant={subRule.enabled ? 'success' : 'secondary'}>
                            {subRule.enabled ? '启用中' : '已停用'}
                          </Badge>
                        )}
                      </CardTitle>
                    </div>
                    {subRule ? (
                      <div className="flex items-center gap-1">
                        <Button variant="outline" size="sm" onClick={() => openEditModal(subRule)}>编辑</Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => {
                            if (confirm('确认删除关注回复？')) deleteMutation.mutate(subRule.id)
                          }}
                          className="h-8 w-8 text-destructive hover:bg-destructive/10"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      </div>
                    ) : (
                      <Button size="sm" onClick={() => openCreateModal('subscribe')}>配置关注回复</Button>
                    )}
                  </CardHeader>
                  <CardContent className="text-xs text-muted-foreground">
                    {subRule ? (
                      <div className="p-3 bg-muted/60 rounded-lg text-foreground whitespace-pre-wrap">{subRule.content}</div>
                    ) : (
                      <div>用户首次扫码或搜索关注时自动推送的第一条欢迎消息。</div>
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
                        {defRule && (
                          <Badge variant={defRule.enabled ? 'success' : 'secondary'}>
                            {defRule.enabled ? '启用中' : '已停用'}
                          </Badge>
                        )}
                      </CardTitle>
                    </div>
                    {defRule ? (
                      <div className="flex items-center gap-1">
                        <Button variant="outline" size="sm" onClick={() => openEditModal(defRule)}>编辑</Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => {
                            if (confirm('确认删除默认回复？')) deleteMutation.mutate(defRule.id)
                          }}
                          className="h-8 w-8 text-destructive hover:bg-destructive/10"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      </div>
                    ) : (
                      <Button size="sm" onClick={() => openCreateModal('default')}>配置默认回复</Button>
                    )}
                  </CardHeader>
                  <CardContent className="text-xs text-muted-foreground">
                    {defRule ? (
                      <div className="p-3 bg-muted/60 rounded-lg text-foreground whitespace-pre-wrap">{defRule.content}</div>
                    ) : (
                      <div>未命中任何关键词且未开启 AI 时的统一人工提示或指引。</div>
                    )}
                  </CardContent>
                </Card>
              )
            })()}
          </div>

          {/* 关键词规则列表 */}
          <div className="space-y-4">
            <div className="flex items-center justify-between gap-3">
              <h3 className="text-lg font-bold">关键词匹配规则列表</h3>
              <div className="relative w-64">
                <Search className="absolute left-2.5 top-2.5 h-3.5 w-3.5 text-muted-foreground" />
                <Input
                  value={searchKw}
                  onChange={(e) => setSearchKw(e.target.value)}
                  placeholder="搜索规则或关键词..."
                  className="pl-8 text-xs h-8"
                />
              </div>
            </div>

            {isLoading ? (
              <div className="py-8 text-center text-muted-foreground">加载中...</div>
            ) : filteredKeywordRules.length > 0 ? (
              <div className="grid gap-4">
                {filteredKeywordRules.map((rule) => {
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

                        <div className="flex flex-wrap gap-1.5 items-center">
                          <span className="text-xs text-muted-foreground mr-1">关键词:</span>
                          {parsedKw.map((k, idx) => (
                            <Badge key={idx} variant="secondary" className="text-xs gap-1 font-mono">
                              <span>{k.keyword}</span>
                              <span className="text-[10px] text-muted-foreground">
                                ({k.match_type === 'full' ? '精准' : '半匹配'})
                              </span>
                            </Badge>
                          ))}
                        </div>

                        <div className="text-xs text-muted-foreground bg-muted/40 p-2.5 rounded-lg whitespace-pre-wrap">
                          {rule.content}
                        </div>
                      </div>

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
                暂无关键词回复规则，点击右上角「新建关键词规则」开始配置
              </Card>
            )}
          </div>
        </TabsContent>

        {/* AI 智能助手标签页 */}
        <TabsContent value="ai" className="mt-6 space-y-6">
          <div className="grid gap-6 lg:grid-cols-12 items-start">
            {/* 左侧：AI 配置参数 */}
            <div className="lg:col-span-7">
              <Card>
                <CardHeader className="pb-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <CardTitle className="text-base flex items-center gap-2">
                        <Bot className="h-5 w-5 text-purple-600" />
                        <span>AI 智能机器人配置</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        兼容 OpenAI / ChatGPT / DeepSeek / 通义千问 / Kimi / Ollama 等接口规范
                      </CardDescription>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-semibold">{aiEnabled ? 'AI 回复已开启' : '已关闭'}</span>
                      <Switch checked={aiEnabled} onCheckedChange={setAiEnabled} />
                    </div>
                  </div>
                </CardHeader>

                <CardContent className="space-y-4 text-xs">
                  <div className="space-y-1.5">
                    <label className="font-semibold text-foreground">服务商快速预设 (Provider)</label>
                    <Select value={aiProvider} onValueChange={handleProviderChange}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="openai">OpenAI (官方接口)</SelectItem>
                        <SelectItem value="deepseek">DeepSeek (深度求索)</SelectItem>
                        <SelectItem value="qwen">通义千问 (DashScope)</SelectItem>
                        <SelectItem value="moonshot">Moonshot / Kimi</SelectItem>
                        <SelectItem value="ollama">Ollama (本地私有化部署)</SelectItem>
                        <SelectItem value="custom">自定义兼容接口</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="space-y-1.5">
                    <label className="font-semibold text-foreground">接口 Base URL (API 端点) *</label>
                    <Input
                      value={aiBaseUrl}
                      onChange={(e) => setAiBaseUrl(e.target.value)}
                      placeholder="https://api.openai.com/v1"
                      className="font-mono text-xs"
                    />
                    <p className="text-[11px] text-muted-foreground">系统将向 <code>{aiBaseUrl}/chat/completions</code> 发送请求</p>
                  </div>

                  <div className="space-y-1.5">
                    <label className="font-semibold text-foreground">API Key (密钥) *</label>
                    <Input
                      type="password"
                      value={aiApiKey}
                      onChange={(e) => setAiApiKey(e.target.value)}
                      placeholder="sk-..."
                      className="font-mono text-xs"
                    />
                  </div>

                  <div className="grid grid-cols-2 gap-3">
                    <div className="space-y-1.5">
                      <label className="font-semibold text-foreground">模型名称 (Model) *</label>
                      <Input
                        value={aiModel}
                        onChange={(e) => setAiModel(e.target.value)}
                        placeholder="gpt-4o-mini 或 deepseek-chat"
                        className="font-mono text-xs"
                      />
                    </div>

                    <div className="space-y-1.5">
                      <label className="font-semibold text-foreground">触发时机模式 (Match Mode)</label>
                      <Select value={aiMatchMode} onValueChange={setAiMatchMode}>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="fallback">兜底回复 (未命中关键词时AI应答)</SelectItem>
                          <SelectItem value="all">全量回复 (所有用户消息均由AI应答)</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>

                  <div className="space-y-1.5">
                    <label className="font-semibold text-foreground">系统人设 Prompt (System Prompt)</label>
                    <textarea
                      value={aiSystemPrompt}
                      onChange={(e) => setAiSystemPrompt(e.target.value)}
                      rows={3}
                      className="w-full rounded-lg border border-input bg-background p-2.5 text-xs focus:outline-none focus:ring-1 focus:ring-ring"
                      placeholder="定义 AI 客服身份、语调及知识边界..."
                    />
                  </div>

                  <div className="pt-2 flex justify-end">
                    <Button
                      onClick={() =>
                        saveAiMutation.mutate({
                          enabled: aiEnabled,
                          provider: aiProvider,
                          base_url: aiBaseUrl,
                          api_key: aiApiKey,
                          model: aiModel,
                          system_prompt: aiSystemPrompt,
                          match_mode: aiMatchMode,
                          temperature: 0.7,
                          max_tokens: 800,
                          timeout_sec: aiTimeout,
                        })
                      }
                      disabled={saveAiMutation.isPending}
                      className="gap-1.5 bg-emerald-600 hover:bg-emerald-700"
                    >
                      <Sparkles className="h-4 w-4" />
                      <span>{saveAiMutation.isPending ? '保存中...' : '保存 AI 配置'}</span>
                    </Button>
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* 右侧：在线调试测试器 */}
            <div className="lg:col-span-5 space-y-4">
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-base flex items-center gap-2">
                    <Zap className="h-4 w-4 text-amber-500" />
                    <span>在线实时连通性测试</span>
                  </CardTitle>
                  <CardDescription className="text-xs">直接与已填写的 BaseURL 及 Model 发起模拟提问</CardDescription>
                </CardHeader>
                <CardContent className="space-y-3 text-xs">
                  <div className="space-y-1">
                    <label className="font-semibold text-foreground">测试提问内容:</label>
                    <textarea
                      value={testPrompt}
                      onChange={(e) => setTestPrompt(e.target.value)}
                      rows={2}
                      className="w-full rounded-lg border border-input bg-background p-2.5 text-xs"
                      placeholder="输入测试问题..."
                    />
                  </div>

                  <Button
                    onClick={handleTestAI}
                    disabled={testingAI}
                    variant="outline"
                    className="w-full gap-2 border-emerald-500/40 text-emerald-600 hover:bg-emerald-500/10"
                  >
                    <Send className="h-3.5 w-3.5" />
                    <span>{testingAI ? '正在请求大模型...' : '发送测试并验证回复'}</span>
                  </Button>

                  {testResponse && (
                    <div className="space-y-1 p-3 bg-muted/60 rounded-xl border">
                      <div className="font-semibold text-foreground flex items-center gap-1">
                        <CheckCircle2 className="h-3.5 w-3.5 text-emerald-600" />
                        <span>AI 回复结果:</span>
                      </div>
                      <div className="text-foreground whitespace-pre-wrap leading-relaxed">{testResponse}</div>
                    </div>
                  )}
                </CardContent>
              </Card>
            </div>
          </div>
        </TabsContent>
      </Tabs>

      {/* 固定规则新增/编辑 Modal */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-lg max-h-[85vh] overflow-y-auto">
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
