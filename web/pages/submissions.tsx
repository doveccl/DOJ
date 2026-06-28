import { SearchOutlined } from '@ant-design/icons'
import { Button, Card, Flex, Form, Select, Table, Tag, Typography } from 'antd'
import type { TableProps } from 'antd'
import { useQuery } from '@tanstack/react-query'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'

import { api, apiData } from '../client'
import type { SubmissionListItem } from '../client'
import { ProblemLink, UserLink } from '../components/entity'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { SubmissionStatus } from '../components/status'
import { useRemoteSearch } from '../components/use-debounced-value'
import { useLocale } from '../locale'
import { formatTime, memoryText, problemCode, problemLabel, submissionCode } from '../utils/format'
import { pageFromParams, pageSizeFromParams, setPageParams } from '../utils/pagination'

type SubmissionFilters = {
  problem?: string
  user?: string
  assignment?: string
  contest?: string
}

export function SubmissionsPage() {
  const { lang, text } = useLocale()
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const problem = normalizeProblemValue(params.get('problem') ?? '')
  const user = params.get('user') ?? ''
  const assignment = params.get('assignment') ?? ''
  const contest = params.get('contest') ?? ''
  const page = pageFromParams(params)
  const pageSize = pageSizeFromParams(params)
  const problemSearch = useRemoteSearch()
  const userSearch = useRemoteSearch()
  const assignmentSearch = useRemoteSearch()
  const contestSearch = useRemoteSearch()
  const query = useQuery({
    queryKey: ['submissions', problem, user, assignment, contest, page, pageSize],
    queryFn: () => apiData(api.GET('/api/submissions', { params: { query: cleanFilters({ problem, user, assignment, contest, page, pageSize }) } }))
  })
  const languages = useQuery({ queryKey: ['languages'], queryFn: () => apiData(api.GET('/api/languages')) })
  const problems = useQuery({
    queryKey: ['problems', 'submission-filter', problemSearch.searchText],
    queryFn: () => apiData(api.GET('/api/problems', { params: { query: { q: problemSearch.searchText, page: 1, pageSize: 50 } } })).then((page) => page.items),
    enabled: problemSearch.active || problem.length > 0
  })
  const users = useQuery({
    queryKey: ['users', 'submission-filter', userSearch.searchText],
    queryFn: () => apiData(api.GET('/api/users', { params: { query: { q: userSearch.searchText } } })),
    enabled: userSearch.active || user.length > 0
  })
  const assignments = useQuery({
    queryKey: ['assignments', 'submission-filter', assignmentSearch.searchText],
    queryFn: () => apiData(api.GET('/api/assignments', { params: { query: { q: assignmentSearch.searchText, page: 1, pageSize: 50 } } })).then((page) => page.items),
    enabled: assignmentSearch.active || assignment.length > 0
  })
  const contests = useQuery({
    queryKey: ['contests', 'submission-filter', contestSearch.searchText],
    queryFn: () => apiData(api.GET('/api/contests', { params: { query: { q: contestSearch.searchText, page: 1, pageSize: 50 } } })).then((page) => page.items),
    enabled: contestSearch.active || contest.length > 0
  })
  const languageNames = new Map((languages.data ?? []).map((item) => [item.id, item.name]))
  const problemOptions = selectFilterOptions(
    problem ? [{ value: problem, label: numericProblem(problem) ? problemCode(Number(problem)) : problem }] : [],
    (problems.data ?? []).map((item) => ({ value: String(item.id), label: problemLabel(item.id, item.title) }))
  )
  const userOptions = selectFilterOptions(user ? [{ value: user, label: user }] : [], (users.data ?? []).map((item) => ({ value: item.name, label: item.name })))
  const assignmentOptions = selectFilterOptions(assignment ? [{ value: assignment, label: `#${assignment}` }] : [], (assignments.data ?? []).map((item) => ({ value: String(item.id), label: item.title })))
  const contestOptions = selectFilterOptions(contest ? [{ value: contest, label: `#${contest}` }] : [], (contests.data ?? []).map((item) => ({ value: String(item.id), label: item.title })))

  function submit(values: SubmissionFilters) {
    const next = new URLSearchParams()
    if (values.problem) {
      next.set('problem', values.problem)
    }
    if (values.user) {
      next.set('user', values.user)
    }
    if (values.assignment) {
      next.set('assignment', values.assignment)
    }
    if (values.contest) {
      next.set('contest', values.contest)
    }
    setParams(next)
  }

  function clear() {
    setParams(new URLSearchParams())
  }

  return (
    <Card>
      <Flex vertical gap={16}>
        <Form layout="inline" initialValues={{ problem: problem || undefined, user: user || undefined, assignment: assignment || undefined, contest: contest || undefined }} onFinish={submit} key={`${problem}:${user}:${assignment}:${contest}`}>
          <Form.Item name="problem">
            <Select
              allowClear
              loading={problems.isFetching}
              onOpenChange={problemSearch.setOpen}
              onSearch={problemSearch.setSearch}
              placeholder={text.submissions.problem}
              options={problemOptions}
              showSearch={{ filterOption: false }}
              style={{ width: 240 }}
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
              style={{ width: 160 }}
            />
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
              style={{ width: 180 }}
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
              style={{ width: 180 }}
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
        </Form>
        {query.isError ? (
          <ErrorBlock error={query.error} />
        ) : query.isLoading ? (
          <LoadingBlock />
        ) : (
          <Table<SubmissionListItem>
            tableLayout="fixed"
            rowKey="id"
            rowClassName="clickableRow"
            onRow={(row) => ({
              onClick: (event) => {
                const target = event.target as HTMLElement
                if (target.closest('a, button, input, textarea, [role="button"], [role="combobox"]')) {
                  return
                }
                navigate(`/submissions/${row.id}`)
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

function cleanFilters<T extends Record<string, string | number | undefined>>(filters: T) {
  return Object.fromEntries(Object.entries(filters).filter(([, value]) => value !== undefined && value !== '')) as T
}

function normalizeProblemValue(value: string) {
  return value.trim().replace(/^p/i, '')
}

function numericProblem(value: string) {
  return /^\d+$/.test(value)
}

function selectFilterOptions(...lists: { value: string; label: string }[][]) {
  const merged = new Map<string, string>()
  for (const list of lists) {
    for (const item of list) {
      if (!merged.has(item.value)) {
        merged.set(item.value, item.label)
      }
    }
  }
  return Array.from(merged, ([value, label]) => ({ value, label }))
}

function submissionColumns(text: ReturnType<typeof useLocale>['text'], lang: string, languageNames: Map<string, string>): TableProps<SubmissionListItem>['columns'] {
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
        <Flex align="center" gap={8} className="tableTitleLine">
          <ProblemLink id={row.problemId} title={row.problemTitle} />
        </Flex>
      )
    },
    {
      title: text.submissions.user,
      dataIndex: 'user',
      render: (user: string) => <UserLink name={user} />
    },
    {
      title: text.submissions.status,
      dataIndex: 'status',
      render: (status: string) => <SubmissionStatus status={status} />
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
      render: (_, row) => <Tag>{languageNames.get(row.language) ?? row.language}</Tag>
    },
    {
      title: text.submissions.created,
      dataIndex: 'createdAt',
      render: (value: string) => <Typography.Text className="nowrap">{formatTime(value, lang)}</Typography.Text>
    }
  ]
}
