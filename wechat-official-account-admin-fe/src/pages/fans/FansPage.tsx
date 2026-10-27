import React, { useState, useMemo } from 'react'
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
  UserCheck,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatDate } from '@/lib/utils'

export const FansPage: React.FC = () => {
  const queryClient = useQueryClient()
  const { currentApp } = useAppStore()

  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [selectedTag, setSelectedTag] = useState('')
  const [selectedOpenIDs, setSelectedOpenIDs] = useState<string[]>([])

  // 备注弹窗
  const [remarkModalOpen, setRemarkModalOpen] = useState(false)
  const [editingFan, setEditingFan] = useState<any>(null)
  const [remarkText, setRemarkText] = useState('')

  // 标签管理弹窗
  const [tagModalOpen, setTagModalOpen] = useState(false)
  const [newTagName, setNewTagName] = useState('')

  // 获取粉丝列表
  const { data: fansResult, isLoading } = useQuery<PageResult<any>>({
    queryKey: ['fans', currentApp?.appid, page, keyword, selectedTag],
    queryFn: () =>
      api.get(`/api/fans?page=${page}&page_size=10&keyword=${encodeURIComponent(keyword)}&tag_id=${selectedTag}`),
    enabled: !!currentApp,
  })

  // 获取微信标签列表
  const { data: tags } = useQuery<any[]>({
    queryKey: ['fans-tags', currentApp?.appid],
    queryFn: () => api.get('/api/fans/tags'),
    enabled: !!currentApp,
  })

  // 触发后台同步
  const syncMutation = useMutation({
    mutationFn: () => api.post('/api/fans/sync'),
    onSuccess: (data: any) => {
      toast.success(data?.message || '微信粉丝同步任务已派发')
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

  // 创建新标签
  const createTagMutation = useMutation({
    mutationFn: (name: string) => api.post('/api/fans/tags', { name }),
    onSuccess: () => {
      toast.success('标签创建成功')
      queryClient.invalidateQueries({ queryKey: ['fans-tags'] })
      setNewTagName('')
    },
  })

  // 删除标签
  const deleteTagMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/fans/tags/${id}`),
    onSuccess: () => {
      toast.success('标签已删除')
      queryClient.invalidateQueries({ queryKey: ['fans-tags'] })
    },
  })

  // 批量打/移除标签
  const batchTagMutation = useMutation({
    mutationFn: (values: { openids: string[]; tag_id: number; action: string }) =>
      api.post('/api/fans/tags/batch', values),
    onSuccess: () => {
      toast.success('批量标签更新成功')
      queryClient.invalidateQueries({ queryKey: ['fans'] })
      queryClient.invalidateQueries({ queryKey: ['fans-tags'] })
      setSelectedOpenIDs([])
    },
  })

  // TanStack Table 列配置
  const columns = useMemo<ColumnDef<any>[]>(
    () => [
      {
        id: 'select',
        header: ({ table }) => (
          <input
            type="checkbox"
            className="rounded"
            checked={
              table.getPrePaginationRowModel().rows.length > 0 &&
              selectedOpenIDs.length === table.getPrePaginationRowModel().rows.length
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
            className="rounded"
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
        cell: ({ row }) => (
          <div className="flex items-center gap-3">
            <div className="h-9 w-9 rounded-full bg-muted overflow-hidden flex items-center justify-center font-bold text-xs text-muted-foreground shrink-0 border">
              {row.original.headimgurl ? (
                <img src={row.original.headimgurl} alt="" className="h-full w-full object-cover" />
              ) : (
                row.original.nickname?.slice(0, 1) || 'U'
              )}
            </div>
            <div>
              <div className="font-semibold text-sm leading-tight flex items-center gap-1.5">
                <span>{row.original.nickname || '微信用户'}</span>
                {row.original.sex === 1 ? (
                  <span className="text-[10px] text-blue-500 font-bold">♂</span>
                ) : row.original.sex === 2 ? (
                  <span className="text-[10px] text-pink-500 font-bold">♀</span>
                ) : null}
              </div>
              <div className="text-[11px] text-muted-foreground font-mono">{row.original.openid}</div>
            </div>
          </div>
        ),
      },
      {
        accessorKey: 'remark',
        header: '备注名',
        cell: ({ row }) => (
          <div className="flex items-center gap-1.5 group">
            <span className="text-sm">{row.original.remark || '-'}</span>
            <button
              onClick={() => {
                setEditingFan(row.original)
                setRemarkText(row.original.remark || '')
                setRemarkModalOpen(true)
              }}
              className="opacity-0 group-hover:opacity-100 transition-opacity text-muted-foreground hover:text-primary"
            >
              <Edit2 className="h-3.5 w-3.5" />
            </button>
          </div>
        ),
      },
      {
        accessorKey: 'tag_ids',
        header: '用户标签',
        cell: ({ row }) => {
          let list: number[] = []
          try {
            list = JSON.parse(row.original.tag_ids || '[]')
          } catch {}

          return (
            <div className="flex flex-wrap gap-1">
              {list.map((tagId) => {
                const found = tags?.find((t) => t.tag_id === tagId)
                return (
                  <Badge key={tagId} variant="secondary" className="text-[10px]">
                    {found ? found.name : `Tag ${tagId}`}
                  </Badge>
                )
              })}
              {list.length === 0 && <span className="text-xs text-muted-foreground">-</span>}
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

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">粉丝与标签管理</h2>
          <p className="text-sm text-muted-foreground">
            粉丝全量表格管理、多维搜索、设置备注与批量标签操作
          </p>
        </div>

        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={() => setTagModalOpen(true)} className="gap-1.5 text-xs">
            <Tag className="h-4 w-4" />
            <span>标签管理</span>
          </Button>
          <Button
            onClick={() => syncMutation.mutate()}
            disabled={syncMutation.isPending}
            className="gap-1.5 text-xs bg-emerald-600 hover:bg-emerald-700 text-white font-semibold"
          >
            <RefreshCw className={`h-4 w-4 ${syncMutation.isPending ? 'animate-spin' : ''}`} />
            <span>异步同步粉丝</span>
          </Button>
        </div>
      </div>

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
                暂无粉丝数据，请先配置公众号并点击右上角「异步同步粉丝」
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>

      {/* 分页控制器 */}
      {fansResult?.pagination && (
        <div className="flex items-center justify-between text-xs text-muted-foreground pt-2">
          <span>共 {fansResult.pagination.total} 位粉丝</span>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={page <= 1}
              onClick={() => setPage(page - 1)}
              className="h-8 text-xs"
            >
              上一页
            </Button>
            <span>
              第 {page} 页 / 共 {Math.ceil(fansResult.pagination.total / fansResult.pagination.page_size) || 1} 页
            </span>
            <Button
              variant="outline"
              size="sm"
              disabled={page >= Math.ceil(fansResult.pagination.total / fansResult.pagination.page_size)}
              onClick={() => setPage(page + 1)}
              className="h-8 text-xs"
            >
              下一页
            </Button>
          </div>
        </div>
      )}

      {/* 备注修改弹窗 */}
      <Dialog open={remarkModalOpen} onOpenChange={setRemarkModalOpen}>
        <DialogContent className="sm:max-w-md">
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

      {/* 标签管理弹窗 (支持新增与删除) */}
      <Dialog open={tagModalOpen} onOpenChange={setTagModalOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>微信粉丝标签管理</DialogTitle>
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
                        if (confirm(`确认删除标签 "${t.name}"？`)) {
                          deleteTagMutation.mutate(t.id)
                        }
                      }}
                      className="text-muted-foreground hover:text-destructive"
                      title="删除标签"
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
