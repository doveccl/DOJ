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
  Popconfirm,
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
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom'

import {
  APIError,
  api,
  apiData,
  uploadProblemImage
} from '../../client'
import type { Problem } from '../../client'
import { CodeEditor } from '../../components/code'
import { JudgeModeSelect } from '../../components/judge'
import { MarkdownPreview } from '../../components/markdown'
import { ErrorBlock, LoadingBlock } from '../../components/state'
import { ProblemStatus } from '../../components/status'
import { TagList } from '../../components/tags'
import { useLocale } from '../../locale'
import { useSession } from '../../session'
import { submissionDraftKey } from '../../utils/draft'
import { formatBytes, formatLimit, problemCode } from '../../utils/format'
import { problemAssetUploadMarkdownURL, problemMarkdownID } from '../../utils/markdown'
import { ProblemAssetsManager, ProblemManageActions } from './assets'
import { ProblemFormFields } from './form'
import type { ProblemFormValues } from './form'

const languageStorageKey = 'doj.language'

export function ProblemDetailPage() {
  const { text } = useLocale()
  const session = useSession()
  const { message, modal } = App.useApp()
  const client = useQueryClient()
  const navigate = useNavigate()
  const params = useParams()
  const [searchParams] = useSearchParams()
  const id = Number(params.id)
  const assignmentParam = searchParams.get('assignment')
  const contestParam = searchParams.get('contest')
  const assignmentID = positiveID(assignmentParam)
  const contestID = positiveID(contestParam)
  const invalidContext = (assignmentParam !== null && assignmentID === undefined) || (contestParam !== null && contestID === undefined) || (assignmentID !== undefined && contestID !== undefined)
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
    queryKey: ['problem-state', id, assignmentID, contestID],
    queryFn: () => apiData(api.GET('/api/problem-state', { params: { query: { ids: String(id), assignment: assignmentID, contest: contestID } } })),
    enabled: Number.isFinite(id) && !invalidContext
  })
  const site = useQuery({ queryKey: ['site'], queryFn: () => apiData(api.GET('/api/site')) })
  const languages = useQuery({ queryKey: ['languages'], queryFn: () => apiData(api.GET('/api/languages')) })
  const draftKey = session.signedIn && Number.isFinite(id) && lang
    ? submissionDraftKey(session.name, id, lang, assignmentID, contestID)
    : ''
  useEffect(() => {
    setSource(draftKey ? window.localStorage.getItem(draftKey) ?? '' : '')
  }, [draftKey])
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
      return apiData(api.POST('/api/submissions', { body: {
        problemId: query.data.id,
        assignmentId: assignmentID,
        contestId: contestID,
        language: lang,
        code: source,
        public: publicSource
      } }))
    },
    onSuccess: (item) => {
      message.success(text.problem.queued)
      navigate(`/submissions/${item.id}`)
    },
    onError: (error) => {
      if (
        error instanceof APIError &&
        error.status === 403 &&
        (error.message === 'assignment has ended' || error.message === 'contest is not running')
      ) {
        modal.confirm({
          title: text.problem.contextClosed,
          content: text.problem.contextClosedDescription,
          okText: text.problem.switchToPractice,
          cancelText: text.common.cancel,
          onOk: () => {
            window.localStorage.setItem(submissionDraftKey(session.name, id, lang), source)
            navigate(`/problems/${id}`, { replace: true })
          }
        })
        return
      }
      showError(error)
    }
  })
  const edit = useMutation({
    mutationFn: (values: ProblemFormValues) => {
      if (!query.data) {
        throw new Error(text.common.emptyResponse)
      }
      return apiData(api.PATCH('/api/problems/{id}', { params: { path: { id } }, body: {
        title: values.title,
        statement: values.statement ?? '',
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
    mutationFn: (visible: boolean) =>
      apiData(api.PATCH('/api/problems/{id}/visibility', { params: { path: { id } }, body: { visible } })),
    onSuccess: (next, visible) => {
      client.setQueryData(['problem', id], (current: Problem | undefined) => ({ ...next, statement: current?.statement }))
      void client.invalidateQueries({ queryKey: ['problems'] })
      void client.invalidateQueries({ queryKey: ['home'] })
      message.success(visible ? text.problems.shown : text.problems.hiddenDone)
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
      void client.invalidateQueries({ queryKey: ['problem-state'] })
      void client.invalidateQueries({ queryKey: ['problem', id] })
      void client.invalidateQueries({ queryKey: ['problems'] })
      void client.invalidateQueries({ queryKey: ['submissions'] })
      message.success(text.submissions.rejudged)
    },
    onError: showError
  })

  if (invalidContext) {
    return <ErrorBlock error={text.problem.invalidContext} />
  }
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
  const myProblemSubmissions = `/submissions?problem=${problemCode(problem.id)}${session.signedIn ? `&user=${encodeURIComponent(session.name)}` : ''}`
  const problemStats = state.isLoading
    ? [
        { title: text.problem.record, value: <Skeleton.Input active size="small" style={{ width: 72 }} /> },
        { title: text.problem.pass, value: <Skeleton.Input active size="small" style={{ width: 72 }} /> },
        { title: text.problem.discussion, value: <Skeleton.Input active size="small" style={{ width: 56 }} /> }
      ]
    : [
        {
          title: text.problem.record,
          value: problemState ? (
            <Link to={myProblemSubmissions}>
              <ProblemStatus status={problemState.status} />
            </Link>
          ) : (
            <Skeleton.Input active size="small" style={{ width: 72 }} />
          )
        },
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
                    <Tooltip title={problem.visible ? text.problems.hide : text.problems.show}>
                      <Button
                        aria-label={`${problem.visible ? text.problems.hide : text.problems.show} ${problemCode(problem.id)}`}
                        type="text"
                        size="small"
                        icon={problem.visible ? <EyeOutlined className="okIcon" /> : <EyeInvisibleOutlined className="mutedIcon" />}
                        loading={visibility.isPending}
                        disabled={visibility.isPending}
                        onClick={() => visibility.mutate(!problem.visible)}
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
                      <Button onClick={() => setProblemEditing(false)}>
                        {text.common.cancel}
                      </Button>
                      <Button type="primary" htmlType="submit" form="problem-edit-form" loading={edit.isPending}>
                        {text.common.save}
                      </Button>
                    </Space>
                  ) : (
                    <Button aria-label={text.common.edit} icon={<EditOutlined />} onClick={() => setProblemEditing(true)}>
                      {text.common.edit}
                    </Button>
                  )
                ) : null
              }
            >
              {problemEditing ? (
                <Form<ProblemFormValues>
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
                    mode: problem.mode,
                    timeMs: problem.timeMs,
                    memoryMb: problem.memoryMb
                  }}
                  onFinish={(values) => edit.mutate(values)}
                >
                  <ProblemFormFields statement={{ editorId: statementMarkdownID, upload: uploadStatementImage }} />
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
                        value={problem.mode as 'default' | 'strict' | 'custom'}
                        onChange={(next) => modeUpdate.mutate(next)}
                        disabled={modeUpdate.isPending}
                        loading={modeUpdate.isPending}
                        className="modeSelect"
                      />
                    </Flex>
                    <Flex align="center" justify="space-between" gap={12}>
                      <Typography.Text type="secondary">{text.problem.cases}</Typography.Text>
                      <Typography.Text strong>{problem.cases ?? 0}</Typography.Text>
                    </Flex>
                    <Flex align="center" justify="space-between" gap={12}>
                      <Typography.Text type="secondary">{text.problem.dataSize}</Typography.Text>
                      <Typography.Text strong>{formatBytes(problem.dataBytes ?? 0)}</Typography.Text>
                    </Flex>
                  </Flex>
                </Card>
              ) : null}
            </Flex>
          </Col>
        </Row>
        {session.signedIn ? (
          <Card
            title={
              <Flex align="center" gap={8} wrap>
                <Typography.Text strong>{text.problem.submit}</Typography.Text>
                {assignmentID !== undefined ? <Tag color="blue" style={{ marginInlineEnd: 0 }}><Link to={`/assignments/${assignmentID}`}>{`${text.assignments.title} #${assignmentID}`}</Link></Tag> : null}
                {contestID !== undefined ? <Tag color="purple" style={{ marginInlineEnd: 0 }}><Link to={`/contests/${contestID}`}>{`${text.contests.title} #${contestID}`}</Link></Tag> : null}
              </Flex>
            }
          >
            <Flex vertical gap={12}>
              <CodeEditor
                value={source}
                language={lang}
                minHeight="360px"
                onChange={(next) => {
                  setSource(next)
                  if (!draftKey) {
                    return
                  }
                  if (next) {
                    window.localStorage.setItem(draftKey, next)
                  } else {
                    window.localStorage.removeItem(draftKey)
                  }
                }}
              />
              <Flex align="center" justify="space-between" gap={12} wrap style={{ width: '100%' }}>
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

function positiveID(value: string | null) {
  if (value === null || !/^\d+$/.test(value)) {
    return undefined
  }
  const id = Number(value)
  return Number.isSafeInteger(id) && id > 0 ? id : undefined
}
