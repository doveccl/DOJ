import { useContext, useEffect, useState } from 'react'
import { message, Button, Card, Divider, Form, Input, Modal, Popconfirm, Table, Tag, TablePaginationConfig } from 'antd'
import { parseMemory, parseTime } from '../../../common/function'
import { ProblemForm } from '../../component/form/problem'
import { delProblem, getProblems, postProblem, putProblem, rejudgeSubmission } from '../../model'
import { IProblem } from '../../interface'
import { GlobalContext } from '../../global'

const defaultPage: TablePaginationConfig = {
  total: 0,
  current: 1,
  pageSize: 50
}

export default function ManageProblem() {
  const [global, setGlobal] = useContext(GlobalContext)
  useEffect(() => setGlobal({ path: ['Manage', 'Problem'] }), [])

  const [form] = Form.useForm<IProblem>()
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [problems, setProblems] = useState<IProblem[]>([])
  const [open, setOpen] = useState(false)
  const [problem, setProblem] = useState<IProblem>()
  const [pagination, setPagination] = useState(defaultPage)

  const { current, pageSize } = pagination
  useEffect(() => {
    if (global.user) {
      setLoading(true)
      getProblems({
        search,
        all: true,
        page: current,
        size: pageSize
      }).then(({ total, list }) => {
        setProblems(list)
        setPagination({ ...pagination, total })
      }).catch(e => {
        message.error(e)
      }).finally(() => {
        setLoading(false)
      })
    }
  }, [global.user, current, pageSize, search])

  return <>
    <Card
      title="Problems"
      extra={<>
        <Input.Search
          style={{ width: 200 }}
          placeholder="Title, content or tags"
          onSearch={value => {
            setSearch(value)
            setPagination({ ...pagination, current: 1 })
          }}
        />
        <Divider type="vertical" />
        <Button onClick={() => {
          setProblem(undefined)
          setOpen(true)
        }}>Create Problem</Button>
      </>}
    >
      <Modal
        style={{ minWidth: 1000 }}
        destroyOnClose={true}
        open={open}
        title={problem ? 'Edit' : 'Create'}
        confirmLoading={loading}
        onCancel={() => setOpen(false)}
        onOk={async () => {
          setLoading(true)
          try {
            const p = await form.validateFields()
            try {
              if (problem) { // Edit
                await putProblem(problem._id, p)
                const n = Object.assign({}, problem, p)
                setProblems(problems.map(o => o === problem ? n : o))
              } else { // Create
                setProblems([await postProblem(p), ...problems])
              }
              setOpen(false)
              message.success('success')
            } catch (e) {
              message.error(`${e}`)
            }
          } catch (e) {
            // ignore form validate error
            console.debug(e)
          } finally {
            setLoading(false)
          }
        }}
      >
        <ProblemForm problem={problem} form={form} />
      </Modal>
      <Table
        rowKey="_id"
        size="middle"
        loading={loading}
        dataSource={problems}
        pagination={pagination}
        onChange={setPagination}
        columns={[
          { title: 'ID', width: 250, dataIndex: '_id', render: t => (
            <a href={`/problem/${t}`} target="_blank">{t}</a>
          ) },
          { title: 'Title', dataIndex: 'title' },
          { title: 'Time', dataIndex: 'timeLimit', render: parseTime },
          { title: 'Memory', dataIndex: 'memoryLimit', render: parseMemory },
          { title: 'Tags', key: 'tags', render: (t, r) => <>
            {r.tags.map((tag, i) => <Tag key={i}>{tag}</Tag>)}
          </> },
          { title: 'Data', width: 250, dataIndex: 'data', render: t => (
            t ? <a href={`/api/file/${t}`} target="_blank">{t}</a> : 'null'
          ) },
          { title: 'Action', key: 'action', render: (_, r) => <>
            <a onClick={() => (setProblem(r), setOpen(true))}>Edit</a>
            <Divider type="vertical" />
            <Popconfirm title="Rejudge all submissions?" onConfirm={() => {
              rejudgeSubmission({ pid: r._id })
                .then(() => message.success('rejudge success'))
                .catch(message.error)
            }}>
              <a>Rejudge</a>
            </Popconfirm>
            <Divider type="vertical" />
            <Popconfirm title="Delete this problem?" onConfirm={() => {
              delProblem(r._id).then(() => {
                setProblems(problems.filter(p => p._id !== r._id))
              }).catch(message.error)
            }}>
              <a style={{ color: 'red' }}>Delete</a>
            </Popconfirm>
          </> }
        ]}
      />
    </Card>
  </>
}
