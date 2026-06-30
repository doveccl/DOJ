import { FullscreenOutlined, PlayCircleOutlined, ReloadOutlined } from '@ant-design/icons'
import { Alert, Button, Flex, Space, Spin, Tag, Typography } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useRef } from 'react'
import screenfull from 'screenfull'

import { api, apiData } from '../client'
import type { PlagiarismJob } from '../client'
import { useLocale } from '../locale'

type Props = {
  scope: 'assignment' | 'contest'
  id: number
}

export function PlagiarismPanel({ scope, id }: Props) {
  const { text } = useLocale()
  const queryClient = useQueryClient()
  const frameRef = useRef<HTMLIFrameElement>(null)
  const jobs = useQuery({
    queryKey: ['plagiarism', scope, id],
    queryFn: () => apiData(api.GET('/api/admin/plagiarism', { params: { query: { scope, id } } })),
    refetchInterval: (query) => {
      const status = query.state.data?.items[0]?.status
      return status === 'queued' || status === 'running' ? 2000 : false
    }
  })
  const create = useMutation({
    mutationFn: () =>
      scope === 'assignment'
        ? apiData(api.POST('/api/admin/plagiarism/assignments/{id}', { params: { path: { id } } }))
        : apiData(api.POST('/api/admin/plagiarism/contests/{id}', { params: { path: { id } } })),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['plagiarism', scope, id] })
  })
  const job = jobs.data?.items[0]
  const busy = create.isPending || job?.status === 'queued' || job?.status === 'running'

  function fullscreen() {
    const frame = frameRef.current
    if (frame && screenfull.isEnabled) {
      void screenfull.request(frame)
    }
  }

  return (
    <Flex vertical gap={12}>
      <Flex justify="space-between" align="center" gap={12} wrap>
        <Space>
          <Button type="primary" icon={<PlayCircleOutlined />} loading={create.isPending} onClick={() => create.mutate()}>
            {text.plagiarism.run}
          </Button>
          <Button icon={<ReloadOutlined />} loading={jobs.isFetching} onClick={() => void jobs.refetch()}>
            {text.common.refresh}
          </Button>
        </Space>
        {job ? <PlagiarismStatus job={job} /> : <Typography.Text type="secondary">{text.plagiarism.empty}</Typography.Text>}
      </Flex>
      {create.isError ? <Alert type="error" showIcon message={create.error instanceof Error ? create.error.message : text.common.loadingFailed} /> : null}
      {job?.status === 'failed' ? <Alert type="error" showIcon message={job.message || text.plagiarism.failed} /> : null}
      {busy ? (
        <Alert type="info" showIcon message={<Space><Spin size="small" />{text.plagiarism.running}</Space>} />
      ) : job?.viewerUrl ? (
        <Flex vertical gap={8}>
          <Flex justify="flex-end">
            <Button icon={<FullscreenOutlined />} onClick={fullscreen}>
              {text.plagiarism.fullscreen}
            </Button>
          </Flex>
          <iframe ref={frameRef} title={text.plagiarism.title} src={job.viewerUrl} style={{ width: '100%', height: 720, border: 0 }} />
        </Flex>
      ) : null}
    </Flex>
  )
}

function PlagiarismStatus({ job }: { job: PlagiarismJob }) {
  const { text } = useLocale()
  const color = job.status === 'done' ? 'success' : job.status === 'failed' ? 'error' : 'processing'
  return (
    <Space>
      <Tag color={color}>{text.plagiarism.status[job.status]}</Tag>
      {job.message ? <Typography.Text type="secondary">{job.message}</Typography.Text> : null}
    </Space>
  )
}
