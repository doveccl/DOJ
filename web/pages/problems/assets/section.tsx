import { DeleteOutlined, DownloadOutlined, EditOutlined, UploadOutlined } from '@ant-design/icons'
import { App, Button, Card, Divider, Popconfirm, Space, Table, Typography, Upload } from 'antd'
import type { UploadFile, UploadProps } from 'antd'
import { useState } from 'react'
import type { ReactNode } from 'react'

import { apiUrl, csrfHeaders } from '../../../client'
import type { AssetFile, ProblemAssets } from '../../../client'
import { useLocale } from '../../../locale'
import { formatBytes } from '../../../utils/format'
import { acceptedDataFile, dataPairRows } from './files'
import type { DataPairRow } from './files'

export function AssetSection({
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
