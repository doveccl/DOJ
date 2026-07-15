import {
  DownloadOutlined,
  FolderOpenOutlined,
  ReloadOutlined
} from '@ant-design/icons'
import { App, Button, Flex, Modal, Popconfirm, Progress, Space } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useCallback, useState } from 'react'

import {
  api,
  APIError,
  apiData,
  problemAssetsDownloadURL,
  problemFileDownloadURL,
  uploadProblemAssets
} from '../../../client'
import type { Problem, ProblemAssets } from '../../../client'
import { useLocale } from '../../../locale'
import { problemCode } from '../../../utils/format'
import { downloadURL } from './files'
import { AssetSection } from './section'

export function ProblemAssetsManager({
  id,
  mode,
  open,
  onClose
}: {
  id: number
  mode: Problem['mode']
  open: boolean
  onClose: () => void
}) {
  const { text } = useLocale()
  const { message } = App.useApp()
  const client = useQueryClient()
  const [pendingUpload, setPendingUpload] = useState<{ section: 'data' | 'judge'; files: File[] } | null>(null)
  const [uploadProgress, setUploadProgress] = useState(0)
  const assets = useQuery({
    queryKey: ['problem-assets', id],
    queryFn: () => apiData(api.GET('/api/problems/{id}/assets', { params: { path: { id } } })),
    enabled: Number.isFinite(id) && open
  })
  const showError = (error: unknown) => {
    message.error(error instanceof Error ? error.message : text.common.loadingFailed)
  }
  const setAssets = useCallback((next: ProblemAssets) => {
    client.setQueryData(['problem-assets', id], next)
    void client.invalidateQueries({ queryKey: ['problem', id] })
    void client.invalidateQueries({ queryKey: ['problems'] })
  }, [client, id])
  const removeAsset = useMutation({
    mutationFn: (key: string) => apiData(api.DELETE('/api/problems/{id}/assets/files', {
      params: { path: { id }, query: { key } },
      headers: { 'If-Match': `"${assets.data?.version ?? ''}"` }
    })),
    onSuccess: (next) => {
      setAssets(next)
      message.success(text.common.saved)
    },
    onError: showError
  })
  const upload = useMutation({
    mutationFn: ({ section, files }: { section: 'data' | 'judge'; files: File[] }) => uploadProblemAssets(id, section, files, assets.data?.version ?? '', setUploadProgress),
    onMutate: () => setUploadProgress(0),
    onSuccess: (next) => {
      setPendingUpload(null)
      setAssets(next)
      message.success(text.problem.uploadDone)
    },
    onError: (error, variables) => {
      if (error instanceof APIError && error.status === 412) {
        setPendingUpload(variables)
        void assets.refetch()
      }
      showError(error)
    }
  })
  const score = useMutation({
    mutationFn: ({ caseId, value }: { caseId: string; value: number | null }) => apiData(api.PATCH('/api/problems/{id}/assets/cases/score', {
      params: { path: { id }, query: { case: caseId } },
      body: { score: value },
      headers: { 'If-Match': `"${assets.data?.version ?? ''}"` }
    })),
    onSuccess: (next) => {
      setAssets(next)
      message.success(text.common.saved)
    },
    onError: showError
  })

  return (
    <>
      <Modal
        open={open}
        destroyOnHidden
        footer={null}
        title={text.problem.assets}
        width={{ xs: 'calc(100vw - 32px)', sm: 920 }}
        onCancel={onClose}
      >
        <Flex vertical gap={12}>
          {pendingUpload ? <Button onClick={() => upload.mutate(pendingUpload)}>{text.problem.retryUpload(pendingUpload.files.length)}</Button> : null}
          {upload.isPending ? <Progress percent={uploadProgress} status="active" format={(percent) => percent === 100 ? text.problem.uploadProcessing : `${percent}%`} /> : null}
          <AssetSection
            title={text.problem.data}
            files={assets.data?.data ?? []}
            cases={assets.data?.caseList ?? []}
            section="data"
            loading={assets.isLoading || removeAsset.isPending || upload.isPending || score.isPending}
            onUpload={(files) => upload.mutate({ section: 'data', files })}
            onDownload={(file) => downloadURL(problemFileDownloadURL(id, 'data', file.name), file.name)}
            onDelete={(key) => removeAsset.mutateAsync(key)}
            onScore={(caseId, value) => score.mutate({ caseId, value })}
          />
          {mode === 'custom' ? (
            <AssetSection
              title={text.problem.judge}
              files={assets.data?.judge ?? []}
              section="judge"
              loading={assets.isLoading || removeAsset.isPending || upload.isPending}
              onUpload={(files) => upload.mutate({ section: 'judge', files })}
              onDownload={(file) => downloadURL(problemFileDownloadURL(id, 'judge', file.name), file.name)}
              onDelete={(key) => removeAsset.mutateAsync(key)}
            />
          ) : null}
        </Flex>
      </Modal>
    </>
  )
}

export function ProblemManageActions({
  id,
  rejudgeLoading,
  onOpenAssets,
  onRejudge
}: {
  id: number
  rejudgeLoading: boolean
  onOpenAssets: () => void
  onRejudge: () => void
}) {
  const { text } = useLocale()
  return (
    <Space size={8} wrap>
      <Button icon={<FolderOpenOutlined />} onClick={onOpenAssets}>
        {text.problem.assetManage}
      </Button>
      <Button icon={<DownloadOutlined />} onClick={() => downloadURL(problemAssetsDownloadURL(id), `${problemCode(id)}.zip`)}>
        {text.problem.downloadAssets}
      </Button>
      <Popconfirm title={text.problem.confirmRejudgeAll} okText={text.problem.rejudgeAll} cancelText={text.common.cancel} onConfirm={onRejudge}>
        <Button icon={<ReloadOutlined />} loading={rejudgeLoading}>
          {text.problem.rejudgeAll}
        </Button>
      </Popconfirm>
    </Space>
  )
}
