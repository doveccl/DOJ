import moment from 'moment'
import React, { useContext, useEffect, useState } from 'react'
import { message, Button, Card, Col, Divider, Input, Modal, Popconfirm, Row, Table, TablePaginationConfig, Form } from 'antd'
import { FileTextOutlined, BarsOutlined } from '@ant-design/icons'
import { ContestForm } from '../../component/form/contest'
import { delContest, getContests, getProblem, getProblems, hasToken, postContest, putContest, putProblem } from '../../model'
import { IContest, IProblem } from '../../interface'
import { GlobalContext } from '../../global'
import { renderType } from '../contest'

const defaultPage: TablePaginationConfig = {
	total: 0,
	current: 1,
	pageSize: 50,
	hideOnSinglePage: true
}

function ContestProblems({ cid = '' }) {
	const [pid, setPid] = useState('')
	const [pkey, setPkey] = useState('')
	const [loading, setLoading] = useState(true)
	const [problems, setProblems] = useState([] as IProblem[])

	useEffect(() => {
		setLoading(true)
		getProblems({
			cid,
			all: true
		}).then(({ list }) => {
			setProblems(list)
		}).catch(e => {
			message.error(e)
		}).finally(() => {
			setLoading(false)
		})
	}, [])

	return <>
		<Row gutter={16}>
			<Col span={10}>
				<Input
					placeholder="Problem ID"
					onChange={e => setPid(e.target.value)}
					prefix={<FileTextOutlined style={{ color: '#CCC' }} />}
				/>
			</Col>
			<Col span={10}>
				<Input
					placeholder="Problem Key (A, B, C, D ...)"
					onChange={e => setPkey(e.target.value)}
					prefix={<BarsOutlined style={{ color: '#CCC' }} />}
				/>
			</Col>
			<Col span={4}>
				<Button block disabled={!pid || !pkey} type="primary" onClick={async () => {
					try {
						await putProblem(pid, { contest: { id: cid, key: pkey } })
						const others = problems.filter(p => p._id !== pid)
						setProblems([await getProblem(pid), ...others])
					} catch (e) {
						message.error(e)
					}
				}}>Add</Button>
			</Col>
		</Row>
		<div className="divider" />
		<Table
			rowKey="_id"
			size="middle"
			loading={loading}
			dataSource={problems}
			pagination={false}
			columns={[
				{ title: 'Problem ID', width: 250, dataIndex: '_id', render: t => (
					<a href={`/problem/${t}`} target="_blank">{t}</a>
				) },
				{ title: 'Problem Key', dataIndex: ['contest', 'key'] },
				{ title: 'Problem Title', dataIndex: 'title' },
				{ title: 'Action', key: 'action', render: (_, r) => <Popconfirm
					title="Delete this problem?" onConfirm={() => {
						putProblem(r._id, { contest: null })
							.then(() => setProblems(problems.filter(p => p._id !== r._id)))
							.catch(message.error)
					}}
				>
					<a style={{ color: 'red' }}>Delete</a>
				</Popconfirm> }
			]}
		/>
	</>
}

export default function ManageContest() {
	const [global, setGlobal] = useContext(GlobalContext)
	useEffect(() => setGlobal({ path: ['Manage', 'Contest'] }), [])

	const [form] = Form.useForm<IContest>()
	const [loading, setLoading] = useState(true)
	const [contests, setContests] = useState([] as IContest[])
	const [open, setOpen] = useState(false)
	const [contest, setContest] = useState(null as IContest)
	const [pagination, setPagination] = useState(defaultPage)

	const { current, pageSize } = pagination
	useEffect(() => {
		if (global.user) {
			setLoading(true)
			getContests({
				page: current,
				size: pageSize
			}).then(({ total, list }) => {
				setContests(list)
				setPagination({ ...pagination, total })
			}).catch(e => {
				message.error(e)
			}).finally(() => {
				setLoading(false)
			})
		}
	}, [global.user, current, pageSize])

	return <>
		<Card
			title="Contests"
			extra={<Button onClick={() => {
				setContest(null)
				setOpen(true)
			}}>Create Contest</Button>}
		>
			<Modal
				style={{ minWidth: 1000 }}
				destroyOnClose={true}
				visible={open}
				title={contest ? 'Edit' : 'Create'}
				confirmLoading={loading}
				onCancel={() => setOpen(false)}
				onOk={async () => {
					setLoading(true)
					try {
						const c = await form.validateFields()
						try {
							if (contest) { // Edit
								await putContest(contest._id, c)
								const n = Object.assign({}, contest, c)
								setContests(contests.map(o => o === contest ? n : o))
							} else { // Create
								setContests([await postContest(c), ...contests])
							}
							setOpen(false)
							message.success('success')
						} catch (e) {
							message.error(e)
						}
					} catch (e) {
						// ignore form validate error
						console.debug(e)
					} finally {
						setLoading(false)
					}
				}}
			>
				<ContestForm contest={contest} form={form} />
			</Modal>
			<Table
				rowKey="_id"
				size="middle"
				expandedRowRender={t => <ContestProblems cid={t._id} />}
				loading={loading}
				dataSource={contests}
				pagination={pagination}
				onChange={setPagination}
				columns={[
					{ title: 'ID', width: 250, dataIndex: '_id', render: t => (
						<a href={`/contest/${t}`} target="_blank">{t}</a>
					) },
					{ title: 'Title', dataIndex: 'title' },
					{ title: 'Type', dataIndex: 'type', render: renderType },
					{ title: 'Start At', dataIndex: 'startAt', render: t => moment(t).format('llll') },
					{ title: 'End At', dataIndex: 'endAt', render: t => moment(t).format('llll') },
					{ title: 'Action', key: 'action', render: (_, r) => <>
						<a onClick={() => (setContest(r), setOpen(true))}>Edit</a>
						<Divider type="vertical" />
						<Popconfirm title="Delete this contest?" onConfirm={() => {
							delContest(r._id).then(() => {
								setContests(contests.filter(c => c._id !== r._id))
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
