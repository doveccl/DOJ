import React, { useContext, useEffect, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { message, Button, Card, Col, Input, Row, Table, Tag, TablePaginationConfig } from 'antd'
import { UserOutlined, FileTextOutlined } from '@ant-design/icons'

import { parseMemory, parseTime } from '../../../common/function'
import { IResult, Status } from '../../../common/interface'
import { LoginTip } from '../../component/login-tip'
import { getSubmissions } from '../../model'
import { ISubmission } from '../../util/interface'
import { GlobalContext } from '../../global'

export const renderStatus = (r: IResult) => {
	switch (r.status) {
		case Status.WAIT: return <Tag color="blue">Pending</Tag>
		case Status.AC: return <Tag color="green">Accepted</Tag>
		case Status.WA: return <Tag color="red">Wrong Answer</Tag>
		case Status.TLE: return <Tag color="gold">Time Limit Exceed</Tag>
		case Status.MLE: return <Tag color="volcano">Memory Limit Exceed</Tag>
		case Status.RE: return <Tag color="magenta">Runtime Error</Tag>
		case Status.CE: return <Tag color="purple">Compile Error</Tag>
		case Status.SE: return <Tag color="cyan">System Error</Tag>
		case Status.FREEZE: return <Tag color="geekblue">Frozen</Tag>
		default: return <Tag>unknown</Tag>
	}
}

const defaultPage: TablePaginationConfig = {
	total: 0,
	current: 1,
	pageSize: 50
}

export default function Submissions() {
	const navigate = useNavigate()
	const [search, setSearch] = useSearchParams()
	const [global, setGlobal] = useContext(GlobalContext)
	useEffect(() => setGlobal({ path: ['Submission'] }), [])

	const [flag, update] = useState({})
	const [loading, setLoading] = useState(true)
	const [submissions, setSubmissions] = useState([] as ISubmission[])
	const [pagination, setPagination] = useState(defaultPage)
	const [pid, uname] = [search.get('pid'), search.get('uname')]
	const [tmpPid, setTmpPid] = useState(pid)
	const [tmpUname, setTmpUname] = useState(uname)

	const { current, pageSize, total } = pagination
	useEffect(() => {
		if (global.user) {
			setLoading(true)
			getSubmissions({
				pid,
				uname,
				page: current,
				size: pageSize
			}).then(({ total, list }) => {
				setSubmissions(list)
				setPagination({ ...pagination, total })
			}).catch(e => {
				message.error(e)
			}).finally(() => {
				setLoading(false)
			})
		}
	}, [global.user, current, pageSize, pid, uname, flag])

	return <>
		<LoginTip />
		<Card title="Submissions">
			<Row gutter={8}>
				<Col span={10}>
					<Input
						allowClear
						value={tmpUname}
						placeholder="Username"
						onChange={e => setTmpUname(e.target.value)}
						prefix={<UserOutlined style={{ color: '#CCC' }} />}
					/>
				</Col>
				<Col span={10}>
					<Input
						allowClear
						value={tmpPid}
						placeholder="Problem ID"
						onChange={e => setTmpPid(e.target.value)}
						prefix={<FileTextOutlined style={{ color: '#CCC' }} />}
					/>
				</Col>
				<Col span={4}>
					<Button block type="primary" onClick={() => {
						if (pid !== tmpPid || uname !== tmpUname) {
							const search: Record<string, string> = {}
							if (tmpPid) search.pid = tmpPid
							if (tmpUname) search.uname = tmpUname
							setPagination({ ...pagination, current: 1 })
							setSearch(search)
						} else {
							update({}) // refresh only
						}
					}}>Search / Refresh</Button>
				</Col>
			</Row>
			<div className="divider" />
			<Table
				rowKey="_id"
				size="middle"
				loading={loading}
				dataSource={submissions}
				pagination={pagination}
				onChange={setPagination}
				columns={[
					{ title: 'Index', dataIndex: '_id', render: (t, r, i) => <Link
						children={total - pageSize * (current - 1) - i} to={`/submission/${t}`}
					/> },
					{ title: 'User', dataIndex: 'uname' },
					{ title: 'Problem', dataIndex: 'ptitle', render: (t, r) => <Link
						children={t} to={`/problem/${r.pid}`}
					/> },
					{ title: 'Status', dataIndex: 'result', render: renderStatus, onCell: (r) => ({
						onClick: () => navigate(`/submission/${r._id}`)
					}) },
					{ title: 'Time', align: 'center', dataIndex: ['result', 'time'], render: parseTime },
					{ title: 'Memory', align: 'center', dataIndex: ['result', 'memory'], render: parseMemory },
					{ title: 'Language', align: 'center', dataIndex: 'language', render: l => (
						global.languages?.[l].name ?? 'unknown'
					) },
					{ title: 'Submit At', align: 'center', dataIndex: 'createdAt', render: t => (
						new Date(t).toLocaleString()
					) }
				]}
			/>
		</Card>
	</>
}
