import React, { useState, useMemo, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  useReactTable,
  getCoreRowModel,
  flexRender,
  ColumnDef,
} from '@tanstack/react-table'
import { api, PageResult } from '@/lib/api'
import { useAppStore } from '@/stores/app'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from '@/components/ui/table'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  Users,
  Tag,
  RefreshCw,
  Search,
  Edit2,
  Trash2,
  CheckCircle2,
  AlertCircle,
  Plus,
  Check,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatDate } from '@/lib/utils'

export const FansPage: React.FC = () => {
  const queryClient = useQueryClient()
  const { currentApp } = useAppStore()

  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [keyword, setKeyword] = useState('')
  const [selectedTag, setSelectedTag] = useState('')
  const [selectedOpenIDs, setSelectedOpenIDs] = useState<string[]>([])
  const [jumpPage, setJumpPage] = useState('')

  // 备注弹窗
  const [remarkModalOpen, setRemarkModalOpen] = useState(false)
  const [editingFan, setEditingFan] = useState<any>(null)
  const [remarkText, setRemarkText] = useState('')

  // 单用户标签管理弹窗 (重点解决行内无法打标签的问题！)
  const [singleTagModalOpen, setSingleTagModalOpen] = useState(false)
  const [taggingFan, setTaggingFan] = useState<any>(null)

  // 微信标签体系管理弹窗
  const [tagModalOpen, setTagModalOpen] = useState(false)
  const [newTagName, setNewTagName] = useState('')

  // 轮询同步进度状态
  const [isSyncing, setIsSyncing] = useState(false)

  // 获取粉丝列表
  const { data: fansResult, isLoading, refetch: refetchFans } = useQuery<PageResult<any>>({
    queryKey: ['fans', currentApp?.appid, page, pageSize, keyword, selectedTag],
    queryFn: () =>
      api.get(
        `/api/fans?page=${page}&page_size=${pageSize}&keyword=${encodeURIComponent(
          keyword
        )}&tag_id=${selectedTag}`
      ),
    enabled: !!currentApp,
  })

  // 获取微信标签列表
  const { data: tags } = useQuery<any[]>({
    queryKey: ['fans-tags', currentApp?.appid],
    queryFn: () => api.get('/api/fans/tags'),
    enabled: !!currentApp,
  })

  // 轮询同步任务进度
  const { data: syncStatus, refetch: refetchSyncStatus } = useQuery<any>({
    queryKey: ['fans-sync-status', currentApp?.appid],
    queryFn: () => api.get('/api/fans/sync-status'),
    enabled: !!currentApp,
    refetchInterval: isSyncing ? 1500 : false,
  })

  useEffect(() => {
    if (syncStatus?.status === 'running') {
      setIsSyncing(true)
    } else if (syncStatus?.status === 'completed' && isSyncing) {
      setIsSyncing(false)
      toast.success(syncStatus.message || '微信粉丝资料与头像全量同步完成！')
      queryClient.invalidateQueries({ queryKey: ['fans'] })
      queryClient.invalidateQueries({ queryKey: ['fans-tags'] })
    } else if (syncStatus?.status === 'failed' && isSyncing) {
      setIsSyncing(false)
      toast.error('粉丝同步失败: ' + (syncStatus.error || syncStatus.message))
    }
  }, [syncStatus, isSyncing, queryClient])

  // 触发后台同步
  const syncMutation = useMutation({
    mutationFn: () => api.post('/api/fans/sync'),
    onSuccess: (data: any) => {
      setIsSyncing(true)
      toast.info(data?.message || '同步任务已启动，正在后台拉取')
      refetchSyncStatus()
    },
    onError: (err: any) => {
      toast.error('触发同步失败: ' + (err.message || '网络错误'))
    },
  })

  // 更新备注
  const updateRemarkMutation = useMutation({
    mutationFn: (values: { openid: string; remark: string }) => api.post('/api/fans/remark', values),
    onSuccess: () => {
      toast.success('粉丝备注更新成功')
      queryClient.invalidateQueries({ queryKey: ['fans'] })
      setRemarkModalOpen(false)
    },
  })

  // 创建新标签 (优先向微信官方同步创建，失败时退化为本地标签并提示)
  const createTagMutation = useMutation({
    mutationFn: (name: string) => api.post('/api/fans/tags', { name }),
    onSuccess: (d: any) => {
      if (d?.synced) {
        toast.success(`微信标签创建成功并同步至官方 (tag_id=${d?.tag?.tag_id})`)
      } else {
        toast.warning(d?.message || '标签仅在本地创建，未同步至微信官方', { duration: 6000 })
      }
      queryClient.invalidateQueries({ queryKey: ['fans-tags'] })
      setNewTagName('')
    },
  })

  // 删除标签 (向微信官方同步删除)
  const deleteTagMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/fans/tags/${id}`),
    onSuccess: () => {
      toast.success('微信标签已同步删除')
      queryClient.invalidateQueries({ queryKey: ['fans-tags'] })
      queryClient.invalidateQueries({ queryKey: ['fans'] })
    },
  })

  // 批量打/移除标签 (向微信官方同步)
  const batchTagMutation = useMutation({
    mutationFn: (values: { openids: string[]; tag_id: number; action: string }) =>
      api.post('/api/fans/tags/batch', values),
    onSuccess: () => {
      toast.success('微信标签同步更新成功！')
      queryClient.invalidateQueries({ queryKey: ['fans'] })
      queryClient.invalidateQueries({ queryKey: ['fans-tags'] })
      setSelectedOpenIDs([])
    },
  })

  // 计算总页数
  const total = fansResult?.pagination?.total || 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  // TanStack Table 列配置
  const columns = useMemo<ColumnDef<any>[]>(
    () => [
      {
        id: 'select',
        header: () => (
          <input
            type="checkbox"
            className="rounded cursor-pointer"
            checked={
              (fansResult?.list || []).length > 0 &&
              selectedOpenIDs.length === (fansResult?.list || []).length
            }
            onChange={(e) => {
              if (e.target.checked) {
                const all = fansResult?.list.map((r: any) => r.openid) || []
                setSelectedOpenIDs(all)
              } else {
                setSelectedOpenIDs([])
              }
            }}
          />
        ),
        cell: ({ row }) => (
          <input
            type="checkbox"
            className="rounded cursor-pointer"
            checked={selectedOpenIDs.includes(row.original.openid)}
            onChange={(e) => {
              if (e.target.checked) {
                setSelectedOpenIDs([...selectedOpenIDs, row.original.openid])
              } else {
                setSelectedOpenIDs(selectedOpenIDs.filter((id) => id !== row.original.openid))
              }
            }}
          />
        ),
      },
      {
        accessorKey: 'nickname',
        header: '粉丝用户',
        cell: ({ row }) => {
          const fan = row.original
          const displayNick = fan.nickname || fan.remark || `微信粉丝_${fan.openid.slice(-6)}`
          const initialChar = displayNick.slice(0, 1).toUpperCase()

          return (
            <div className="flex items-center gap-3">
              {/* 头像展示 (带 no-referrer 规避微信图片防盗链，若为空提供美观首字母渐变底) */}
              <div className="h-10 w-10 rounded-full overflow-hidden flex items-center justify-center font-bold text-xs text-white shrink-0 border shadow-sm bg-gradient-to-tr from-emerald-600 to-teal-500">
                {fan.headimgurl ? (
                  <img
                    src={fan.headimgurl}
                    referrerPolicy="no-referrer"
                    alt={displayNick}
                    className="h-full w-full object-cover"
                    onError={(e) => {
                      ;(e.target as HTMLElement).style.display = 'none'
                    }}
                  />
                ) : (
                  <span>{initialChar}</span>
                )}
              </div>

              <div>
                <div className="font-semibold text-sm leading-tight flex items-center gap-1.5">
                  <span className="text-foreground">{displayNick}</span>
                  {fan.sex === 1 ? (
                    <span className="text-[11px] text-blue-500 font-bold" title="男">♂</span>
                  ) : fan.sex === 2 ? (
                    <span className="text-[11px] text-pink-500 font-bold" title="女">♀</span>
                  ) : null}
                </div>
                <div className="text-[11px] text-muted-foreground font-mono mt-0.5">{fan.openid}</div>
              </div>
            </div>
          )
        },
      },
      {
        accessorKey: 'remark',
        header: '专属备注',
        cell: ({ row }) => (
          <div className="flex items-center gap-1.5 group">
            <span className="text-sm">{row.original.remark || '-'}</span>
            <button
              onClick={() => {
                setEditingFan(row.original)
                setRemarkText(row.original.remark || '')
                setRemarkModalOpen(true)
              }}
              className="opacity-0 group-hover:opacity-100 transition-opacity text-muted-foreground hover:text-primary p-0.5"
              title="设置备注"
            >
              <Edit2 className="h-3.5 w-3.5" />
            </button>
          </div>
        ),
      },
      {
        accessorKey: 'tag_ids',
        header: '标签与分组',
        cell: ({ row }) => {
          let list: number[] = []
          try {
            list = JSON.parse(row.original.tag_ids || '[]')
          } catch {}

          return (
            <div className="flex flex-wrap items-center gap-1">
              {list.map((tagId) => {
                const found = tags?.find((t) => t.tag_id === tagId)
                return (
                  <Badge key={tagId} variant="secondary" className="text-[10px] py-0 px-1.5">
                    {found ? found.name : `Tag ${tagId}`}
                  </Badge>
                )
              })}
              {list.length === 0 && <span className="text-xs text-muted-foreground mr-1">无标签</span>}

              {/* 行内单用户打标签快捷按钮 */}
              <button
                type="button"
                onClick={() => {
                  setTaggingFan(row.original)
                  setSingleTagModalOpen(true)
                }}
                className="text-primary hover:underline text-[11px] inline-flex items-center gap-0.5 ml-1"
                title="为此粉丝设置/修改标签"
              >
                <Plus className="h-3 w-3" />
                <span>贴标签</span>
              </button>
            </div>
          )
        },
      },
      {
        accessorKey: 'subscribe_time',
        header: '关注时间',
        cell: ({ row }) => (
          <span className="text-xs text-muted-foreground">
            {formatDate(row.original.subscribe_time ? new Date(row.original.subscribe_time * 1000) : null)}
          </span>
        ),
      },
      {
        accessorKey: 'subscribe',
        header: '状态',
        cell: ({ row }) => (
          <Badge variant={row.original.subscribe === 1 ? 'success' : 'secondary'} className="text-[10px]">
            {row.original.subscribe === 1 ? '已关注' : '已取关'}
          </Badge>
        ),
      },
    ],
    [selectedOpenIDs, fansResult, tags]
  )

  const table = useReactTable({
    data: fansResult?.list || [],
    columns,
    getCoreRowModel: getCoreRowModel(),
  })

  // 为单个粉丝切换标签
  const handleToggleFanTag = (fan: any, tagId: number, hasTag: boolean) => {
    batchTagMutation.mutate({
      openids: [fan.openid],
      tag_id: tagId,
      action: hasTag ? 'remove' : 'add',
    })
  }

  // 页码直接跳转
  const handleJumpPage = (e: React.FormEvent) => {
    e.preventDefault()
    const target = parseInt(jumpPage)
    if (target >= 1 && target <= totalPages) {
      setPage(target)
      setJumpPage('')
    } else {
      toast.error(`请输入 1 到 ${totalPages} 之间的有效页码`)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">粉丝与标签管理</h2>
          <p className="text-sm text-muted-foreground">
            全量微信粉丝同步、头像与名称清晰展示、全功能翻页与微信官方双向标签管理
          </p>
        </div>

        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={() => setTagModalOpen(true)} className="gap-1.5 text-xs">
            <Tag className="h-4 w-4" />
            <span>标签管理</span>
          </Button>
          <Button
            onClick={() => syncMutation.mutate()}
            disabled={isSyncing || syncMutation.isPending}
            className="gap-1.5 text-xs bg-emerald-600 hover:bg-emerald-700 text-white font-semibold"
          >
            <RefreshCw className={`h-4 w-4 ${isSyncing || syncMutation.isPending ? 'animate-spin' : ''}`} />
            <span>{isSyncing ? '正在拉取微信粉丝...' : '异步同步粉丝'}</span>
          </Button>
        </div>
      </div>

      {/* 实时同步执行情况与进度监控卡片 */}
      {syncStatus && syncStatus.status !== 'idle' && (
        <Card
          className={`border-2 transition-all ${
            syncStatus.status === 'running'
              ? 'border-emerald-500 bg-emerald-500/5'
              : syncStatus.status === 'failed'
              ? 'border-destructive bg-destructive/5'
              : 'border-border'
          }`}
        >
          <CardContent className="p-4 space-y-2.5">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                {syncStatus.status === 'running' ? (
                  <RefreshCw className="h-4 w-4 text-emerald-600 animate-spin" />
                ) : syncStatus.status === 'completed' ? (
                  <CheckCircle2 className="h-4 w-4 text-emerald-600" />
                ) : (
                  <AlertCircle className="h-4 w-4 text-destructive" />
                )}
                <span className="font-semibold text-sm text-foreground">
                  {syncStatus.status === 'running'
                    ? '微信粉丝全量同步正在执行中...'
                    : syncStatus.status === 'completed'
                    ? '粉丝数据全量同步已完成'
                    : '粉丝同步遇到错误'}
                </span>
                <Badge
                  variant={
                    syncStatus.status === 'running'
                      ? 'default'
                      : syncStatus.status === 'completed'
                      ? 'success'
                      : 'destructive'
                  }
                  className="text-[10px]"
                >
                  {syncStatus.status.toUpperCase()}
                </Badge>
              </div>

              {syncStatus.total_fans > 0 && (
                <span className="font-mono text-xs font-semibold text-primary">
                  {syncStatus.synced_fans} / {syncStatus.total_fans} 位 ({syncStatus.percent}%)
                </span>
              )}
            </div>

            <div className="w-full bg-muted rounded-full h-2 overflow-hidden">
              <div
                className={`h-full transition-all duration-300 rounded-full ${
                  syncStatus.status === 'failed' ? 'bg-destructive' : 'bg-emerald-600'
                }`}
                style={{ width: `${Math.max(5, syncStatus.percent || 0)}%` }}
              />
            </div>

            <div className="flex items-center justify-between text-xs text-muted-foreground">
              <span>{syncStatus.message}</span>
              {syncStatus.start_time && <span>开始于: {formatDate(syncStatus.start_time)}</span>}
            </div>
          </CardContent>
        </Card>
      )}

      {/* 搜索与筛选栏 */}
      <div className="flex flex-wrap items-center justify-between gap-3 p-4 bg-card rounded-xl border">
        <div className="flex flex-wrap items-center gap-2.5 flex-1 max-w-xl">
          <div className="relative flex-1 min-w-[200px]">
            <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
              placeholder="搜索粉丝昵称、OpenID 或 备注名..."
              className="pl-9 text-xs h-9"
            />
          </div>

          <select
            value={selectedTag}
            onChange={(e) => {
              setSelectedTag(e.target.value)
              setPage(1)
            }}
            className="h-9 px-3 rounded-lg border border-input bg-background text-xs"
          >
            <option value="">全部标签</option>
            {tags?.map((t) => (
              <option key={t.tag_id} value={t.tag_id}>
                {t.name} ({t.count}人)
              </option>
            ))}
          </select>
        </div>

        {/* 批量操作工具条 */}
        {selectedOpenIDs.length > 0 && (
          <div className="flex items-center gap-2 text-xs bg-primary/10 text-primary px-3 py-1.5 rounded-lg font-medium">
            <span>已选 {selectedOpenIDs.length} 人</span>
            <select
              onChange={(e) => {
                const tagId = parseInt(e.target.value)
                if (tagId) {
                  batchTagMutation.mutate({
                    openids: selectedOpenIDs,
                    tag_id: tagId,
                    action: 'add',
                  })
                }
              }}
              className="h-7 text-xs bg-background border rounded px-1 text-foreground"
            >
              <option value="">批量打标签...</option>
              {tags?.map((t) => (
                <option key={t.tag_id} value={t.tag_id}>
                  {t.name}
                </option>
              ))}
            </select>
            <select
              onChange={(e) => {
                const tagId = parseInt(e.target.value)
                if (tagId) {
                  batchTagMutation.mutate({
                    openids: selectedOpenIDs,
                    tag_id: tagId,
                    action: 'remove',
                  })
                }
              }}
              className="h-7 text-xs bg-background border rounded px-1 text-foreground"
            >
              <option value="">批量移除标签...</option>
              {tags?.map((t) => (
                <option key={t.tag_id} value={t.tag_id}>
                  {t.name}
                </option>
              ))}
            </select>
          </div>
        )}
      </div>

      {/* TanStack Table 渲染 */}
      <Table>
        <TableHeader>
          {table.getHeaderGroups().map((headerGroup) => (
            <TableRow key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <TableHead key={header.id}>
                  {flexRender(header.column.columnDef.header, header.getContext())}
                </TableHead>
              ))}
            </TableRow>
          ))}
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableRow>
              <TableCell colSpan={columns.length} className="h-32 text-center text-muted-foreground">
                正在加载粉丝数据...
              </TableCell>
            </TableRow>
          ) : table.getRowModel().rows?.length ? (
            table.getRowModel().rows.map((row) => (
              <TableRow key={row.id}>
                {row.getVisibleCells().map((cell) => (
                  <TableCell key={cell.id}>
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </TableCell>
                ))}
              </TableRow>
            ))
          ) : (
            <TableRow>
              <TableCell colSpan={columns.length} className="h-32 text-center text-muted-foreground">
                暂无粉丝数据，请点击右上角「异步同步粉丝」全量拉取微信数据
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>

      {/* 强化版全功能翻页控制器 (重点解决无法翻页问题！) */}
      <div className="flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-muted-foreground pt-3 border-t">
        <div className="flex items-center gap-3">
          <span>共 {total} 位粉丝</span>
          <div className="flex items-center gap-1.5">
            <span>每页展示:</span>
            <select
              value={pageSize}
              onChange={(e) => {
                setPageSize(parseInt(e.target.value))
                setPage(1)
              }}
              className="h-7 px-2 rounded border border-input bg-background text-xs text-foreground"
            >
              <option value="10">10 条/页</option>
              <option value="20">20 条/页</option>
              <option value="50">50 条/页</option>
              <option value="100">100 条/页</option>
            </select>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1">
            <Button
              variant="outline"
              size="sm"
              disabled={page <= 1}
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              className="h-8 px-2.5 text-xs gap-1"
            >
              <ChevronLeft className="h-3.5 w-3.5" />
              <span>上一页</span>
            </Button>

            <span className="px-2 font-medium text-foreground">
              第 {page} / {totalPages} 页
            </span>

            <Button
              variant="outline"
              size="sm"
              disabled={page >= totalPages}
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              className="h-8 px-2.5 text-xs gap-1"
            >
              <span>下一页</span>
              <ChevronRight className="h-3.5 w-3.5" />
            </Button>
          </div>

          {totalPages > 1 && (
            <form onSubmit={handleJumpPage} className="flex items-center gap-1.5">
              <span>跳至</span>
              <Input
                type="number"
                min={1}
                max={totalPages}
                value={jumpPage}
                onChange={(e) => setJumpPage(e.target.value)}
                className="h-7 w-14 text-center text-xs p-1"
                placeholder={page.toString()}
              />
              <span>页</span>
              <Button type="submit" variant="secondary" size="sm" className="h-7 text-xs px-2">
                确定
              </Button>
            </form>
          )}
        </div>
      </div>

      {/* 单用户标签管理弹窗 (重点解决无法打标签问题！) */}
      <Dialog open={singleTagModalOpen} onOpenChange={setSingleTagModalOpen}>
        <DialogContent className="sm:max-w-md max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Tag className="h-4 w-4 text-emerald-600" />
              <span>为粉丝设置标签</span>
            </DialogTitle>
          </DialogHeader>

          {taggingFan && (
            <div className="space-y-4 py-2 text-xs">
              <div className="p-3 bg-muted/60 rounded-xl flex items-center gap-3">
                <div className="h-9 w-9 rounded-full bg-emerald-600 text-white flex items-center justify-center font-bold">
                  {taggingFan.nickname ? taggingFan.nickname.slice(0, 1) : 'U'}
                </div>
                <div>
                  <div className="font-semibold text-sm text-foreground">{taggingFan.nickname || '微信用户'}</div>
                  <div className="text-muted-foreground font-mono text-[11px]">{taggingFan.openid}</div>
                </div>
              </div>

              <div className="space-y-2">
                <div className="font-semibold text-foreground">选择微信标签 (点击即可直接向微信同步打/去标签):</div>
                <div className="grid grid-cols-2 gap-2 max-h-56 overflow-y-auto p-1">
                  {tags?.map((t) => {
                    let userTags: number[] = []
                    try {
                      userTags = JSON.parse(taggingFan.tag_ids || '[]')
                    } catch {}
                    const isTagged = userTags.includes(t.tag_id)

                    return (
                      <button
                        key={t.id}
                        type="button"
                        onClick={() => {
                          handleToggleFanTag(taggingFan, t.tag_id, isTagged)
                          // 乐观更新本地
                          let nextTags: number[] = []
                          if (isTagged) {
                            nextTags = userTags.filter((x) => x !== t.tag_id)
                          } else {
                            nextTags = [...userTags, t.tag_id]
                          }
                          setTaggingFan({ ...taggingFan, tag_ids: JSON.stringify(nextTags) })
                        }}
                        className={`flex items-center justify-between p-2.5 rounded-lg border text-xs text-left transition-all ${
                          isTagged
                            ? 'border-emerald-600 bg-emerald-500/10 text-emerald-800 dark:text-emerald-300 font-semibold shadow-sm'
                            : 'hover:bg-muted text-foreground'
                        }`}
                      >
                        <span className="truncate">{t.name}</span>
                        {isTagged && <Check className="h-3.5 w-3.5 text-emerald-600 shrink-0" />}
                      </button>
                    )
                  })}
                </div>
              </div>

              <DialogFooter className="pt-2">
                <Button variant="default" size="sm" onClick={() => setSingleTagModalOpen(false)}>
                  完成设置
                </Button>
              </DialogFooter>
            </div>
          )}
        </DialogContent>
      </Dialog>

      {/* 备注修改弹窗 */}
      <Dialog open={remarkModalOpen} onOpenChange={setRemarkModalOpen}>
        <DialogContent className="sm:max-w-md max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>设置用户备注名</DialogTitle>
          </DialogHeader>
          <div className="space-y-3 py-2 text-sm">
            <div className="text-xs text-muted-foreground">
              为用户 <span className="font-semibold text-foreground">{editingFan?.nickname}</span> 设置专属备注:
            </div>
            <Input
              value={remarkText}
              onChange={(e) => setRemarkText(e.target.value)}
              placeholder="输入备注名 (例如: VIP学员-张三)"
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRemarkModalOpen(false)}>取消</Button>
            <Button
              onClick={() => {
                if (editingFan) {
                  updateRemarkMutation.mutate({ openid: editingFan.openid, remark: remarkText })
                }
              }}
              disabled={updateRemarkMutation.isPending}
            >
              确定保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 微信标签管理弹窗 */}
      <Dialog open={tagModalOpen} onOpenChange={setTagModalOpen}>
        <DialogContent className="sm:max-w-md max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>微信官方粉丝标签管理</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2 text-sm">
            <div className="flex gap-2">
              <Input
                value={newTagName}
                onChange={(e) => setNewTagName(e.target.value)}
                placeholder="新标签名称 (如: 意向客户)"
              />
              <Button onClick={() => newTagName.trim() && createTagMutation.mutate(newTagName.trim())}>
                创建
              </Button>
            </div>

            <div className="divide-y max-h-60 overflow-y-auto rounded-lg border">
              {tags?.map((t) => (
                <div key={t.id} className="p-2.5 flex items-center justify-between text-xs">
                  <div className="flex items-center gap-2">
                    <span className="font-semibold">{t.name}</span>
                    <Badge variant="outline" className="text-[10px]">{t.count} 人</Badge>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-[10px] text-muted-foreground font-mono">ID: {t.tag_id}</span>
                    <button
                      onClick={() => {
                        if (confirm(`确认向微信官方同步删除标签 "${t.name}"？`)) {
                          deleteTagMutation.mutate(t.id)
                        }
                      }}
                      className="text-muted-foreground hover:text-destructive"
                      title="同步删除标签"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
