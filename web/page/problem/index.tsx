import React, { useContext, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { message, Card, Input, Progress, Table, Tag, TablePaginationConfig } from 'antd'
import { CheckOutlined } from '@ant-design/icons'

import { LoginTip } from '../../component/login-tip'
import { getProblems } from '../../model'
import { IProblem } from '../../interface'
import { GlobalContext } from '../../global'

const defaultPage: TablePaginationConfig = {
	total: 0,
	current: 1,
	pageSize: 50
}

export default function Problems() {
	const navigate = useNavigate()
	const [global, setGlobal] = useContext(GlobalContext)
	useEffect(() => setGlobal({ path: ['Problem'] }), [])

	const [search, setSearch] = useState('')
	const [loading, setLoading] = useState(true)
	const [problems, setProblems] = useState([] as IProblem[])
	const [pagination, setPagination] = useState(defaultPage)

	const { current, pageSize } = pagination
	useEffect(() => {
		if (global.user) {
			setLoading(true)
			getProblems({
				search,
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
		<LoginTip />
		<Card
			title="Problems"
			className="list"
			extra={<Input.Search
				placeholder="Title, content or tags"
				onSearch={value => {
					setSearch(value)
					setPagination({ ...pagination, current: 1 })
				}}
			/>}
		>
			<Table
				rowKey="_id"
				size="middle"
				onRow={({ _id }) => ({
					onClick: () => navigate(`/problem/${_id}`)
				})}
				loading={loading}
				dataSource={problems}
				pagination={pagination}
				onChange={setPagination}
				columns={[
					{ title: 'AC', width: 20, align: 'center', key: 'solved', render: (_, r) =>
						r.solved ? <CheckOutlined style={{ color: '#52c41a' }} /> : null
					},
					{ title: 'Title', width: 200, dataIndex: 'title' },
					{ title: 'Tags', width: 100, key: 'tags', render: (_, r) => <>
						{r.tags.map((tag, i) => <Tag key={i}>{tag}</Tag>)}
					</> },
					{ title: 'Ratio', width: 100, key: 'ratio', render: (_, r) =>
						<Progress percent={Math.floor(100 * r.solve / r.submit)} status="active" />
					}
				]}
			/>
		</Card>
	</>
}
