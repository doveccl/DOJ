import { ExportOutlined, PlayCircleOutlined, ReloadOutlined } from '@ant-design/icons'
import { Alert, Button, Flex, Space, Spin, Tag, Typography } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { api, apiData, apiUrl } from '../client'
import type { PlagiarismJob } from '../client'
import { useLocale } from '../locale'

type Props = {
  scope: 'assignment' | 'contest'
  id: number
}

export function PlagiarismPanel({ scope, id }: Props) {
  const { text } = useLocale()
  const queryClient = useQueryClient()
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
  const reportViewerUrl = job?.reportUrl
    ? `/jplag/${job.id}?file=${encodeURIComponent(apiUrl(job.reportUrl).toString())}`
    : ''

  return (
    <Flex vertical gap={12}>
      <Flex justify="space-between" align="center" gap={12} wrap>
        <Space>
          <Button type="primary" icon={<PlayCircleOutlined />} loading={create.isPending} onClick={() => create.mutate()}>
            {text.plagiarism.run}
          </Button>
          <Button icon={<ReloadOutlined />} onClick={() => void jobs.refetch()}>
            {text.common.refresh}
          </Button>
        </Space>
        {busy ? <Space><Spin size="small" /> <Typography.Text type="secondary">{text.plagiarism.running}</Typography.Text></Space> : job ? <PlagiarismStatus job={job} /> : null}
      </Flex>
      {create.isError ? <Alert type="error" showIcon message={create.error instanceof Error ? create.error.message : text.common.loadingFailed} /> : null}
      {busy ? null : job?.status === 'failed' ? (
        <Alert type="error" showIcon message={text.plagiarism.failed} description={cleanFailure(job.message) || text.plagiarism.failed} />
      ) : job && reportViewerUrl ? (
        <Alert
          type="success"
          showIcon
          message={job.message}
          description={text.plagiarism.openHint}
          action={
            <Button icon={<ExportOutlined />} href={reportViewerUrl} target="_blank" rel="noreferrer">
              {text.plagiarism.openReport}
            </Button>
          }
        />
      ) : (
        <Alert type="info" showIcon message={text.plagiarism.empty} />
      )}
    </Flex>
  )
}

function PlagiarismStatus({ job }: { job: PlagiarismJob }) {
  const { text } = useLocale()
  const color = job.status === 'done' ? 'success' : job.status === 'failed' ? 'error' : 'processing'
  return <Tag color={color}>{text.plagiarism.status[job.status]}</Tag>
}

function cleanFailure(message: string) {
  let clean = message.trim()
  const jsonStart = clean.indexOf('{')
  if (jsonStart >= 0) {
    try {
      const body = JSON.parse(clean.slice(jsonStart)) as { message?: string }
      clean = body.message || clean
    } catch {
      // Keep the original message.
    }
  }
  return clean
    .replace(/^JPlag failed:\s*/, '')
    .replace(/^exit status \d+:\s*/, '')
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => line && !line.includes('JPlagVersionChecker'))
    .join('\n')
}
