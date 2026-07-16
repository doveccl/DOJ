import { UnorderedListOutlined } from '@ant-design/icons'
import { Alert, App as AntApp, Button, Card, Flex, Form, Space, Table, Tabs, Tag, Tooltip, Typography } from 'antd'
import type { TableProps } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import dayjs from 'dayjs'
import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'

import { api, apiData } from '../../client'
import type { ProblemListItem, ProblemState, RankUser } from '../../client'
import { ScheduledDetailHeader } from '../../components/scheduled-detail-header'
import { ProblemLink, UserLink } from '../../components/entity'
import { defaultProblemSort } from '../../components/problem-ref'
import { ErrorBlock, LoadingBlock } from '../../components/state'
import { ContestKindTag } from '../../components/status'
import { contestTarget, ScheduleTag } from '../../components/time'
import { useLocale } from '../../locale'
import { useSession } from '../../session'
import { usePageTitle } from '../../title'
import { formatTime, problemCode, problemLabel } from '../../utils/format'
import { ContestFormFields } from './form'
import type { ContestFormValues } from './form'

export function ContestDetailPage() {
  const { lang, text } = useLocale()
  const session = useSession()
  const { message, modal } = AntApp.useApp()
  const client = useQueryClient()
  const params = useParams()
  const id = Number(params.id)
  const [editOpen, setEditOpen] = useState(false)
  const [editForm] = Form.useForm<ContestFormValues>()
  const query = useQuery({
    queryKey: ['contest', id],
    queryFn: () => apiData(api.GET('/api/contests/{id}', { params: { path: { id } } })),
    enabled: Number.isFinite(id)
  })
  const problemIds = query.data?.problems.map((item) => item.id).join(',') ?? ''
  const state = useQuery({
    queryKey: ['problem-state', 'contest', id, problemIds],
    queryFn: () => apiData(api.GET('/api/problem-state', { params: { query: { ids: problemIds, contest: id } } })),
    enabled: Number.isFinite(id) && problemIds.length > 0
  })
  usePageTitle(query.data?.contest.title)
  const showError = (error: unknown) => {
    message.error(error instanceof Error ? error.message : text.common.loadingFailed)
  }
  const update = useMutation({
    mutationFn: (values: ContestFormValues) =>
      apiData(api.PATCH('/api/contests/{id}', { params: { path: { id } }, body: {
        title: values.title,
        description: values.description,
        kind: values.kind,
        startAt: values.startAt.toISOString(),
        endAt: values.endAt.toISOString(),
        freezeAt: values.kind === 'ICPC' ? (values.freezeAt?.toISOString() ?? '') : '',
        problems: values.problems ?? []
      } })),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['contests'] })
      void client.invalidateQueries({ queryKey: ['contest', id] })
      void client.invalidateQueries({ queryKey: ['home'] })
      setEditOpen(false)
      message.success(text.common.saved)
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
  if (!query.data) {
    return <ErrorBlock error={text.common.emptyResponse} />
  }

  const { contest, problems, rank } = query.data
  const oiRankHidden = contest.kind === 'OI' && contest.status === 'running' && !session.admin
  const recordsUser = session.signedIn && !session.admin ? session.name : ''
  const stateByProblem = new Map((state.data ?? []).map((item) => [item.problemId, item]))
  const problemOptions = problems.map((item) => ({
    value: item.id,
    label: problemLabel(item.id, item.title)
  }))
  const editFormId = `contest-${contest.id}-edit-form`
  const save = (values: ContestFormValues) => {
    if (contest.status === 'pending') {
      update.mutate(values)
      return
    }
    modal.confirm({
      title: text.contests.changeWarning,
      content: text.contests.changeWarningDescription,
      okText: text.common.save,
      cancelText: text.common.cancel,
      onOk: () => update.mutate(values)
    })
  }

  return (
    <Flex vertical gap={16}>
      <ScheduledDetailHeader
        descriptionId={`contest-${contest.id}-description`}
        descriptionValue={query.data.description}
        status={
          <ScheduleTag
            kind="contest"
            status={contest.status}
            target={contestTarget(contest.status, contest.startAt, contest.endAt)}
            range={`${formatTime(contest.startAt, lang)} - ${formatTime(contest.endAt, lang)}`}
            onFinish={() => void query.refetch()}
          />
        }
        title={contest.title}
        titleTag={<ContestKindTag kind={contest.kind} />}
        recordsHref={`/submissions?contest=${contest.id}${recordsUser ? `&user=${encodeURIComponent(recordsUser)}` : ''}`}
        recordsLabel={recordsUser ? text.submissions.myRecords : text.submissions.allRecords}
        admin={session.admin}
        editing={editOpen}
        onStartEdit={() => setEditOpen(true)}
        onCancelEdit={() => setEditOpen(false)}
        saving={update.isPending}
        editFormId={editFormId}
      >
        <Form<ContestFormValues>
          id={editFormId}
          key={editFormId}
          form={editForm}
          preserve={false}
          layout="vertical"
          initialValues={{
            title: contest.title,
            description: query.data.description,
            kind: contest.kind,
            startAt: dayjs(contest.startAt),
            endAt: dayjs(contest.endAt),
            freezeAt: contest.freezeAt ? dayjs(contest.freezeAt) : null,
            problems: problems.map((problem, index) => ({ id: problem.id, sort: problem.sort || defaultProblemSort(index) }))
          }}
          onFinish={save}
        >
          <ContestFormFields form={editForm} editorId={`contest-${contest.id}-description-edit`} problemOptions={problemOptions} />
        </Form>
      </ScheduledDetailHeader>
      <Card>
        <Tabs
          items={[
            {
              key: 'problems',
              label: text.contests.problems,
              children: <Table<ProblemListItem> rowKey="id" columns={problemColumns(text, contest.id, contest.status === 'ended', recordsUser, stateByProblem)} dataSource={problems} pagination={false} loading={state.isLoading} scroll={{ x: 640 }} />
            },
            {
              key: 'rank',
              label: text.contests.rank,
              children: (
                <Flex vertical gap={12}>
                  {contest.status === 'frozen' ? <Alert type={session.admin ? 'info' : 'warning'} showIcon title={session.admin ? text.contests.realtimeRank : text.contests.frozenRank} /> : null}
                  {oiRankHidden ? (
                    <Alert type="info" showIcon title={text.contests.oiRankHidden} />
                  ) : (
                    <Table<RankUser>
                      rowKey="rank"
                      columns={rankColumns(text, contest.kind, problems)}
                      dataSource={rank}
                      pagination={false}
                      scroll={{ x: 'max-content' }}
                      locale={{ emptyText: text.contests.emptyRank }}
                    />
                  )}
                </Flex>
              )
            }
          ]}
        />
      </Card>
    </Flex>
  )
}

function rankColumns(text: ReturnType<typeof useLocale>['text'], kind: string, problems: ProblemListItem[]): TableProps<RankUser>['columns'] {
  const columns: NonNullable<TableProps<RankUser>['columns']> = [
    {
      title: text.rank.rank,
      dataIndex: 'rank',
      width: 72,
      align: 'center'
    },
    {
      title: text.rank.user,
      width: 200,
      render: (_, row) => <UserLink name={row.user} avatar={row.avatar} />
    }
  ]
  if (kind === 'OI') {
    columns.splice(2, 0, {
      title: text.rank.score,
      dataIndex: 'score',
      width: 88,
      align: 'center'
    })
    return [...columns, ...rankProblemColumns(kind, problems)]
  }
  columns.splice(
    2,
    0,
    {
      title: text.rank.ac,
      dataIndex: 'ac',
      width: 80,
      align: 'center'
    },
    {
      title: text.rank.penalty,
      dataIndex: 'penalty',
      width: 100,
      align: 'center'
    }
  )
  return [...columns, ...rankProblemColumns(kind, problems)]
}

function rankProblemColumns(kind: string, problems: ProblemListItem[]): NonNullable<TableProps<RankUser>['columns']> {
  return problems.map((problem) => ({
    key: `problem-${problem.id}`,
    align: 'center',
    title: (
      <Tooltip title={problemLabel(problem.id, problem.title)}>
        <span>{problem.sort || problemCode(problem.id)}</span>
      </Tooltip>
    ),
    render: (_, row) => <RankProblemCell kind={kind} item={row.problems.find((item) => item.problemId === problem.id)} />
  }))
}

function RankProblemCell({ kind, item }: { kind: string; item?: RankUser['problems'][number] }) {
  if (!item || item.status === 'none') {
    return <Typography.Text type="secondary">-</Typography.Text>
  }
  if (kind === 'ICPC') {
    if (item.status === 'pending') {
      return <Tag color="processing">{item.submit > 0 ? `?${item.submit}` : '?'}</Tag>
    }
    if (item.status === 'ac') {
      const wrong = Math.max(0, item.submit - 1)
      return (
        <Space orientation="vertical" size={0} align="center">
          <Tag color="success">{wrong > 0 ? `+${wrong}` : '+'}</Tag>
          <Typography.Text type="secondary">{item.penalty}</Typography.Text>
        </Space>
      )
    }
    return <Tag color="error">-{item.submit}</Tag>
  }
  if (item.status === 'pending') {
    return <Tag color="processing">?</Tag>
  }
  if (item.status === 'ac') {
    return <Tag color="success">{item.score}</Tag>
  }
  return <Tag color={item.score > 0 ? 'warning' : 'error'}>{item.score}</Tag>
}

function problemColumns(text: ReturnType<typeof useLocale>['text'], contestID: number, contestEnded: boolean, recordsUser: string, state: Map<number, ProblemState>): TableProps<ProblemListItem>['columns'] {
  const columns: TableProps<ProblemListItem>['columns'] = [
    {
      title: text.contests.status,
      width: 140,
      render: (_, row) => <ContestProblemStatus record={state.get(row.id)} text={text} />
    },
    {
      title: text.common.sort,
      dataIndex: 'sort',
      width: 88,
      render: (sort: string | undefined, row) => <Typography.Text>{sort || problemCode(row.id)}</Typography.Text>
    },
    {
      title: text.submissions.problem,
      dataIndex: 'title',
      ellipsis: { showTitle: false },
      render: (title: string, row) => <ProblemLink id={row.id} title={title} search={contestEnded ? '' : `?contest=${contestID}`} />
    },
    {
      title: text.common.actions,
      width: 80,
      align: 'right',
      render: (_, row) => (
        <Tooltip title={text.submissions.viewProblemRecords}>
          <Button aria-label={`${text.submissions.viewProblemRecords} ${problemCode(row.id)}`} type="text" icon={<UnorderedListOutlined />} href={`/submissions?contest=${contestID}&problem=${row.id}${recordsUser ? `&user=${encodeURIComponent(recordsUser)}` : ''}`} />
        </Tooltip>
      )
    }
  ]
  return columns
}

function ContestProblemStatus({ record, text }: { record?: ProblemState; text: ReturnType<typeof useLocale>['text'] }) {
  const status = record?.status
  if (status === 'pending') {
    return <Tag color="processing">{text.submissions.statuses.pending}</Tag>
  }
  if (status === 'ac') {
    return <Tag color="success">{text.contests.completed}</Tag>
  }
  if (status === 'tried') {
    return <Tag color="warning">{text.contests.attempted}</Tag>
  }
  return <Tag>{text.contests.notCompleted}</Tag>
}
