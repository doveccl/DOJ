import moment from 'moment'
import React, { useContext, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { message, Card, Table, Tag, TablePaginationConfig } from 'antd'
import { ContestType } from '../../../common/interface'
import { LoginTip } from '../../component/login-tip'
import { getContests } from '../../model'
import { IContest } from '../../interface'
import { GlobalContext } from '../../global'

export const renderType = (t: ContestType) => {
	switch (t) {
		case ContestType.OI: return <Tag color="volcano">OI</Tag>
		case ContestType.ICPC: return <Tag color="cyan">ICPC</Tag>
		default: return <Tag>unknown</Tag>
	}
}

const defaultPage: TablePaginationConfig = {
	total: 0,
	current: 1,
	pageSize: 50
}

export default function Contests() {
	const navigate = useNavigate()
	const [global, setGlobal] = useContext(GlobalContext)
	useEffect(() => setGlobal({ path: ['Contest'] }), [])

	const [loading, setLoading] = useState(true)
	const [contests, setContests] = useState([] as IContest[])
	const [pagination, setPagination] = useState(defaultPage)

	const { current, pageSize } = pagination
	useEffect(() => {
		global.user && getContests({
			rank: true,
			page: pagination.current,
			size: pagination.pageSize
		}).then(({ total, list }) => {
			setContests(list)
			setPagination({ ...pagination, total })
		}).catch(e => {
			message.error(e)
		}).finally(() => {
			setLoading(false)
		})
	}, [global.user, current, pageSize])

	return <>
		<LoginTip />
		<Card title="Contests" className="list">
			<Table
				rowKey="_id"
				size="middle"
				onRow={({ _id }) => ({
					onClick: () => navigate(`/contest/${_id}`)
				})}
				loading={loading}
				dataSource={contests}
				pagination={pagination}
				onChange={setPagination}
				columns={[
					{ title: 'Title', width: 200, dataIndex: 'title' },
					{ title: 'Type', width: 100, dataIndex: 'type', render: renderType},
					{
						title: 'Start At', width: 200, dataIndex: 'startAt',
						render: d => moment(d).format()
					},
					{
						title: 'End At', width: 200, dataIndex: 'endAt',
						render: d => moment(d).format()
					},
					{ title: 'Status', width: 100, key: 'status', render: (t, r) => {
						const now = new Date()
						switch (true) {
							case now < new Date(r.startAt):
								return <Tag color="blue">Pending</Tag>
							case now < new Date(r.endAt):
								return <Tag color="green">Running</Tag>
							default: return <Tag color="red">Ended</Tag>
						}
					}}
				]}
			/>
		</Card>
	</>
}
