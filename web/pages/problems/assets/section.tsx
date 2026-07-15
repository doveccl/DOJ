import { DeleteOutlined, DownloadOutlined, UploadOutlined } from '@ant-design/icons'
import { Button, Card, Flex, InputNumber, Popconfirm, Space, Table, Typography } from 'antd'
import { useRef } from 'react'

import type { PackageCase, PackageFile, ProblemPackage } from '../../../client'
import { useLocale } from '../../../locale'
import { formatBytes } from '../../../utils/format'
import { acceptedDataFile, dataPairRows } from './files'
import type { DataPairRow } from './files'

export function AssetSection({
  title, files, cases, section, loading, disabled, uploadProgress, onUpload, onDownload, onDelete, onClear, onScore
}: {
  title: string
  files: PackageFile[]
  cases?: PackageCase[]
  section: 'data' | 'judge'
  loading: boolean
  disabled: boolean
  uploadProgress?: number
  onUpload: (files: File[]) => void
  onDownload: (file: PackageFile) => void
  onDelete: (key: string) => Promise<ProblemPackage>
  onClear: () => Promise<ProblemPackage>
  onScore?: (id: string, score: number | null) => void
}) {
  const { lang, text } = useLocale()
  const input = useRef<HTMLInputElement>(null)
  const scores = new Map((cases ?? []).map((item) => [item.id, item.score]))
  const tableLoading = {
    spinning: loading,
    percent: uploadProgress,
    description: uploadProgress === undefined ? undefined : uploadProgress === 100 ? text.problem.uploadProcessing : `${uploadProgress}%`
  }
  return (
    <Card
      size="small"
      title={section === 'judge' ? (
        <Flex align="center" gap={6}>
          <span>{title}</span>
          <Typography.Link style={{ fontSize: 14, fontWeight: 400 }} href={`https://github.com/doveccl/DOJ/wiki/Custom-Judge${lang === 'zh' ? '.zh-CN' : ''}`} target="_blank" rel="noreferrer">
            {text.problem.judgeHelpLink}
          </Typography.Link>
        </Flex>
      ) : title}
      extra={
        <>
          <input
            ref={input}
            hidden
            multiple
            type="file"
            accept={section === 'data' ? '.in,.out,.ans,.txt,.zip' : undefined}
            onChange={(event) => {
              const selected = Array.from(event.target.files ?? []).filter((file) => section !== 'data' || file.name.toLowerCase().endsWith('.zip') || acceptedDataFile(file.name))
              if (selected.length > 0) onUpload(selected)
              event.target.value = ''
            }}
          />
          <Space size={6}>
            <Button size="small" disabled={disabled} loading={loading} icon={<UploadOutlined />} onClick={() => input.current?.click()}>
              {text.problem.upload}
            </Button>
            <Popconfirm title={text.common.confirmClear} okText={text.common.clear} cancelText={text.common.cancel} onConfirm={onClear}>
              <Button size="small" danger disabled={disabled || files.length === 0} icon={<DeleteOutlined />}>{text.common.clear}</Button>
            </Popconfirm>
          </Space>
        </>
      }
    >
      {section === 'data' ? (
        <Table<DataPairRow>
          rowKey="key"
          size="small"
          loading={tableLoading}
          dataSource={dataPairRows(files)}
          pagination={{ pageSize: 10, hideOnSinglePage: true, showSizeChanger: false, size: 'small' }}
          tableLayout="auto"
          columns={[
            { title: text.problem.inputFile, render: (_, row) => row.input ? <AssetCell file={row.input} disabled={disabled} onDownload={onDownload} onDelete={onDelete} /> : null },
            { title: text.problem.outputFile, render: (_, row) => row.output ? <AssetCell file={row.output} disabled={disabled} onDownload={onDownload} onDelete={onDelete} /> : null },
            {
              title: text.problem.score,
              width: 92,
              render: (_, row) => scores.has(row.key) && onScore ? (
                <InputNumber
                  key={`${row.key}:${scores.get(row.key)}`}
                  min={0}
                  placeholder="10"
                  size="small"
                  disabled={disabled}
                  defaultValue={scores.get(row.key) ?? undefined}
                  onBlur={(event) => {
                    const raw = event.target.value.trim()
                    if (raw === '' && scores.get(row.key) != null) onScore(row.key, null)
                    const value = Number(raw)
                    if (raw !== '' && Number.isFinite(value) && value !== scores.get(row.key)) onScore(row.key, value)
                  }}
                />
              ) : null
            }
          ]}
        />
      ) : (
        <Table<PackageFile>
          rowKey="key"
          size="small"
          loading={tableLoading}
          dataSource={files}
          pagination={false}
          tableLayout="auto"
          columns={[
            { title: text.problem.judgeFile, render: (_, file) => <AssetCell file={file} disabled={disabled} onDownload={onDownload} onDelete={onDelete} /> }
          ]}
        />
      )}
    </Card>
  )
}

function AssetCell({ file, disabled, onDownload, onDelete }: { file: PackageFile; disabled: boolean; onDownload: (file: PackageFile) => void; onDelete: (key: string) => Promise<ProblemPackage> }) {
  return (
    <Flex align="center" justify="space-between" gap={8}>
      <AssetName file={file} />
      <AssetActions file={file} disabled={disabled} onDownload={onDownload} onDelete={onDelete} />
    </Flex>
  )
}

function AssetName({ file }: { file: PackageFile }) {
  return <Typography.Text style={{ minWidth: 0 }} ellipsis={{ tooltip: `${file.name} (${formatBytes(file.size)})` }}>{file.name} <Typography.Text type="secondary">({formatBytes(file.size)})</Typography.Text></Typography.Text>
}

function AssetActions({ file, disabled, onDownload, onDelete }: { file: PackageFile; disabled: boolean; onDownload: (file: PackageFile) => void; onDelete: (key: string) => Promise<ProblemPackage> }) {
  const { text } = useLocale()
  return (
    <Space size={4}>
      <Button size="small" type="text" icon={<DownloadOutlined />} aria-label={text.common.download} onClick={() => onDownload(file)} />
      <Popconfirm title={text.common.confirmDelete} okText={text.common.delete} cancelText={text.common.cancel} onConfirm={() => onDelete(file.key)}>
        <Button size="small" type="text" danger disabled={disabled} icon={<DeleteOutlined />} aria-label={text.common.delete} />
      </Popconfirm>
    </Space>
  )
}
