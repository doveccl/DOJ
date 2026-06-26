import {
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  EyeInvisibleOutlined,
  EyeOutlined,
  FileAddOutlined,
  FolderOpenOutlined,
  SendOutlined,
  UploadOutlined
} from '@ant-design/icons'
import {
  App,
  Button,
  Card,
  Checkbox,
  Col,
  Flex,
  Form,
  Input,
  Modal,
  Popconfirm,
  Row,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
  Upload
} from 'antd'
import type { UploadProps } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useCallback, useEffect, useState } from 'react'
import type { ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import {
  createProblemCase,
  deleteProblemAsset,
  downloadProblemAssets,
  downloadProblemFile,
  fillJudgeTemplate,
  getLangs,
  getProblem,
  getProblemAssetContent,
  getProblemAssets,
  getSite,
  submitCode,
  updateProblem,
  updateProblemAssetContent,
  updateProblemVisibility,
  uploadProblemImage,
  uploadProblemAsset
} from '../client'
import type { AssetContent, AssetFile, Language as SubmitLang, Problem, ProblemAssets } from '../client'
import { CodeEditor } from '../components/code'
import { JudgeModeSelect } from '../components/judge'
import { LimitInput } from '../components/limit'
import { MarkdownEditor, MarkdownPreview } from '../components/markdown'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { SubmissionStatus } from '../components/status'
import { useLocale } from '../locale'
import { useSession } from '../session'
import { formatBytes, formatLimit, problemCode } from '../utils/format'
import { limits } from '../utils/limits'
import { problemAssetUploadMarkdownURL } from '../utils/markdown'

const sourceTemplate = `#include <bits/stdc++.h>
using namespace std;

int main() {
  long long a, b;
  cin >> a >> b;
  cout << a + b << '\\n';
  return 0;
}
`

const languageStorageKey = 'doj.language'

type ProblemEditForm = {
  title: string
  statement: string
  tags: string[]
  timeMs: number
  memoryMb: number
}

export function ProblemDetailPage() {
  const { text } = useLocale()
  const session = useSession()
  const { message } = App.useApp()
  const client = useQueryClient()
  const navigate = useNavigate()
  const params = useParams()
  const id = Number(params.id)
  const [lang, setLang] = useState('')
  const [source, setSource] = useState(sourceTemplate)
  const [sourceDirty, setSourceDirty] = useState(false)
  const [publicSource, setPublicSource] = useState(false)
  const [publicSourceTouched, setPublicSourceTouched] = useState(false)
  const [problemEditing, setProblemEditing] = useState(false)
  const [assetsOpen, setAssetsOpen] = useState(false)
  const [caseOpen, setCaseOpen] = useState(false)
  const [assetEdit, setAssetEdit] = useState<AssetContent | null>(null)
  const [assetDraft, setAssetDraft] = useState('')
  const uploadStatementImage = useCallback(
    async (file: File) => problemAssetUploadMarkdownURL(await uploadProblemImage(id, file), id),
    [id]
  )
  const statementAssetBase = Number.isFinite(id) ? `/api/problems/${id}/assets/` : undefined
  const query = useQuery({
    queryKey: ['problem', id],
    queryFn: () => getProblem(id),
    enabled: Number.isFinite(id)
  })
  const site = useQuery({ queryKey: ['site'], queryFn: getSite })
  const languages = useQuery({ queryKey: ['languages'], queryFn: getLangs })
  const assets = useQuery({
    queryKey: ['problem-assets', id],
    queryFn: () => getProblemAssets(id),
    enabled: Number.isFinite(id) && session.admin && assetsOpen
  })
  useEffect(() => {
    if (!publicSourceTouched) {
      setPublicSource(site.data?.defaultSubmissionPublic ?? false)
    }
  }, [publicSourceTouched, site.data?.defaultSubmissionPublic])
  useEffect(() => {
    const items = languages.data ?? []
    if (items.length === 0) {
      setLang('')
      return
    }
    if (items.some((item) => item.id === lang)) {
      return
    }
    const stored = readStoredLanguage()
    const first = items.find((item) => item.id === stored) ?? items[0]
    setLang(first.id)
    storeLanguage(first.id)
    if (!sourceDirty) {
      setSource(templateForLang(first))
    }
  }, [lang, languages.data, sourceDirty])
  const showError = (error: unknown) => {
    message.error(error instanceof Error ? error.message : text.common.loadingFailed)
  }
  const submit = useMutation({
    mutationFn: () => {
      if (!query.data) {
        throw new Error(text.common.emptyResponse)
      }
      return submitCode({ problemId: query.data.id, language: lang, code: source, public: publicSource })
    },
    onSuccess: (item) => {
      message.success(text.problem.queued)
      navigate(`/submissions/${item.id}`)
    },
    onError: showError
  })
  const edit = useMutation({
    mutationFn: (values: ProblemEditForm) => {
      if (!query.data) {
        throw new Error(text.common.emptyResponse)
      }
      return updateProblem(id, {
        title: values.title,
        statement: values.statement,
        tags: values.tags ?? [],
        timeMs: values.timeMs,
        memoryMb: values.memoryMb
      })
    },
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['problem', id] })
      void client.invalidateQueries({ queryKey: ['problems'] })
      void client.invalidateQueries({ queryKey: ['home'] })
      message.success(text.common.saved)
      setProblemEditing(false)
    },
    onError: showError
  })
  const visibility = useMutation({
    mutationFn: (target: Problem) =>
      updateProblemVisibility(id, {
        visible: !target.visible
      }),
    onSuccess: (next) => {
      client.setQueryData(['problem', id], (current: Problem | undefined) => ({ ...next, statement: current?.statement }))
      void client.invalidateQueries({ queryKey: ['problems'] })
      void client.invalidateQueries({ queryKey: ['home'] })
      message.success(next.visible ? text.problems.shown : text.problems.hiddenDone)
    },
    onError: showError
  })
  const modeUpdate = useMutation({
    mutationFn: (mode: string) => {
      if (!query.data) {
        throw new Error(text.common.emptyResponse)
      }
      return updateProblem(id, { mode })
    },
    onSuccess: (next) => {
      client.setQueryData(['problem', id], next)
      void client.invalidateQueries({ queryKey: ['problems'] })
      void client.invalidateQueries({ queryKey: ['home'] })
      message.success(text.common.saved)
    },
    onError: showError
  })
  const uploadAsset = useMutation({
    mutationFn: ({ section, file }: { section: 'data' | 'judge' | 'assets'; file: File }) => uploadProblemAsset(id, section, file),
    onSuccess: (next) => {
      client.setQueryData(['problem-assets', id], next)
      void client.invalidateQueries({ queryKey: ['problem', id] })
      void client.invalidateQueries({ queryKey: ['problems'] })
      message.success(text.common.saved)
    },
    onError: showError
  })
  const removeAsset = useMutation({
    mutationFn: (key: string) => deleteProblemAsset(id, key),
    onSuccess: (next) => {
      client.setQueryData(['problem-assets', id], next)
      void client.invalidateQueries({ queryKey: ['problem', id] })
      void client.invalidateQueries({ queryKey: ['problems'] })
      message.success(text.common.saved)
    },
    onError: showError
  })
  const openAsset = useMutation({
    mutationFn: (file: AssetFile) => getProblemAssetContent(id, file.key),
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
      return updateProblemAssetContent(id, { key: assetEdit.key, content: assetDraft })
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
    mutationFn: (values: { name: string; input: string; output: string }) => createProblemCase(id, values),
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
    mutationFn: () => fillJudgeTemplate(id),
    onSuccess: (next) => {
      client.setQueryData(['problem-assets', id], next)
      message.success(text.common.saved)
    },
    onError: showError
  })
  const downloadAssets = useMutation({
    mutationFn: () => downloadProblemAssets(id),
    onSuccess: (blob) => saveBlob(blob, `${problemCode(id)}.zip`),
    onError: showError
  })
  const downloadAsset = useMutation({
    mutationFn: ({ section, file }: { section: 'data' | 'judge'; file: AssetFile }) => downloadProblemFile(id, section, file.name),
    onSuccess: (blob, variables) => saveBlob(blob, variables.file.name),
    onError: showError
  })

  if (query.isLoading) {
    return <LoadingBlock />
  }
  if (query.isError) {
    return <ErrorBlock error={query.error} />
  }

  const problem = query.data as Problem | undefined
  if (!problem) {
    return <ErrorBlock error={text.common.emptyResponse} />
  }
  const langItems = languages.data ?? []
  const langOptions = langItems.map((item) => ({ value: item.id, label: item.name }))
  const selectedLang = langItems.find((item) => item.id === lang)

  function changeLang(next: string) {
    setLang(next)
    storeLanguage(next)
    if (!sourceDirty) {
      setSource(templateForLang((languages.data ?? []).find((item) => item.id === next)))
    }
  }

  function openEdit() {
    setProblemEditing(true)
  }

  return (
    <>
      <Flex vertical gap={20} className="pageStack">
        <Row gutter={[20, 20]} align="top">
          <Col xs={24} lg={16}>
            <Card
              className="statementCard"
              title={
                <Flex vertical gap={8} className="problemHeadTitle">
                  <Flex align="center" gap={10} wrap={false} className="problemTitleLine">
                    {session.admin ? (
                      <Tooltip title={problem.visible ? text.problems.currentVisible : text.problems.currentHidden}>
                        <Button
                          aria-label={`${problem.visible ? text.problems.hide : text.problems.show} ${problemCode(problem.id)}`}
                          type="text"
                          size="small"
                          icon={problem.visible ? <EyeOutlined className="okIcon" /> : <EyeInvisibleOutlined className="mutedIcon" />}
                          loading={visibility.isPending}
                          disabled={visibility.isPending}
                          onClick={() => visibility.mutate(problem)}
                        />
                      </Tooltip>
                    ) : null}
                    <Typography.Text strong>{problemCode(problem.id)}</Typography.Text>
                    <Typography.Text strong ellipsis className="problemTitleText">
                      {problem.title}
                    </Typography.Text>
                  </Flex>
                  <Flex align="center" gap={8} wrap className="problemMetaLine">
                    <Tag color="blue">{formatLimit(problem)}</Tag>
                    {problem.tags.map((tag) => (
                      <Tag key={tag}>{tag}</Tag>
                    ))}
                    {session.admin ? (
                      problemEditing ? (
                        <Space size={8}>
                          <Button size="small" onClick={() => setProblemEditing(false)}>
                            {text.common.cancel}
                          </Button>
                          <Button size="small" type="primary" htmlType="submit" form="problem-edit-form" loading={edit.isPending}>
                            {text.common.save}
                          </Button>
                        </Space>
                      ) : (
                        <Button aria-label={text.common.edit} size="small" icon={<EditOutlined />} onClick={openEdit}>
                          {text.common.edit}
                        </Button>
                      )
                    ) : null}
                  </Flex>
                </Flex>
              }
            >
              {problemEditing ? (
                <Form<ProblemEditForm>
                  id="problem-edit-form"
                  key={problemEditKey(problem)}
                  preserve={false}
                  layout="vertical"
                  initialValues={problemFormValues(problem)}
                  onFinish={(values) => edit.mutate(values)}
                >
                  <Form.Item name="title" label={text.problems.title} rules={[{ required: true, whitespace: true }]}>
                    <Input maxLength={limits.title} showCount />
                  </Form.Item>
                  <Form.Item name="tags" label={text.problems.tag}>
                    <Select mode="tags" tokenSeparators={[',', '，', ' ']} />
                  </Form.Item>
                  <Row gutter={12}>
                    <Col xs={24} md={12}>
                      <Form.Item name="timeMs" label={text.problems.time} rules={[{ required: true }]}>
                        <LimitInput min={100} step={100} unit="ms" />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={12}>
                      <Form.Item name="memoryMb" label={text.problems.memory} rules={[{ required: true }]}>
                        <LimitInput min={16} step={16} unit="MB" />
                      </Form.Item>
                    </Col>
                  </Row>
                  <Form.Item name="statement" label={text.problem.statement} rules={[{ required: true, whitespace: true }]}>
                    <MarkdownEditor minHeight={420} trust="trusted" assetBase={statementAssetBase} upload={uploadStatementImage} />
                  </Form.Item>
                </Form>
              ) : (
                <MarkdownPreview value={problem.statement || `# ${problem.title}`} trust="trusted" assetBase={statementAssetBase} />
              )}
            </Card>
          </Col>
          <Col xs={24} lg={8}>
            <Flex vertical gap={16} className="pageStack">
              <Card>
                <Row gutter={[12, 12]}>
                  {problem.latest ? (
                    <Col span={8}>
                      <ProblemStat title={text.problem.record}>{recordNode(problem)}</ProblemStat>
                    </Col>
                  ) : null}
                  <Col span={problem.latest ? 8 : 12}>
                    <ProblemStat title={text.problem.pass}>
                      <Link to={`/submissions?problem=${problemCode(problem.id)}`}>{passPercent(problem)}</Link>
                    </ProblemStat>
                  </Col>
                  <Col span={problem.latest ? 8 : 12}>
                    <ProblemStat title={text.problem.discussion}>
                      <Link to={`/discussion?tags=${problemCode(problem.id)}`}>{problem.discussions}</Link>
                    </ProblemStat>
                  </Col>
                </Row>
              </Card>
              {session.admin ? (
                <Card
                  title={text.problem.assets}
                  extra={
                    <Space size={6}>
                      <Button size="small" icon={<DownloadOutlined />} loading={downloadAssets.isPending} onClick={() => downloadAssets.mutate()}>
                        {text.problem.downloadAssets}
                      </Button>
                      <Button size="small" icon={<FolderOpenOutlined />} onClick={() => setAssetsOpen(true)}>
                        {text.problem.manage}
                      </Button>
                    </Space>
                  }
                >
                  <Flex vertical gap={12}>
                    <ResourceRow label={text.problem.mode}>
                      <JudgeModeSelect
                        size="small"
                        value={problem.mode as 'default' | 'strict' | 'custom'}
                        onChange={(next) => modeUpdate.mutate(next)}
                        disabled={modeUpdate.isPending}
                        loading={modeUpdate.isPending}
                        className="modeSelect"
                      />
                    </ResourceRow>
                    <ResourceRow label={text.problem.cases}>{problem.cases}</ResourceRow>
                    <ResourceRow label={text.problem.dataSize}>{formatBytes(problem.dataBytes)}</ResourceRow>
                  </Flex>
                </Card>
              ) : null}
            </Flex>
          </Col>
        </Row>
        {session.signedIn ? (
          <Card title={text.problem.submit}>
            <Flex vertical gap={12}>
              <CodeEditor
                value={source}
                language={lang}
                minHeight="360px"
                onChange={(next) => {
                  setSourceDirty(true)
                  setSource(next)
                }}
              />
              <Flex className="submitRow" gap={12} wrap>
                <Space size={12} wrap>
                  <Select
                    value={lang || undefined}
                    options={langOptions}
                    onChange={changeLang}
                    loading={languages.isLoading}
                    placeholder={text.problem.noLanguages}
                    aria-label={text.problem.language}
                    className="submitLang"
                  />
                  <Checkbox
                    checked={publicSource}
                    onChange={(event) => {
                      setPublicSourceTouched(true)
                      setPublicSource(event.target.checked)
                    }}
                  >
                    {text.problem.publicSource}
                  </Checkbox>
                </Space>
                <Button
                  type="primary"
                  icon={<SendOutlined />}
                  loading={submit.isPending}
                  disabled={source.trim() === '' || lang === ''}
                  onClick={() => submit.mutate()}
                >
                  {text.common.submitCode}
                </Button>
              </Flex>
            </Flex>
          </Card>
        ) : null}
      </Flex>
      {session.admin ? (
        <>
          <Modal
            open={assetsOpen}
            destroyOnHidden
            footer={null}
            title={text.problem.assets}
            width={920}
            onCancel={() => setAssetsOpen(false)}
          >
            <Flex vertical gap={12}>
              <AssetSection
                title={text.problem.data}
                files={assets.data?.data ?? []}
                section="data"
                loading={assets.isLoading || uploadAsset.isPending || removeAsset.isPending}
                onUpload={(file) => uploadAsset.mutate({ section: 'data', file })}
                onEdit={(file) => openAsset.mutate(file)}
                onDownload={(file) => downloadAsset.mutate({ section: 'data', file })}
                onDelete={(key) => removeAsset.mutate(key)}
                extra={
                  <Button size="small" icon={<FileAddOutlined />} onClick={() => setCaseOpen(true)}>
                    {text.problem.addCase}
                  </Button>
                }
              />
              <AssetSection
                title={text.problem.judge}
                files={assets.data?.judge ?? []}
                section="judge"
                loading={assets.isLoading || uploadAsset.isPending || removeAsset.isPending || fillTemplate.isPending}
                onUpload={(file) => uploadAsset.mutate({ section: 'judge', file })}
                onEdit={(file) => openAsset.mutate(file)}
                onDownload={(file) => downloadAsset.mutate({ section: 'judge', file })}
                onDelete={(key) => removeAsset.mutate(key)}
                extra={
                  (assets.data?.judge.length ?? 0) === 0 ? (
                    <Button size="small" icon={<FileAddOutlined />} loading={fillTemplate.isPending} onClick={() => fillTemplate.mutate()}>
                      {text.problem.fillTemplate}
                    </Button>
                  ) : null
                }
              />
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
            width={860}
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
      ) : null}
    </>
  )
}

function readStoredLanguage() {
  if (typeof window === 'undefined') {
    return ''
  }
  return window.localStorage.getItem(languageStorageKey) ?? ''
}

function storeLanguage(value: string) {
  if (typeof window === 'undefined') {
    return
  }
  window.localStorage.setItem(languageStorageKey, value)
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
  loading,
  extra,
  onUpload,
  onEdit,
  onDownload,
  onDelete
}: {
  title: string
  files: AssetFile[]
  section: 'data' | 'judge' | 'assets'
  loading: boolean
  extra?: ReactNode
  onUpload: (file: File) => void
  onEdit: (file: AssetFile) => void
  onDownload: (file: AssetFile) => void
  onDelete: (key: string) => void
}) {
  const { text } = useLocale()
  const beforeUpload: UploadProps['beforeUpload'] = (file) => {
    onUpload(file)
    return Upload.LIST_IGNORE
  }
  return (
    <Card
      size="small"
      title={title}
      extra={
        <Space wrap size={6}>
          {extra}
          <Upload beforeUpload={beforeUpload} showUploadList={false} multiple>
            <Button size="small" icon={<UploadOutlined />}>{text.problem.upload}</Button>
          </Upload>
        </Space>
      }
    >
      <Upload.Dragger
        className="assetDrop"
        beforeUpload={beforeUpload}
        showUploadList={false}
        multiple
        openFileDialogOnClick={false}
      >
        <Table<AssetFile>
          rowKey="key"
          size="small"
          loading={loading}
          pagination={false}
          dataSource={files}
          columns={[
            {
              title: section === 'data' ? text.problem.dataFile : section === 'judge' ? text.problem.judgeFile : text.problem.assetFile,
              dataIndex: 'name',
              ellipsis: true,
              render: (name: string) => <Typography.Text>{name}</Typography.Text>
            },
            {
              title: text.problem.size,
              dataIndex: 'size',
              render: (size: number) => <Typography.Text type="secondary">{formatBytes(size)}</Typography.Text>
            },
            {
              title: text.common.actions,
              key: 'actions',
              render: (_, row) => (
                <Space size={4}>
                  {row.editable ? (
                    <Button size="small" type="text" icon={<EditOutlined />} aria-label={text.common.edit} onClick={() => onEdit(row)} />
                  ) : null}
                  <Button size="small" type="text" icon={<DownloadOutlined />} aria-label={text.problem.downloadAssets} onClick={() => onDownload(row)} />
                  <Popconfirm title={text.common.confirmDelete} okText={text.common.delete} cancelText={text.common.cancel} onConfirm={() => onDelete(row.key)}>
                    <Button size="small" danger type="text" icon={<DeleteOutlined />} aria-label={text.common.delete} />
                  </Popconfirm>
                </Space>
              )
            }
          ]}
        />
      </Upload.Dragger>
    </Card>
  )
}

function ProblemStat({ title, children }: { title: string; children: ReactNode }) {
  return (
    <Flex vertical gap={4}>
      <Typography.Text type="secondary">{title}</Typography.Text>
      <Typography.Text strong>{children}</Typography.Text>
    </Flex>
  )
}

function ResourceRow({ label, children }: { label: string; children: ReactNode }) {
  const value = typeof children === 'string' || typeof children === 'number' ? <Typography.Text strong>{children}</Typography.Text> : children
  return (
    <Flex align="center" justify="space-between" gap={12}>
      <Typography.Text type="secondary">{label}</Typography.Text>
      <span className="resourceValue">{value}</span>
    </Flex>
  )
}

function recordNode(problem: Problem) {
  if (!problem.latest) return null
  return (
    <Link to={`/submissions/${problem.latest.id}`}>
      <SubmissionStatus status={problem.latest.status} />
    </Link>
  )
}

function passPercent(problem: Problem) {
  return `${problem.submit > 0 ? Math.round((problem.ac / problem.submit) * 100) : 0}%`
}

function saveBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  anchor.click()
  URL.revokeObjectURL(url)
}

function problemFormValues(problem: Problem): ProblemEditForm {
  return {
    title: problem.title,
    statement: problem.statement || `# ${problem.title}`,
    tags: problem.tags,
    timeMs: problem.timeMs,
    memoryMb: problem.memoryMb
  }
}

function problemEditKey(problem: Problem) {
  return [
    problem.id,
    problem.title,
    problem.timeMs,
    problem.memoryMb,
    problem.tags.join(','),
    problem.statement || `# ${problem.title}`
  ].join(':')
}

function templateForLang(lang?: SubmitLang) {
  const key = `${lang?.id ?? ''} ${lang?.source ?? ''}`.toLowerCase()
  if (key.includes('python') || key.includes('py')) {
    return `a, b = map(int, input().split())
print(a + b)
`
  }
  if (key.includes('go')) {
    return `package main

import "fmt"

func main() {
	var a, b int64
	fmt.Scan(&a, &b)
	fmt.Println(a + b)
}
`
  }
  if (key.includes('rust') || key.includes('rs')) {
    return `use std::io::{self, Read};

fn main() {
    let mut input = String::new();
    io::stdin().read_to_string(&mut input).unwrap();
    let nums: Vec<i64> = input.split_whitespace().map(|s| s.parse().unwrap()).collect();
    println!("{}", nums[0] + nums[1]);
}
`
  }
  if (key.includes('java')) {
    return `import java.util.*;

public class Main {
  public static void main(String[] args) {
    Scanner in = new Scanner(System.in);
    long a = in.nextLong();
    long b = in.nextLong();
    System.out.println(a + b);
  }
}
`
  }
  if (key.includes('javascript') || key.includes('typescript') || key.includes('main.js') || key.includes('main.ts')) {
    return `const fs = require('fs');
const [a, b] = fs.readFileSync(0, 'utf8').trim().split(/\\s+/).map(Number);
console.log(a + b);
`
  }
  return sourceTemplate
}
