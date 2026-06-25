import { DeleteOutlined, EyeInvisibleOutlined, EyeOutlined, PlusOutlined, SearchOutlined } from '@ant-design/icons'
import {
  App as AntApp,
  Button,
  Card,
  Flex,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography
} from 'antd'
import type { TableProps } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'

import { createProblem, deleteProblem, getProblems, updateProblemVisibility } from '../client'
import type { Problem } from '../client'
import { ProblemLink } from '../components/entity'
import { JudgeModeSelect } from '../components/judge'
import { LimitInput } from '../components/limit'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { useLocale } from '../locale'
import { useSession } from '../session'
import { formatLimit, formatPass, problemCode } from '../utils/format'
import { limits } from '../utils/limits'

type ProblemForm = {
  title: string
  tags?: string[]
  mode: string
  timeMs: number
  memoryMb: number
}

export function ProblemsPage() {
  const { text } = useLocale()
  const session = useSession()
  const { message } = AntApp.useApp()
  const client = useQueryClient()
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const [open, setOpen] = useState(false)
  const q = params.get('q') ?? ''
  const tag = params.get('tag') ?? ''
  const query = useQuery({ queryKey: ['problems', q, tag], queryFn: () => getProblems({ q, tag }) })
  const showError = (error: unknown) => {
    message.error(error instanceof Error ? error.message : text.common.loadingFailed)
  }
  const create = useMutation({
    mutationFn: (values: ProblemForm) =>
      createProblem({
        title: values.title,
        tags: values.tags ?? [],
        visible: true,
        mode: values.mode,
        timeMs: values.timeMs,
        memoryMb: values.memoryMb
      }),
    onSuccess: (item) => {
      void client.invalidateQueries({ queryKey: ['problems'] })
      void client.invalidateQueries({ queryKey: ['home'] })
      message.success(text.common.saved)
      closeModal()
      navigate(`/problems/${item.id}`)
    },
    onError: showError
  })
  const remove = useMutation({
    mutationFn: deleteProblem,
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['problems'] })
      void client.invalidateQueries({ queryKey: ['home'] })
      message.success(text.common.saved)
    },
    onError: showError
  })
  const visibility = useMutation({
    mutationFn: (item: Problem) =>
      updateProblemVisibility(item.id, {
        visible: !item.visible
      }),
    onSuccess: (item) => {
      replaceProblemInCaches(client, item)
      void client.invalidateQueries({ queryKey: ['problems'] })
      void client.invalidateQueries({ queryKey: ['home'] })
      message.success(item.visible ? text.problems.shown : text.problems.hiddenDone)
    },
    onError: showError
  })
  const columns = problemColumns(text, {
    remove: (id) => remove.mutate(id),
    toggle: (item) => visibility.mutate(item),
    toggling: (item) => visibility.isPending && visibility.variables?.id === item.id
  }, session.admin)

  const allTags = Array.from(new Set((query.data ?? []).flatMap((item) => item.tags))).map((item) => ({
    label: item,
    value: item
  }))

  function submit(values: { q?: string; tag?: string }) {
    const next = new URLSearchParams()
    if (values.q) {
      next.set('q', values.q)
    }
    if (values.tag) {
      next.set('tag', values.tag)
    }
    setParams(next)
  }

  function clear() {
    setParams(new URLSearchParams())
  }

  function openCreate() {
    setOpen(true)
  }

  function closeModal() {
    setOpen(false)
  }

  function save(values: ProblemForm) {
    create.mutate(values)
  }

  return (
    <Card>
      <Flex vertical gap={16}>
        <Form layout="inline" initialValues={{ q: q || undefined, tag: tag || undefined }} onFinish={submit} key={`${q}:${tag}`}>
          <Form.Item name="q">
            <Input placeholder={text.problems.q} allowClear style={{ width: 280 }} />
          </Form.Item>
          <Form.Item name="tag">
            <Select showSearch optionFilterProp="label" placeholder={text.problems.tag} allowClear options={allTags} style={{ width: 220 }} />
          </Form.Item>
          <Form.Item>
            <Button onClick={clear}>{text.common.clear}</Button>
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" icon={<SearchOutlined />}>
              {text.common.search}
            </Button>
          </Form.Item>
          {session.admin ? (
            <Form.Item>
              <Button icon={<PlusOutlined />} onClick={openCreate}>
                {text.common.createProblem}
              </Button>
            </Form.Item>
          ) : null}
        </Form>
        {query.isError ? (
          <ErrorBlock error={query.error} />
        ) : query.isLoading ? (
          <LoadingBlock />
        ) : (
          <Table<Problem>
            rowKey="id"
            columns={columns}
            dataSource={query.data}
            pagination={{ pageSize: 20, showSizeChanger: true }}
          />
        )}
      </Flex>
      {session.admin && open ? (
        <ProblemModal loading={create.isPending} onCancel={closeModal} onSave={save} />
      ) : null}
    </Card>
  )
}

function ProblemModal({
  loading,
  onCancel,
  onSave
}: {
  loading: boolean
  onCancel: () => void
  onSave: (values: ProblemForm) => void
}) {
  const { text } = useLocale()
  const [form] = Form.useForm<ProblemForm>()
  const initialValues = { tags: [], mode: 'default', timeMs: 1000, memoryMb: 256 }

  return (
    <Modal
      open
      destroyOnHidden
      title={text.common.createProblem}
      okText={text.common.create}
      cancelText={text.common.cancel}
      confirmLoading={loading}
      onCancel={onCancel}
      onOk={() => form.submit()}
    >
      <Form<ProblemForm> form={form} preserve={false} layout="vertical" initialValues={initialValues} onFinish={onSave}>
        <Form.Item name="title" label={text.problems.title} rules={[{ required: true, whitespace: true }]}>
          <Input maxLength={limits.title} showCount />
        </Form.Item>
        <Form.Item name="tags" label={text.problems.tag}>
          <Select mode="tags" tokenSeparators={[',', '，', ' ']} />
        </Form.Item>
        <Form.Item name="mode" label={text.problem.mode}>
          <JudgeModeSelect />
        </Form.Item>
        <Space size={12} style={{ width: '100%' }} align="start">
          <Form.Item name="timeMs" label={text.problems.time} rules={[{ required: true }]}>
            <LimitInput min={100} step={100} unit="ms" />
          </Form.Item>
          <Form.Item name="memoryMb" label={text.problems.memory} rules={[{ required: true }]}>
            <LimitInput min={16} step={16} unit="MB" />
          </Form.Item>
        </Space>
      </Form>
    </Modal>
  )
}

function problemColumns(
  text: ReturnType<typeof useLocale>['text'],
  actions: {
    remove: (id: number) => void
    toggle: (item: Problem) => void
    toggling: (item: Problem) => boolean
  },
  admin: boolean
): TableProps<Problem>['columns'] {
  const columns: TableProps<Problem>['columns'] = [
    {
      title: text.submissions.problem,
      dataIndex: 'title',
      render: (title: string, row) => (
        <Flex align="center" gap={8} className="tableTitleLine">
          <ProblemLink id={row.id} title={title} />
          <ProblemRecordTag mine={row.mine} />
          {!row.visible ? <Tag>{text.problems.hidden}</Tag> : null}
        </Flex>
      )
    },
    {
      title: text.problems.tag,
      dataIndex: 'tags',
      render: (tags: string[]) => (
        <Space size={[0, 4]} wrap>
          {tags.map((tag) => (
            <Tag key={tag}>{tag}</Tag>
          ))}
        </Space>
      )
    },
    {
      title: text.problems.limit,
      render: (_, row) => <Typography.Text type="secondary">{formatLimit(row)}</Typography.Text>
    },
    {
      title: text.problems.pass,
      render: (_, row) => <Typography.Text>{formatPass(row)}</Typography.Text>
    }
  ]
  if (admin) {
    columns.push({
      title: text.common.actions,
      align: 'right',
      render: (_, row) => (
        <Space size={4}>
          <Tooltip title={row.visible ? text.problems.currentVisible : text.problems.currentHidden}>
            <Button
              aria-label={`${row.visible ? text.problems.hide : text.problems.show} ${problemCode(row.id)}`}
              type="text"
              icon={row.visible ? <EyeOutlined className="okIcon" /> : <EyeInvisibleOutlined className="mutedIcon" />}
              loading={actions.toggling(row)}
              disabled={actions.toggling(row)}
              onClick={(event) => {
                event.stopPropagation()
                actions.toggle(row)
              }}
            />
          </Tooltip>
          <Popconfirm title={text.common.confirmDelete} okText={text.common.delete} cancelText={text.common.cancel} onConfirm={() => actions.remove(row.id)}>
            <Button aria-label={`${text.common.delete} ${problemCode(row.id)}`} type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      )
    })
  }
  return columns
}

function ProblemRecordTag({ mine }: { mine?: string }) {
  const { text } = useLocale()
  if (mine === 'ac') {
    return <Tag color="success">{text.problem.passed}</Tag>
  }
  if (mine === 'tried') {
    return <Tag color="warning">{text.problem.tried}</Tag>
  }
  return null
}

function replaceProblemInCaches(client: ReturnType<typeof useQueryClient>, item: Problem) {
  client.setQueriesData<Problem[]>({ queryKey: ['problems'] }, (old) => {
    if (!old) {
      return old
    }
    return old.map((row) => (row.id === item.id ? item : row))
  })
}
