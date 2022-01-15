import React, { useContext, useEffect, useState } from 'react'
import { message, Avatar, Card, Table, TablePaginationConfig } from 'antd'

import { glink } from '../../../common/function'
import { LoginTip } from '../../component/login-tip'
import { getUsers } from '../../model'
import { IUser } from '../../util/interface'
import { GlobalContext } from '../../global'

const defaultPage: TablePaginationConfig = {
	total: 0,
	current: 1,
	pageSize: 50
}

export default function Rank() {
	const [global, setGlobal] = useContext(GlobalContext)
	useEffect(() => setGlobal({ path: ['Rank'] }), [])

	const [loading, setLoading] = useState(true)
	const [users, setUsers] = useState([] as IUser[])
	const [pagination, setPagination] = useState(defaultPage)

	const { current, pageSize } = pagination
	useEffect(() => {
		global.user && getUsers({
			rank: true,
			page: pagination.current,
			size: pagination.pageSize
		}).then(({ total, list }) => {
			setUsers(list)
			setPagination({ ...pagination, total })
		}).catch(e => {
			message.error(e)
		}).finally(() => {
			setLoading(false)
		})
	}, [global.user, current, pageSize])

	return <>
		<LoginTip />
		<Card title="Rank List">
			<Table
				rowKey="_id"
				size="middle"
				loading={loading}
				dataSource={users}
				pagination={pagination}
				onChange={setPagination}
				columns={[
					{ title: 'Rank', width: 100, key: 'rank', render: (t, r, i) => (
						i + pageSize * (current - 1) + 1
					) },
					{ title: 'User', width: 200, key: 'user', render: (t, r) => <span>
						<Avatar size="small" src={glink(r.mail)} />
						<div className="hdivider" />
						<span>{r.name}</span>
					</span> },
					{ title: 'Introduction', width: 300, dataIndex: 'introduction' },
					{ title: 'AC Ratio', width: 200, key: 'ratio', render: (t, r) => (
						`${r.solve} / ${r.submit} (${(100 * r.solve / r.submit || 0).toFixed(1)}%)`
					) }
				]}
			/>
		</Card>
	</>
}
