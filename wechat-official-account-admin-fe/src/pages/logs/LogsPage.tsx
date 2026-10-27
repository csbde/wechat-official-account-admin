import React, { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api, PageResult } from '@/lib/api'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
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
} from '@/components/ui/dialog'
import { Search, Activity, Eye, Shield } from 'lucide-react'
import { formatDate } from '@/lib/utils'

export const LogsPage: React.FC = () => {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState('')
  const [viewLog, setViewLog] = useState<any>(null)

  const { data: pageResult, isLoading } = useQuery<PageResult<any>>({
    queryKey: ['logs', page, keyword, status],
    queryFn: () =>
      api.get(`/api/logs?page=${page}&page_size=15&keyword=${encodeURIComponent(keyword)}&status=${status}`),
  })

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">请求与追踪审计日志</h2>
          <p className="text-sm text-muted-foreground">
            集成 OpenTelemetry 链路追踪，记录所有 API 请求的耗时、状态码、客户端 IP 与报文内容
          </p>
        </div>
      </div>

      {/* 过滤条 */}
      <div className="flex flex-wrap items-center gap-3 p-4 bg-card rounded-xl border">
        <div className="relative flex-1 min-w-[240px]">
          <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value)
              setPage(1)
            }}
            placeholder="搜索请求路径、Client IP 或 Trace ID..."
            className="pl-9 text-xs h-9"
          />
        </div>

        <select
          value={status}
          onChange={(e) => {
            setStatus(e.target.value)
            setPage(1)
          }}
          className="h-9 px-3 rounded-lg border border-input bg-background text-xs"
        >
          <option value="">全部状态码</option>
          <option value="200">200 OK</option>
          <option value="400">400 Bad Request</option>
          <option value="401">401 Unauthorized</option>
          <option value="403">403 Forbidden</option>
          <option value="500">500 Server Error</option>
        </select>
      </div>

      {/* 表格 */}
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>请求时间</TableHead>
            <TableHead>Method & 状态</TableHead>
            <TableHead>请求路径 (Path)</TableHead>
            <TableHead>耗时 (Latency)</TableHead>
            <TableHead>Trace ID / IP</TableHead>
            <TableHead className="text-right">详情</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableRow>
              <TableCell colSpan={6} className="h-32 text-center text-muted-foreground">
                加载日志中...
              </TableCell>
            </TableRow>
          ) : pageResult?.list && pageResult.list.length > 0 ? (
            pageResult.list.map((log: any) => {
              const isSuccess = log.status_code >= 200 && log.status_code < 300
              return (
                <TableRow key={log.id} className="text-xs">
                  <TableCell className="text-muted-foreground whitespace-nowrap">
                    {formatDate(log.created_at)}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1.5 font-mono">
                      <span className="font-bold text-[11px] uppercase">{log.method}</span>
                      <Badge variant={isSuccess ? 'success' : 'destructive'} className="text-[10px]">
                        {log.status_code}
                      </Badge>
                    </div>
                  </TableCell>
                  <TableCell className="font-mono truncate max-w-xs">{log.path}</TableCell>
                  <TableCell>
                    <span
                      className={`font-mono font-medium ${
                        log.latency_ms > 500 ? 'text-amber-500' : 'text-emerald-600 dark:text-emerald-400'
                      }`}
                    >
                      {log.latency_ms} ms
                    </span>
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-col text-[11px] font-mono text-muted-foreground">
                      <span>{log.client_ip}</span>
                      {log.trace_id && <span className="text-[10px] truncate max-w-[120px]">{log.trace_id}</span>}
                    </div>
                  </TableCell>
                  <TableCell className="text-right">
                    <Button variant="ghost" size="icon" onClick={() => setViewLog(log)} className="h-7 w-7">
                      <Eye className="h-3.5 w-3.5" />
                    </Button>
                  </TableCell>
                </TableRow>
              )
            })
          ) : (
            <TableRow>
              <TableCell colSpan={6} className="h-32 text-center text-muted-foreground">
                暂无审计日志
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>

      {/* 分页 */}
      {pageResult?.pagination && (
        <div className="flex items-center justify-between text-xs text-muted-foreground pt-2">
          <span>共 {pageResult.pagination.total} 条日志</span>
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
            <span>第 {page} 页</span>
            <Button
              variant="outline"
              size="sm"
              disabled={page >= Math.ceil(pageResult.pagination.total / pageResult.pagination.page_size)}
              onClick={() => setPage(page + 1)}
              className="h-8 text-xs"
            >
              下一页
            </Button>
          </div>
        </div>
      )}

      {/* 日志详情弹窗 */}
      <Dialog open={!!viewLog} onOpenChange={() => setViewLog(null)}>
        <DialogContent className="sm:max-w-xl">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 font-mono text-sm">
              <Badge variant="outline">{viewLog?.method}</Badge>
              <span>{viewLog?.path}</span>
            </DialogTitle>
          </DialogHeader>

          {viewLog && (
            <div className="space-y-3 py-2 text-xs">
              <div className="grid grid-cols-2 gap-2 p-3 bg-muted/60 rounded-lg font-mono">
                <div>状态码: {viewLog.status_code}</div>
                <div>耗时: {viewLog.latency_ms} ms</div>
                <div>客户端IP: {viewLog.client_ip}</div>
                <div>时间: {formatDate(viewLog.created_at)}</div>
                <div className="col-span-2">Trace ID: {viewLog.trace_id || '未关联'}</div>
              </div>

              {viewLog.request_body && (
                <div className="space-y-1">
                  <div className="font-semibold text-muted-foreground">Request Body:</div>
                  <pre className="p-3 bg-muted rounded-lg font-mono text-[11px] overflow-x-auto max-h-36">
                    {viewLog.request_body}
                  </pre>
                </div>
              )}

              {viewLog.response_body && (
                <div className="space-y-1">
                  <div className="font-semibold text-muted-foreground">Response Body:</div>
                  <pre className="p-3 bg-muted rounded-lg font-mono text-[11px] overflow-x-auto max-h-48">
                    {viewLog.response_body}
                  </pre>
                </div>
              )}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
