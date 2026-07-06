import {
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  FileAddOutlined,
  FolderOpenOutlined,
  ReloadOutlined,
  UploadOutlined
} from '@ant-design/icons'
import { App, Button, Card, Divider, Flex, Form, Input, Modal, Popconfirm, Space, Table, Typography, Upload } from 'antd'
import type { UploadFile, UploadProps } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useCallback, useState } from 'react'
import type { ReactNode } from 'react'

import {
  api,
  apiData,
  apiUrl,
  csrfHeaders,
  problemAssetsDownloadURL,
  problemFileDownloadURL
} from '../../client'
import type { AssetContent, AssetFile, Problem, ProblemAssets } from '../../client'
import { CodeEditor } from '../../components/code'
import { useLocale } from '../../locale'
import { formatBytes, problemCode } from '../../utils/format'

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

function CaseModal({
  loading,
  onCancel,
  onSave
}: {
  loading: boolean
  onCancel: () => void
  onSave: (values: { name: string; input: string; output: string }) => void
}) {
  const { text } = useLocale()
  const [form] = Form.useForm<{ name: string; input: string; output: string }>()

  return (
    <Modal
      open
      destroyOnHidden
      title={text.problem.addCase}
      okText={text.common.save}
      cancelText={text.common.cancel}
      confirmLoading={loading}
      onCancel={onCancel}
      onOk={() => form.submit()}
    >
      <Form form={form} preserve={false} layout="vertical" initialValues={{ name: '', input: '', output: '' }} onFinish={onSave}>
        <Form.Item name="name" label={text.problem.caseName}>
          <Input placeholder={text.problem.caseNamePlaceholder} />
        </Form.Item>
        <Form.Item name="input" label={`${text.problem.caseName}.in`}>
          <Input.TextArea rows={5} />
        </Form.Item>
        <Form.Item name="output" label={`${text.problem.caseName}.out`}>
          <Input.TextArea rows={5} />
        </Form.Item>
      </Form>
    </Modal>
  )
}

function AssetSection({
  title,
  files,
  section,
  problemId,
  loading,
  extra,
  onUploaded,
  onEdit,
  onDownload,
  onDelete,
  onDeleteAll
}: {
  title: string
  files: AssetFile[]
  section: 'data' | 'judge' | 'assets'
  problemId: number
  loading: boolean
  extra?: ReactNode
  onUploaded: (assets: ProblemAssets) => void
  onEdit: (file: AssetFile) => void
  onDownload: (file: AssetFile) => void
  onDelete: (key: string) => Promise<ProblemAssets>
  onDeleteAll?: () => Promise<ProblemAssets>
}) {
  const { text } = useLocale()
  const { message } = App.useApp()
  const [uploadFiles, setUploadFiles] = useState<UploadFile<ProblemAssets>[]>([])
  const uploadKey = `asset-upload-${section}`
  const accept = section === 'data' ? '.in,.out,.ans,.txt' : undefined
  const beforeUpload: UploadProps<ProblemAssets>['beforeUpload'] = (file) => {
    if (accept && !acceptedDataFile(file.name)) {
      message.error(`${file.name}: ${accept}`)
      return Upload.LIST_IGNORE
    }
    return true
  }
  const onChange: UploadProps<ProblemAssets>['onChange'] = ({ file, fileList }) => {
    const currentFiles = fileList.filter((item) => item.status !== 'removed')
    setUploadFiles(currentFiles.some((item) => item.status === 'uploading') ? currentFiles : [])
    if (file.status === 'uploading') {
      const done = currentFiles.filter((item) => item.status === 'done').length
      message.open({
        key: uploadKey,
        type: 'loading',
        content: currentFiles.length > 1 ? text.problem.uploadedCount(done, currentFiles.length) : text.problem.uploadedPercent(Math.round(file.percent ?? 0)),
        duration: 0
      })
      return
    }
    if (file.status === 'done' && file.response) {
      onUploaded(file.response)
      if (currentFiles.every((item) => item.status === 'done')) {
        message.success({ key: uploadKey, content: text.problem.uploadDone })
      }
    } else if (file.status === 'error') {
      message.error({ key: uploadKey, content: file.error?.message || text.common.loadingFailed })
    }
  }
  return (
    <Card
      size="small"
      title={title}
      extra={
        <Space wrap size={6}>
          <Upload<ProblemAssets>
            accept={accept}
            action={apiUrl(`/api/problems/${problemId}/assets/files`).toString()}
            beforeUpload={beforeUpload}
            data={{ section }}
            disabled={loading}
            fileList={uploadFiles}
            headers={csrfHeaders()}
            withCredentials
            multiple
            progress={{ showInfo: true }}
            onChange={onChange}
            showUploadList={false}
          >
            <Button size="small" disabled={loading} loading={loading} icon={<UploadOutlined />}>{text.problem.upload}</Button>
          </Upload>
          {extra}
          {onDeleteAll ? (
            <Popconfirm title={text.common.confirmClear} okText={text.common.clear} cancelText={text.common.cancel} onConfirm={onDeleteAll}>
              <Button size="small" danger disabled={loading || files.length === 0} loading={loading} icon={<DeleteOutlined />}>{text.common.clear}</Button>
            </Popconfirm>
          ) : null}
        </Space>
      }
    >
      {section === 'data' ? (
        <Table<DataPairRow>
          rowKey="key"
          size="small"
          showHeader={false}
          loading={loading}
          dataSource={dataPairRows(files)}
          pagination={{ pageSize: 10, hideOnSinglePage: true, showSizeChanger: false, size: 'small' }}
          scroll={{ x: 760 }}
          columns={[
            { title: text.problem.inputFile, width: 280, ellipsis: { showTitle: false }, render: (_, row) => row.input ? <AssetName file={row.input} /> : null },
            { align: 'right', render: (_, row) => row.input ? <AssetActions file={row.input} onEdit={onEdit} onDownload={onDownload} onDelete={onDelete} /> : null },
            { align: 'center', render: () => <Divider type="vertical" /> },
            { title: text.problem.outputFile, width: 280, ellipsis: { showTitle: false }, render: (_, row) => row.output ? <AssetName file={row.output} /> : null },
            { align: 'right', render: (_, row) => row.output ? <AssetActions file={row.output} onEdit={onEdit} onDownload={onDownload} onDelete={onDelete} /> : null }
          ]}
        />
      ) : (
        <Table<AssetFile>
          rowKey="key"
          size="small"
          loading={loading}
          dataSource={files}
          pagination={false}
          scroll={{ x: 520 }}
          columns={[
            { title: text.problem.judgeFile, width: 360, ellipsis: { showTitle: false }, render: (_, file) => <AssetName file={file} /> },
            { align: 'right', render: (_, file) => <AssetActions file={file} onEdit={onEdit} onDownload={onDownload} onDelete={onDelete} /> }
          ]}
        />
      )}
    </Card>
  )
}

function AssetName({ file }: { file: AssetFile }) {
  return (
    <Typography.Text ellipsis={{ tooltip: `${file.name} (${formatBytes(file.size)})` }}>
      {file.name} <Typography.Text type="secondary">({formatBytes(file.size)})</Typography.Text>
    </Typography.Text>
  )
}

function AssetActions({
  file,
  onEdit,
  onDownload,
  onDelete
}: {
  file: AssetFile
  onEdit: (file: AssetFile) => void
  onDownload: (file: AssetFile) => void
  onDelete: (key: string) => Promise<ProblemAssets>
}) {
  const { text } = useLocale()
  return (
    <Space size={4}>
      {file.editable ? <Button size="small" type="text" icon={<EditOutlined />} aria-label={text.common.edit} onClick={() => onEdit(file)} /> : null}
      <Button size="small" type="text" icon={<DownloadOutlined />} aria-label={text.common.download} onClick={() => onDownload(file)} />
      <Popconfirm title={text.common.confirmDelete} okText={text.common.delete} cancelText={text.common.cancel} onConfirm={() => onDelete(file.key)}>
        <Button size="small" type="text" danger icon={<DeleteOutlined />} aria-label={text.common.delete} />
      </Popconfirm>
    </Space>
  )
}

type DataPairRow = { key: string; input?: AssetFile; output?: AssetFile }

function dataPairRows(files: AssetFile[]) {
  const rows: DataPairRow[] = []
  const byStem = new Map<string, DataPairRow>()
  for (const file of files) {
    const { stem, kind } = dataCaseStem(file.name)
    if (stem === '' || kind === '') {
      rows.push({ key: file.key, input: file })
      continue
    }
    let row = byStem.get(stem)
    if (!row) {
      row = { key: stem }
      byStem.set(stem, row)
      rows.push(row)
    }
    if (kind === 'in') {
      row.input = file
    } else {
      row.output = file
    }
  }
  return rows
}

function dataCaseStem(name: string) {
  const base = name.split('/').pop() ?? name
  const lower = base.toLowerCase()
  const stem = base.match(/\d+/)?.[0]
  if (stem) {
    if (lower.includes('in')) {
      return { stem, kind: 'in' }
    }
    if (lower.includes('out') || lower.includes('ans')) {
      return { stem, kind: 'out' }
    }
  }
  const index = base.lastIndexOf('.')
  if (index <= 0) {
    return { stem: '', kind: '' }
  }
  switch (lower.slice(index)) {
    case '.in':
      return { stem: base.slice(0, index), kind: 'in' }
    case '.out':
    case '.ans':
      return { stem: base.slice(0, index), kind: 'out' }
    default:
      return { stem: '', kind: '' }
  }
}

function acceptedDataFile(name: string) {
  return ['.in', '.out', '.ans', '.txt'].some((suffix) => name.toLowerCase().endsWith(suffix))
}

function downloadURL(url: string, filename: string) {
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
}
