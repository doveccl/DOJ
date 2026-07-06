import {
  EditOutlined,
  EyeInvisibleOutlined,
  EyeOutlined,
  SendOutlined,
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
  Row,
  Select,
  Skeleton,
  Space,
  Tag,
  Tooltip,
  Typography,
} from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import {
  api,
  apiData,
  uploadProblemImage
} from '../client'
import type { Problem, ProblemState } from '../client'
import { CodeEditor } from '../components/code'
import { JudgeModeSelect } from '../components/judge'
import { LimitInput } from '../components/limit'
import { MarkdownEditor, MarkdownPreview } from '../components/markdown'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { SubmissionStatus } from '../components/status'
import { TagList } from '../components/tags'
import { TagSelect } from '../components/tag-select'
import { useLocale } from '../locale'
import { useSession } from '../session'
import { formatBytes, formatLimit, problemCode } from '../utils/format'
import { limits } from '../utils/limits'
import { problemAssetUploadMarkdownURL, problemMarkdownID } from '../utils/markdown'
import { ProblemAssetsManager, ProblemManageActions } from './problem/assets'

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
  const [source, setSource] = useState('')
  const [publicSource, setPublicSource] = useState(false)
  const [publicSourceTouched, setPublicSourceTouched] = useState(false)
  const [problemEditing, setProblemEditing] = useState(false)
  const [assetsOpen, setAssetsOpen] = useState(false)
  const uploadStatementImage = useCallback(
    async (file: File) => problemAssetUploadMarkdownURL(await uploadProblemImage(id, file), id),
    [id]
  )
  const statementMarkdownID = problemMarkdownID(id)
  const query = useQuery({
    queryKey: ['problem', id],
    queryFn: () => apiData(api.GET('/api/problems/{id}', { params: { path: { id } } })),
    enabled: Number.isFinite(id)
  })
  const state = useQuery({
    queryKey: ['problem-state', id],
    queryFn: () => apiData(api.GET('/api/problem-state', { params: { query: { ids: String(id) } } })),
    enabled: Number.isFinite(id)
  })
  const site = useQuery({ queryKey: ['site'], queryFn: () => apiData(api.GET('/api/site')) })
  const languages = useQuery({ queryKey: ['languages'], queryFn: () => apiData(api.GET('/api/languages')) })
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
    const stored = typeof window === 'undefined' ? '' : window.localStorage.getItem(languageStorageKey) ?? ''
    const first = items.find((item) => item.id === stored) ?? items[0]
    setLang(first.id)
    if (typeof window !== 'undefined') {
      window.localStorage.setItem(languageStorageKey, first.id)
    }
  }, [lang, languages.data])
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
  const rejudge = useMutation({
    mutationFn: () => apiData(api.POST('/api/problems/{id}/rejudge', { params: { path: { id } } })),
    onSuccess: () => {
      client.setQueryData<ProblemState[]>(['problem-state', id], (current) =>
        current?.map((item) => (item.submission ? { ...item, submission: { ...item.submission, status: 'queued', score: 0 } } : item))
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
  if (state.isError) {
    return <ErrorBlock error={state.error} />
  }

  const problem = query.data as Problem | undefined
  if (!problem) {
    return <ErrorBlock error={text.common.emptyResponse} />
  }
  const langItems = languages.data ?? []
  const langOptions = langItems.map((item) => ({ value: item.id, label: item.name }))
  const selectedLang = langItems.find((item) => item.id === lang)
  const problemState = state.data?.[0]
  const problemStats = state.isLoading
    ? [
        { title: text.problem.record, value: <Skeleton.Input active size="small" style={{ width: 72 }} /> },
        { title: text.problem.pass, value: <Skeleton.Input active size="small" style={{ width: 72 }} /> },
        { title: text.problem.discussion, value: <Skeleton.Input active size="small" style={{ width: 56 }} /> }
      ]
    : [
        ...(problemState?.submission
          ? [{
              title: text.problem.record,
              value: (
                <Link to={`/submissions/${problemState.submission.id}`}>
                  <SubmissionStatus status={problemState.submission.status} />
                </Link>
              )
            }]
          : []),
        {
          title: text.problem.pass,
          value: problemState ? (
            <Link to={`/submissions?problem=${problemCode(problem.id)}`}>
              {`${problemState.submit > 0 ? Math.round((problemState.ac / problemState.submit) * 100) : 0}%`}
            </Link>
          ) : (
            <Skeleton.Input active size="small" style={{ width: 72 }} />
          )
        },
        ...(problemState?.discussions !== undefined
          ? [{ title: text.problem.discussion, value: <Link to={`/discussion?tags=${problemCode(problem.id)}`}>{problemState.discussions}</Link> }]
          : [])
      ]
  const manageActions = (
    <ProblemManageActions
      id={id}
      rejudgeLoading={rejudge.isPending}
      onOpenAssets={() => setAssetsOpen(true)}
      onRejudge={() => rejudge.mutate()}
    />
  )

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
                  <Typography.Text strong ellipsis={{ tooltip: `${problemCode(problem.id)} ${problem.title}` }} className="problemTitleText">
                    {`${problemCode(problem.id)} ${problem.title}`}
                  </Typography.Text>
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
                    <Button aria-label={text.common.edit} size="small" icon={<EditOutlined />} onClick={() => setProblemEditing(true)}>
                      {text.common.edit}
                    </Button>
                  )
                ) : null
              }
            >
              {problemEditing ? (
                <Form<ProblemEditForm>
                  id="problem-edit-form"
                  key={[
                    problem.id,
                    problem.title,
                    problem.timeMs,
                    problem.memoryMb,
                    problem.tags.join(','),
                    problem.statement || `# ${problem.title}`
                  ].join(':')}
                  preserve={false}
                  layout="vertical"
                  initialValues={{
                    title: problem.title,
                    statement: problem.statement || `# ${problem.title}`,
                    tags: problem.tags,
                    timeMs: problem.timeMs,
                    memoryMb: problem.memoryMb
                  }}
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
                    <MarkdownEditor id={statementMarkdownID} upload={uploadStatementImage} />
                  </Form.Item>
                </Form>
              ) : (
                <MarkdownPreview id={statementMarkdownID} value={problem.statement || `# ${problem.title}`} />
              )}
            </Card>
          </Col>
          <Col xs={24} lg={8}>
            <Flex vertical gap={16} className="pageStack">
              <Card
                title={<Tag color="blue">{formatLimit(problem)}</Tag>}
                extra={<TagList tags={problem.tags} />}
              >
                <Flex justify="space-around" gap={16} wrap>
                  {problemStats.map((item) => (
                    <Flex key={item.title} vertical align="center" gap={4}>
                      <Typography.Text type="secondary">{item.title}</Typography.Text>
                      <Typography.Text strong>{item.value}</Typography.Text>
                    </Flex>
                  ))}
                </Flex>
              </Card>
              {session.admin ? (
                <Card title={text.problem.manage} extra={manageActions}>
                  <Flex vertical gap={12}>
                    <Flex align="center" justify="space-between" gap={12}>
                      <Typography.Text type="secondary">{text.problem.mode}</Typography.Text>
                      <JudgeModeSelect
                        size="small"
                        value={problem.mode as 'default' | 'strict' | 'custom'}
                        onChange={(next) => modeUpdate.mutate(next)}
                        disabled={modeUpdate.isPending}
                        loading={modeUpdate.isPending}
                        className="modeSelect"
                      />
                    </Flex>
                    <Flex align="center" justify="space-between" gap={12}>
                      <Typography.Text type="secondary">{text.problem.cases}</Typography.Text>
                      <Typography.Text strong>{problem.cases}</Typography.Text>
                    </Flex>
                    <Flex align="center" justify="space-between" gap={12}>
                      <Typography.Text type="secondary">{text.problem.dataSize}</Typography.Text>
                      <Typography.Text strong>{formatBytes(problem.dataBytes)}</Typography.Text>
                    </Flex>
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
                  setSource(next)
                }}
              />
              <Flex className="submitRow" gap={12} wrap>
                <Space size={12} wrap>
                  <Select
                    value={lang || undefined}
                    options={langOptions}
                    onChange={(next) => {
                      setLang(next)
                      if (typeof window !== 'undefined') {
                        window.localStorage.setItem(languageStorageKey, next)
                      }
                    }}
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
        <ProblemAssetsManager id={id} mode={problem.mode} open={assetsOpen} onClose={() => setAssetsOpen(false)} />
      ) : null}
    </>
  )
}
