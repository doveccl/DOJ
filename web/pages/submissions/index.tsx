import { SearchOutlined } from '@ant-design/icons'
import { Button, Card, Checkbox, Flex, Form, Select, Table, Typography } from 'antd'
import type { TableProps } from 'antd'
import { useQuery } from '@tanstack/react-query'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'

import { api, apiData } from '../../client'
import type { SubmissionListItem } from '../../client'
import { ProblemLink, UserLink } from '../../components/entity'
import { ErrorBlock, LoadingBlock } from '../../components/state'
import { SubmissionStatus } from '../../components/status'
import { useRemoteSearch } from '../../components/use-debounced-value'
import { useLocale } from '../../locale'
import type { Lang } from '../../locale'
import { useSession } from '../../session'
import { formatTime, memoryText, problemCode, problemLabel, submissionCode } from '../../utils/format'
import { pageFromParams, pageSizeFromParams, setPageParams } from '../../utils/pagination'

type SubmissionFilters = {
  problem?: number
  user?: string
  language?: string
  status?: string
  assignment?: number
  contest?: number
}

export function SubmissionsPage() {
  const { lang, text } = useLocale()
  const session = useSession()
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const problem = numberParam(params.get('problem'))
  const user = params.get('user') ?? ''
  const language = params.get('language') ?? ''
  const status = params.get('status') ?? ''
  const assignment = numberParam(params.get('assignment'))
  const contest = numberParam(params.get('contest'))
  const onlyMine = session.signedIn && user.toLowerCase() === session.name.toLowerCase()
  const page = pageFromParams(params)
  const pageSize = pageSizeFromParams(params)
  const problemSearch = useRemoteSearch()
  const userSearch = useRemoteSearch()
  const assignmentSearch = useRemoteSearch()
  const contestSearch = useRemoteSearch()
  const problemQuery = problemSearch.searchText || stringParam(problem) || ''
  const assignmentQuery = assignmentSearch.searchText || stringParam(assignment) || ''
  const contestQuery = contestSearch.searchText || stringParam(contest) || ''
  const query = useQuery({
    queryKey: ['submissions', problem, user, language, status, assignment, contest, page, pageSize],
    queryFn: () => apiData(api.GET('/api/submissions', { params: { query: cleanFilters({ problem: stringParam(problem), user, language, status, assignment: stringParam(assignment), contest: stringParam(contest), page, pageSize }) } }))
  })
  const languages = useQuery({ queryKey: ['languages'], queryFn: () => apiData(api.GET('/api/languages')) })
  const problems = useQuery({
    queryKey: ['problems', 'submission-filter', problemQuery],
    queryFn: () => apiData(api.GET('/api/problems', { params: { query: { q: problemQuery, page: 1, pageSize: 50 } } })).then((page) => page.items),
    enabled: problemSearch.active || problem !== undefined
  })
  const users = useQuery({
    queryKey: ['users', 'submission-filter', userSearch.searchText],
    queryFn: () => apiData(api.GET('/api/users', { params: { query: { q: userSearch.searchText } } })),
    enabled: userSearch.active || user.length > 0
  })
  const assignments = useQuery({
    queryKey: ['assignments', 'submission-filter', assignmentQuery],
    queryFn: () => apiData(api.GET('/api/assignments', { params: { query: { q: assignmentQuery, page: 1, pageSize: 50 } } })).then((page) => page.items),
    enabled: assignmentSearch.active || assignment !== undefined
  })
  const contests = useQuery({
    queryKey: ['contests', 'submission-filter', contestQuery],
    queryFn: () => apiData(api.GET('/api/contests', { params: { query: { q: contestQuery, page: 1, pageSize: 50 } } })).then((page) => page.items),
    enabled: contestSearch.active || contest !== undefined
  })
  const languageNames = new Map((languages.data ?? []).map((item) => [item.id, item.name]))
  const languageOptions = (languages.data ?? []).map((item) => ({ value: item.id, label: item.name }))
  const problemOptions = (problems.data ?? []).map((item) => ({ value: item.id, label: problemLabel(item.id, item.title) }))
  const userOptions = (users.data ?? []).map((item) => ({ value: item.name, label: item.name }))
  const assignmentOptions = (assignments.data ?? []).map((item) => ({ value: item.id, label: item.title }))
  const contestOptions = (contests.data ?? []).map((item) => ({ value: item.id, label: item.title }))

  function submit(values: SubmissionFilters) {
    const next = new URLSearchParams()
    if (values.problem !== undefined) {
      next.set('problem', String(values.problem))
    }
    if (values.user) {
      next.set('user', values.user)
    }
    if (values.language) {
      next.set('language', values.language)
    }
    if (values.status) {
      next.set('status', values.status)
    }
    if (values.assignment !== undefined) {
      next.set('assignment', String(values.assignment))
    }
    if (values.contest !== undefined) {
      next.set('contest', String(values.contest))
    }
    setParams(next)
  }

  function clear() {
    setParams(new URLSearchParams())
  }

  function toggleOnlyMine(checked: boolean) {
    const next = new URLSearchParams(params)
    if (checked) {
      next.set('user', session.name)
    } else {
      next.delete('user')
    }
    next.delete('page')
    setParams(next)
  }

  return (
    <Card>
      <Flex vertical gap={16}>
        <Flex className="tableToolbar" justify="space-between" align="center" gap={12} wrap>
          <Form className="tableToolbarForm" layout="inline" initialValues={{ problem, user: user || undefined, language: language || undefined, status: status || undefined, assignment, contest }} onFinish={submit} key={`${problem ?? ''}:${user}:${language}:${status}:${assignment ?? ''}:${contest ?? ''}`}>
            <Form.Item name="problem">
              <Select
                allowClear
                loading={problems.isFetching}
                onOpenChange={problemSearch.setOpen}
                onSearch={problemSearch.setSearch}
                placeholder={text.submissions.problem}
                options={problemOptions}
                showSearch={{ filterOption: false }}
              />
            </Form.Item>
            <Form.Item name="user">
              <Select
                allowClear
                loading={users.isFetching}
                onOpenChange={userSearch.setOpen}
                onSearch={userSearch.setSearch}
                placeholder={text.submissions.user}
                options={userOptions}
                showSearch={{ filterOption: false }}
              />
            </Form.Item>
            <Form.Item name="language">
              <Select allowClear placeholder={text.submissions.language} options={languageOptions} />
            </Form.Item>
            <Form.Item name="status">
              <Select allowClear placeholder={text.submissions.allStatus} options={submissionStatusValues.map((value) => ({ value, label: text.submissions.statuses[value] }))} />
            </Form.Item>
            <Form.Item name="assignment">
              <Select
                allowClear
                loading={assignments.isFetching}
                onOpenChange={assignmentSearch.setOpen}
                onSearch={assignmentSearch.setSearch}
                placeholder={text.assignments.title}
                options={assignmentOptions}
                showSearch={{ filterOption: false }}
              />
            </Form.Item>
            <Form.Item name="contest">
              <Select
                allowClear
                loading={contests.isFetching}
                onOpenChange={contestSearch.setOpen}
                onSearch={contestSearch.setSearch}
                placeholder={text.contests.title}
                options={contestOptions}
                showSearch={{ filterOption: false }}
              />
            </Form.Item>
            <Form.Item>
              <Button onClick={clear}>{text.common.clear}</Button>
            </Form.Item>
            <Form.Item>
              <Button type="primary" htmlType="submit" icon={<SearchOutlined />}>
                {text.common.search}
              </Button>
            </Form.Item>
            {session.signedIn ? (
              <Form.Item>
                <Checkbox checked={onlyMine} onChange={(event) => toggleOnlyMine(event.target.checked)}>
                  {text.submissions.onlyMine}
                </Checkbox>
              </Form.Item>
            ) : null}
          </Form>
        </Flex>
        {query.isError ? (
          <ErrorBlock error={query.error} />
        ) : query.isLoading ? (
          <LoadingBlock />
        ) : (
          <Table<SubmissionListItem>
            rowKey="id"
            scroll={{ x: 1040 }}
            rowClassName="clickableRow"
            onRow={(row) => ({
              role: 'link',
              tabIndex: 0,
              'aria-label': submissionCode(row.id),
              onClick: (event) => {
                const target = event.target as HTMLElement
                if (target.closest('a, button, input, textarea, [role="button"], [role="combobox"]')) {
                  return
                }
                navigate(`/submissions/${row.id}`)
              },
              onKeyDown: (event) => {
                const target = event.target as HTMLElement
                if (target.closest('a, button, input, textarea, [role="button"], [role="combobox"]')) {
                  return
                }
                if (event.key === 'Enter' || event.key === ' ') {
                  event.preventDefault()
                  navigate(`/submissions/${row.id}`)
                }
              }
            })}
            columns={submissionColumns(text, lang, languageNames)}
            dataSource={query.data?.items ?? []}
            pagination={{ current: query.data?.page ?? page, pageSize: query.data?.pageSize ?? pageSize, total: query.data?.total ?? 0, showSizeChanger: true }}
            onChange={(pagination) => setParams(setPageParams(params, pagination.current ?? page, pagination.pageSize ?? pageSize))}
          />
        )}
      </Flex>
    </Card>
  )
}

const submissionStatusValues = ['queued', 'judging', 'AC', 'CE', 'WA', 'PE', 'TLE', 'MLE', 'OLE', 'RE', 'SE'] as const

function cleanFilters<T extends Record<string, string | number | undefined>>(filters: T) {
  return Object.fromEntries(Object.entries(filters).filter(([, value]) => value !== undefined && value !== '')) as T
}

function stringParam(value: number | undefined) {
  return value === undefined ? undefined : String(value)
}

function numberParam(value: string | null) {
  const normalized = (value ?? '').trim().replace(/^(p|c|#)/i, '')
  if (!/^\d+$/.test(normalized)) {
    return undefined
  }
  return Number(normalized)
}

function submissionColumns(text: ReturnType<typeof useLocale>['text'], lang: Lang, languageNames: Map<string, string>): TableProps<SubmissionListItem>['columns'] {
  return [
    {
      title: <Typography.Text className="nowrap">{text.submissions.id}</Typography.Text>,
      dataIndex: 'id',
      render: (id: number) => <Link to={`/submissions/${id}`}>{submissionCode(id)}</Link>
    },
    {
      title: text.submissions.problem,
      width: 360,
      ellipsis: { showTitle: false },
      render: (_, row) => (
        <Flex align="center" className="tableTitleLine">
          <ProblemLink id={row.problemId} title={row.problemTitle} search={row.assignmentId ? `?assignment=${row.assignmentId}` : row.contestId ? `?contest=${row.contestId}` : ''} />
        </Flex>
      )
    },
    {
      title: text.submissions.user,
      dataIndex: 'user',
      width: 180,
      render: (user: string) => <UserLink name={user} />
    },
    {
      title: text.submissions.status,
      dataIndex: 'status',
      width: 150,
      render: (status: string) => <SubmissionStatus status={status} />
    },
    {
      title: text.submissions.score,
      dataIndex: 'score'
    },
    {
      title: text.submissions.time,
      dataIndex: 'timeMs',
      render: (value?: number) => (value === undefined ? '-' : `${value}ms`)
    },
    {
      title: text.submissions.memory,
      dataIndex: 'memoryKb',
      render: (value?: number) => memoryText(value)
    },
    {
      title: text.submissions.language,
      width: 180,
      ellipsis: { showTitle: false },
      render: (_, row) => {
        const language = languageNames.get(row.language) ?? row.language
        return <Typography.Text ellipsis={{ tooltip: language }}>{language}</Typography.Text>
      }
    },
    {
      title: text.submissions.created,
      dataIndex: 'createdAt',
      render: (value: string) => <Typography.Text className="nowrap">{formatTime(value, lang)}</Typography.Text>
    }
  ]
}
