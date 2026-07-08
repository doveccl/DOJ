import {
  DownloadOutlined,
  FileAddOutlined,
  FolderOpenOutlined,
  ReloadOutlined
} from '@ant-design/icons'
import { App, Button, Flex, Modal, Popconfirm, Space } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useCallback, useState } from 'react'

import {
  api,
  apiData,
  problemAssetsDownloadURL,
  problemFileDownloadURL
} from '../../../client'
import type { AssetContent, AssetFile, Problem, ProblemAssets } from '../../../client'
import { CodeEditor } from '../../../components/code'
import { useLocale } from '../../../locale'
import { problemCode } from '../../../utils/format'
import { CaseModal } from './case-modal'
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
  const [caseOpen, setCaseOpen] = useState(false)
  const [assetEdit, setAssetEdit] = useState<AssetContent | null>(null)
  const [assetDraft, setAssetDraft] = useState('')
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
    mutationFn: (key: string) => apiData(api.DELETE('/api/problems/{id}/assets/files', { params: { path: { id }, query: { key } } })),
    onSuccess: (next) => {
      setAssets(next)
      message.success(text.common.saved)
    },
    onError: showError
  })
  const removeAssets = useMutation({
    mutationFn: async (files: AssetFile[]) => {
      let next = assets.data
      for (const file of files) {
        next = await apiData(api.DELETE('/api/problems/{id}/assets/files', { params: { path: { id }, query: { key: file.key } } }))
      }
      if (!next) {
        throw new Error(text.common.emptyResponse)
      }
      return next
    },
    onSuccess: (next) => {
      setAssets(next)
      message.success(text.common.saved)
    },
    onError: showError
  })
  const openAsset = useMutation({
    mutationFn: (file: AssetFile) => apiData(api.GET('/api/problems/{id}/assets/files/content', { params: { path: { id }, query: { key: file.key } } })),
    onSuccess: (content) => {
      setAssetEdit(content)
      setAssetDraft(content.content)
    },
    onError: showError
  })
  const saveAsset = useMutation({
    mutationFn: () => {
      if (!assetEdit) {
        throw new Error(text.common.emptyResponse)
      }
      return apiData(api.PATCH('/api/problems/{id}/assets/files/content', { params: { path: { id } }, body: { key: assetEdit.key, content: assetDraft } }))
    },
    onSuccess: (next) => {
      client.setQueryData(['problem-assets', id], next)
      void client.invalidateQueries({ queryKey: ['problem', id] })
      void client.invalidateQueries({ queryKey: ['problems'] })
      message.success(text.common.saved)
      setAssetEdit(null)
      setAssetDraft('')
    },
    onError: showError
  })
  const addCase = useMutation({
    mutationFn: (body: { name: string; input: string; output: string }) => apiData(api.POST('/api/problems/{id}/assets/cases', { params: { path: { id } }, body })),
    onSuccess: (next) => {
      client.setQueryData(['problem-assets', id], next)
      void client.invalidateQueries({ queryKey: ['problem', id] })
      void client.invalidateQueries({ queryKey: ['problems'] })
      message.success(text.common.saved)
      setCaseOpen(false)
    },
    onError: showError
  })
  const fillTemplate = useMutation({
    mutationFn: () => apiData(api.POST('/api/problems/{id}/assets/template', { params: { path: { id } } })),
    onSuccess: (next) => {
      client.setQueryData(['problem-assets', id], next)
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
          <AssetSection
            title={text.problem.data}
            files={assets.data?.data ?? []}
            section="data"
            problemId={id}
            loading={assets.isLoading || removeAsset.isPending || removeAssets.isPending}
            onUploaded={setAssets}
            onEdit={(file) => openAsset.mutate(file)}
            onDownload={(file) => downloadURL(problemFileDownloadURL(id, 'data', file.name), file.name)}
            onDelete={(key) => removeAsset.mutateAsync(key)}
            onDeleteAll={() => removeAssets.mutateAsync(assets.data?.data ?? [])}
            extra={
              <Button size="small" icon={<FileAddOutlined />} onClick={() => setCaseOpen(true)}>
                {text.problem.addCase}
              </Button>
            }
          />
          {mode === 'custom' ? (
            <AssetSection
              title={text.problem.judge}
              files={assets.data?.judge ?? []}
              section="judge"
              problemId={id}
              loading={assets.isLoading || removeAsset.isPending || fillTemplate.isPending}
              onUploaded={setAssets}
              onEdit={(file) => openAsset.mutate(file)}
              onDownload={(file) => downloadURL(problemFileDownloadURL(id, 'judge', file.name), file.name)}
              onDelete={(key) => removeAsset.mutateAsync(key)}
              extra={
                (assets.data?.judge.length ?? 0) === 0 ? (
                  <Button size="small" icon={<FileAddOutlined />} loading={fillTemplate.isPending} onClick={() => fillTemplate.mutate()}>
                    {text.problem.fillTemplate}
                  </Button>
                ) : null
              }
            />
          ) : null}
        </Flex>
      </Modal>
      {caseOpen ? <CaseModal loading={addCase.isPending} onCancel={() => setCaseOpen(false)} onSave={(values) => addCase.mutate(values)} /> : null}
      <Modal
        open={assetEdit !== null}
        destroyOnHidden
        title={assetEdit?.name}
        okText={text.common.save}
        cancelText={text.common.cancel}
        confirmLoading={saveAsset.isPending}
        width={{ xs: 'calc(100vw - 32px)', sm: 860 }}
        onCancel={() => {
          setAssetEdit(null)
          setAssetDraft('')
        }}
        onOk={() => saveAsset.mutate()}
      >
        <CodeEditor
          value={assetDraft}
          language={assetEdit?.name ?? ''}
          minHeight="420px"
          onChange={(next) => setAssetDraft(next)}
        />
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
    <Space size={6} wrap>
      <Button size="small" icon={<FolderOpenOutlined />} onClick={onOpenAssets}>
        {text.problem.assetManage}
      </Button>
      <Button size="small" icon={<DownloadOutlined />} onClick={() => downloadURL(problemAssetsDownloadURL(id), `${problemCode(id)}.zip`)}>
        {text.problem.downloadAssets}
      </Button>
      <Popconfirm title={text.problem.confirmRejudgeAll} okText={text.problem.rejudgeAll} cancelText={text.common.cancel} onConfirm={onRejudge}>
        <Button size="small" icon={<ReloadOutlined />} loading={rejudgeLoading}>
          {text.problem.rejudgeAll}
        </Button>
      </Popconfirm>
    </Space>
  )
}
