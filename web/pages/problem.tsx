import {
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  EyeInvisibleOutlined,
  EyeOutlined,
  FileAddOutlined,
  FolderOpenOutlined,
  MoreOutlined,
  ReloadOutlined,
  SendOutlined,
  UploadOutlined
} from '@ant-design/icons'
import {
  App,
  Button,
  Card,
  Checkbox,
  Col,
  Divider,
  Dropdown,
  Flex,
  Form,
  Grid,
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
import type { UploadFile, UploadProps } from 'antd'
import type { MenuProps } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useCallback, useEffect, useState } from 'react'
import type { ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import {
  api,
  apiData,
  apiUrl,
  csrfHeaders,
  problemAssetsDownloadURL,
  problemFileDownloadURL,
  uploadProblemImage
} from '../client'
import type { AssetContent, AssetFile, Language as SubmitLang, Problem, ProblemAssets } from '../client'
import { CodeEditor } from '../components/code'
import { EntityTag } from '../components/entity'
import { JudgeModeSelect } from '../components/judge'
import { LimitInput } from '../components/limit'
import { MarkdownEditor, MarkdownPreview } from '../components/markdown'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { SubmissionStatus } from '../components/status'
import { TagSelect } from '../components/tag-select'
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
  const screens = Grid.useBreakpoint()
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
    queryFn: () => apiData(api.GET('/api/problems/{id}', { params: { path: { id } } })),
    enabled: Number.isFinite(id)
  })
  const site = useQuery({ queryKey: ['site'], queryFn: () => apiData(api.GET('/api/site')) })
  const languages = useQuery({ queryKey: ['languages'], queryFn: () => apiData(api.GET('/api/languages')) })
  const assets = useQuery({
    queryKey: ['problem-assets', id],
    queryFn: () => apiData(api.GET('/api/problems/{id}/assets', { params: { path: { id } } })),
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
      return apiData(api.POST('/api/submissions', { body: { problemId: query.data.id, language: lang, code: source, public: publicSource } }))
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
      return apiData(api.PATCH('/api/problems/{id}', { params: { path: { id } }, body: {
        title: values.title,
        statement: values.statement,
        tags: values.tags ?? [],
        timeMs: values.timeMs,
        memoryMb: values.memoryMb
      } }))
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
      apiData(api.PATCH('/api/problems/{id}/visibility', { params: { path: { id } }, body: { visible: !target.visible } })),
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
      return apiData(api.PATCH('/api/problems/{id}', { params: { path: { id } }, body: { mode } }))
    },
    onSuccess: (_next, mode) => {
      client.setQueryData<Problem>(['problem', id], (current) => (current ? { ...current, mode } : current))
      void client.invalidateQueries({ queryKey: ['problems'] })
      void client.invalidateQueries({ queryKey: ['home'] })
      message.success(text.common.saved)
    },
    onError: showError
  })
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
  const rejudge = useMutation({
    mutationFn: () => apiData(api.POST('/api/problems/{id}/rejudge', { params: { path: { id } } })),
    onSuccess: () => {
      client.setQueryData<Problem>(['problem', id], (current) =>
        current?.latest ? { ...current, latest: { ...current.latest, status: 'queued', score: 0 } } : current
      )
      void client.invalidateQueries({ queryKey: ['problem', id] })
      void client.invalidateQueries({ queryKey: ['problems'] })
      void client.invalidateQueries({ queryKey: ['submissions'] })
      message.success(text.submissions.rejudged)
    },
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
  const manageActions = (
    <ProblemManageActions
      id={id}
      rejudgeLoading={rejudge.isPending}
      compact={!screens.sm}
      onOpenAssets={() => setAssetsOpen(true)}
      onRejudge={() => rejudge.mutate()}
    />
  )

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
                <Flex align="center" gap={10} wrap={false} className="problemHeadTitle">
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
                  <Typography.Text strong className="problemCodeText">
                    {problemCode(problem.id)}
                  </Typography.Text>
                  <Typography.Text strong ellipsis={{ tooltip: problem.title }} className="problemTitleText">
                    {problem.title}
                  </Typography.Text>
                  <Tag color="blue">{formatLimit(problem)}</Tag>
                  {problem.tags.map((tag) => (
                    <EntityTag key={tag}>{tag}</EntityTag>
                  ))}
                </Flex>
              }
              extra={
                session.admin ? (
                  problemEditing ? (
                    <Space size={8} className="problemHeadActions">
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
                ) : null
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
	                    <TagSelect kind="problem" mode="tags" />
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
                <Card title={text.problem.manage} extra={manageActions}>
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
            width={{ xs: 'calc(100vw - 32px)', sm: 920 }}
            onCancel={() => setAssetsOpen(false)}
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
              {problem.mode === 'custom' ? (
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
            <Popconfirm title={text.common.confirmDelete} okText={text.common.delete} cancelText={text.common.cancel} onConfirm={onDeleteAll}>
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
            { title: text.problem.inputFile, render: (_, row) => row.input ? <AssetName file={row.input} /> : null },
            { width: 132, align: 'right', render: (_, row) => row.input ? <AssetActions file={row.input} onEdit={onEdit} onDownload={onDownload} onDelete={onDelete} /> : null },
            { width: 32, align: 'center', render: () => <Divider type="vertical" /> },
            { title: text.problem.outputFile, render: (_, row) => row.output ? <AssetName file={row.output} /> : null },
            { width: 132, align: 'right', render: (_, row) => row.output ? <AssetActions file={row.output} onEdit={onEdit} onDownload={onDownload} onDelete={onDelete} /> : null }
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
            { title: text.problem.judgeFile, render: (_, file) => <AssetName file={file} /> },
            { width: 132, align: 'right', render: (_, file) => <AssetActions file={file} onEdit={onEdit} onDownload={onDownload} onDelete={onDelete} /> }
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

function ProblemManageActions({
  id,
  rejudgeLoading,
  compact,
  onOpenAssets,
  onRejudge
}: {
  id: number
  rejudgeLoading: boolean
  compact: boolean
  onOpenAssets: () => void
  onRejudge: () => void
}) {
  const { text } = useLocale()
  const items: MenuProps['items'] = [
    {
      key: 'assets',
      icon: <FolderOpenOutlined />,
      label: text.problem.assetManage,
      onClick: onOpenAssets
    },
    {
      key: 'download',
      icon: <DownloadOutlined />,
      label: text.problem.downloadAssets,
      onClick: () => downloadURL(problemAssetsDownloadURL(id), `${problemCode(id)}.zip`)
    },
    {
      key: 'rejudge',
      icon: <ReloadOutlined />,
      label: (
        <Popconfirm title={text.problem.confirmRejudgeAll} okText={text.problem.rejudgeAll} cancelText={text.common.cancel} onConfirm={onRejudge}>
          <span>{text.problem.rejudgeAll}</span>
        </Popconfirm>
      )
    }
  ]
  if (compact) {
    return (
      <Dropdown menu={{ items }} trigger={['click']}>
        <Button size="small" icon={<MoreOutlined />} aria-label={text.common.actions} loading={rejudgeLoading} />
      </Dropdown>
    )
  }
  return (
    <Space size={6}>
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

function downloadURL(url: string, filename: string) {
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
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
