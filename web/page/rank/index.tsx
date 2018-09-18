import * as React from 'react'

import { message, Avatar, Card, Table } from 'antd'

import LoginTip from '../../component/login-tip'
import { getUsers, hasToken } from '../../model'
import { glink } from '../../util/function'
import { IUser } from '../../util/interface'
import { updateState } from '../../util/state'

export default class extends React.Component {
	public state = {
		loading: true,
		users: [] as IUser[],
		pagination: { current: 1, pageSize: 50, total: 0 }
	}
	private handleChange = (pagination: any) => {
		const pager = { ...this.state.pagination }
		pager.current = pagination.current
		this.setState({ loading: true, pagination: pager })
		const { pageSize: size, current: page } = pager
		getUsers({ page, size, rank: 1 })
			.then(({ total, list: users }) => {
				this.state.pagination.total = total
				this.setState({
					pagination: this.state.pagination,
					loading: false, users
				})
			})
			.catch((err) => {
				message.error(err)
				this.setState({ loading: false })
			})
	}
	public componentWillMount() {
		updateState({ path: [ 'Rank' ] })
		if (hasToken()) { this.handleChange(this.state.pagination) }
	}
	public render() {
		const { current, pageSize } = this.state.pagination
		return <React.Fragment>
			<LoginTip />
			<Card title="Rank List">
				<Table
					rowKey="_id"
					size="middle"
					loading={this.state.loading}
					dataSource={this.state.users}
					pagination={this.state.pagination}
					onChange={this.handleChange}
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
		</React.Fragment>
	}
}
