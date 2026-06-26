import { SearchOutlined } from '@ant-design/icons'
import { Button, Card, Flex, Form, Select, Table, Tag, Typography } from 'antd'
import type { TableProps } from 'antd'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'

import { getAssignments, getContests, getLangs, getProblems, getSubmissions, searchUsers } from '../client'
import type { Submission } from '../client'
import { ProblemLink, UserLink } from '../components/entity'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { SubmissionStatus } from '../components/status'
import { useDebouncedValue } from '../components/use-debounced-value'
import { useLocale } from '../locale'
import { formatTime, memoryText, problemCode, problemLabel, submissionCode } from '../utils/format'

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
  const [problemOpen, setProblemOpen] = useState(false)
  const [userOpen, setUserOpen] = useState(false)
  const [assignmentOpen, setAssignmentOpen] = useState(false)
  const [contestOpen, setContestOpen] = useState(false)
  const [problemSearch, setProblemSearch] = useState('')
  const [userSearch, setUserSearch] = useState('')
  const [assignmentSearch, setAssignmentSearch] = useState('')
  const [contestSearch, setContestSearch] = useState('')
  const problemSearchText = useDebouncedValue(problemSearch.trim())
  const userSearchText = useDebouncedValue(userSearch.trim())
  const assignmentSearchText = useDebouncedValue(assignmentSearch.trim())
  const contestSearchText = useDebouncedValue(contestSearch.trim())
  const query = useQuery({
    queryKey: ['submissions', problem, user, assignment, contest],
    queryFn: () => getSubmissions(cleanFilters({ problem, user, assignment, contest }))
  })
  const languages = useQuery({ queryKey: ['languages'], queryFn: getLangs })
  const problems = useQuery({
    queryKey: ['problems', 'submission-filter', problemSearchText],
    queryFn: () => getProblems({ q: problemSearchText }),
    enabled: problemOpen && problemSearchText.length > 0
  })
  const users = useQuery({
    queryKey: ['users', 'submission-filter', userSearchText],
    queryFn: () => searchUsers({ q: userSearchText }),
    enabled: userOpen && userSearchText.length > 0
  })
  const assignments = useQuery({
    queryKey: ['assignments', 'submission-filter', assignmentSearchText],
    queryFn: () => getAssignments({ q: assignmentSearchText }),
    enabled: assignmentOpen && assignmentSearchText.length > 0
  })
  const contests = useQuery({
    queryKey: ['contests', 'submission-filter', contestSearchText],
    queryFn: () => getContests({ q: contestSearchText }),
    enabled: contestOpen && contestSearchText.length > 0
  })
  const languageNames = new Map((languages.data ?? []).map((item) => [item.id, item.name]))
  const problemOptions = mergeProblemOptions(
    problem ? [{ value: problem, label: numericProblem(problem) ? problemCode(Number(problem)) : problem }] : [],
    (problems.data ?? []).map((item) => ({ value: String(item.id), label: problemLabel(item.id, item.title) }))
  )
  const userOptions = mergeProblemOptions(user ? [{ value: user, label: user }] : [], (users.data ?? []).map((item) => ({ value: item.name, label: item.name })))
  const assignmentOptions = mergeProblemOptions(assignment ? [{ value: assignment, label: `#${assignment}` }] : [], (assignments.data ?? []).map((item) => ({ value: String(item.id), label: item.title })))
  const contestOptions = mergeProblemOptions(contest ? [{ value: contest, label: `#${contest}` }] : [], (contests.data ?? []).map((item) => ({ value: String(item.id), label: item.title })))

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
              showSearch
              allowClear
              filterOption={false}
              loading={problems.isFetching}
              onOpenChange={setProblemOpen}
              onSearch={setProblemSearch}
              placeholder={text.submissions.problem}
              options={problemOptions}
              style={{ width: 240 }}
            />
          </Form.Item>
          <Form.Item name="user">
            <Select
              showSearch
              allowClear
              filterOption={false}
              loading={users.isFetching}
              onOpenChange={setUserOpen}
              onSearch={setUserSearch}
              placeholder={text.submissions.user}
              options={userOptions}
              style={{ width: 160 }}
            />
          </Form.Item>
          <Form.Item name="assignment">
            <Select
              showSearch
              allowClear
              filterOption={false}
              loading={assignments.isFetching}
              onOpenChange={setAssignmentOpen}
              onSearch={setAssignmentSearch}
              placeholder={text.assignments.title}
              options={assignmentOptions}
              style={{ width: 180 }}
            />
          </Form.Item>
          <Form.Item name="contest">
            <Select
              showSearch
              allowClear
              filterOption={false}
              loading={contests.isFetching}
              onOpenChange={setContestOpen}
              onSearch={setContestSearch}
              placeholder={text.contests.title}
              options={contestOptions}
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
          <Table<Submission>
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
            dataSource={query.data}
            pagination={{ pageSize: 20, showSizeChanger: true }}
          />
        )}
      </Flex>
    </Card>
  )
}

function cleanFilters(filters: SubmissionFilters) {
  return Object.fromEntries(Object.entries(filters).filter(([, value]) => value)) as SubmissionFilters
}

function normalizeProblemValue(value: string) {
  return value.trim().replace(/^p/i, '')
}

function numericProblem(value: string) {
  return /^\d+$/.test(value)
}

function mergeProblemOptions(...lists: { value: string; label: string }[][]) {
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

function submissionColumns(text: ReturnType<typeof useLocale>['text'], lang: string, languageNames: Map<string, string>): TableProps<Submission>['columns'] {
  return [
    {
      title: text.submissions.id,
      dataIndex: 'id',
      render: (id: number) => <Link to={`/submissions/${id}`}>{submissionCode(id)}</Link>
    },
    {
      title: text.submissions.problem,
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
