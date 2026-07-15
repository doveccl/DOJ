import {
  DownloadOutlined,
  FolderOpenOutlined,
  ReloadOutlined
} from '@ant-design/icons'
import { App, Button, Flex, Modal, Popconfirm, Space } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useCallback, useState } from 'react'

import {
  api,
  APIError,
  apiData,
  problemArchiveDownloadURL,
  problemPackageFileURL,
  uploadProblemPackage
} from '../../../client'
import type { Problem, ProblemPackage } from '../../../client'
import { useLocale } from '../../../locale'
import { problemCode } from '../../../utils/format'
import { downloadURL } from './files'
import { AssetSection } from './section'

export function ProblemPackageManager({
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
  const pkg = useQuery({
    queryKey: ['problem-package', id],
    queryFn: () => apiData(api.GET('/api/problems/{id}/package', { params: { path: { id } } })),
    enabled: Number.isFinite(id) && open
  })
  const showError = (error: unknown) => {
    message.error(error instanceof Error ? error.message : text.common.loadingFailed)
  }
  const setPackage = useCallback((next: ProblemPackage) => {
    client.setQueryData(['problem-package', id], next)
    void client.invalidateQueries({ queryKey: ['problem', id] })
    void client.invalidateQueries({ queryKey: ['problems'] })
  }, [client, id])
  const removeFile = useMutation({
    mutationFn: (key: string) => apiData(api.DELETE('/api/problems/{id}/package/files', {
      params: { path: { id }, query: { key } },
      headers: { 'If-Match': `"${pkg.data?.version ?? ''}"` }
    })),
    onSuccess: (next) => {
      setPackage(next)
      message.success(text.common.saved)
    },
    onError: showError
  })
  const upload = useMutation({
    mutationFn: ({ section, files }: { section: 'data' | 'judge'; files: File[] }) => uploadProblemPackage(id, section, files, pkg.data?.version ?? '', setUploadProgress),
    onMutate: () => setUploadProgress(0),
    onSuccess: (next) => {
      setPendingUpload(null)
      setPackage(next)
      message.success(text.problem.uploadDone)
    },
    onError: (error, variables) => {
      if (error instanceof APIError && error.status === 412) {
        setPendingUpload(variables)
        void pkg.refetch()
      }
      showError(error)
    }
  })
  const score = useMutation({
    mutationFn: ({ caseId, value }: { caseId: string; value: number | null }) => apiData(api.PATCH('/api/problems/{id}/package/cases/score', {
      params: { path: { id }, query: { case: caseId } },
      body: { score: value },
      headers: { 'If-Match': `"${pkg.data?.version ?? ''}"` }
    })),
    onSuccess: (next) => {
      setPackage(next)
      message.success(text.common.saved)
    },
    onError: showError
  })
  const uploadSection = upload.isPending ? upload.variables.section : null
  const disabled = pkg.isLoading || removeFile.isPending || upload.isPending || score.isPending

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
          <AssetSection
            title={text.problem.data}
            files={pkg.data?.data ?? []}
            cases={pkg.data?.caseList ?? []}
            section="data"
            loading={pkg.isLoading || removeFile.isPending || uploadSection === 'data' || score.isPending}
            disabled={disabled}
            uploadProgress={upload.isPending && upload.variables.section === 'data' ? uploadProgress : undefined}
            onUpload={(files) => upload.mutate({ section: 'data', files })}
            onDownload={(file) => downloadURL(problemPackageFileURL(id, 'data', file.name), file.name)}
            onDelete={(key) => removeFile.mutateAsync(key)}
            onClear={() => removeFile.mutateAsync('data')}
            onScore={(caseId, value) => score.mutate({ caseId, value })}
          />
          {mode === 'custom' ? (
            <AssetSection
              title={text.problem.judge}
              files={pkg.data?.judge ?? []}
              section="judge"
              loading={pkg.isLoading || removeFile.isPending || uploadSection === 'judge'}
              disabled={disabled}
              uploadProgress={upload.isPending && upload.variables.section === 'judge' ? uploadProgress : undefined}
              onUpload={(files) => upload.mutate({ section: 'judge', files })}
              onDownload={(file) => downloadURL(problemPackageFileURL(id, 'judge', file.name), file.name)}
              onDelete={(key) => removeFile.mutateAsync(key)}
              onClear={() => removeFile.mutateAsync('judge')}
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
      <Button icon={<DownloadOutlined />} onClick={() => downloadURL(problemArchiveDownloadURL(id), `${problemCode(id)}.zip`)}>
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
