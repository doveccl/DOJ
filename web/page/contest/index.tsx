import React from 'react'
import { withRouter } from 'react-router-dom'

import { message, Card, Table, Tag } from 'antd'

import { ContestType } from '../../../common/interface'
import { LoginTip } from '../../component/login-tip'
import { getContests, hasToken } from '../../model'
import { HistoryProps, IContest } from '../../util/interface'
import { updateState } from '../../util/state'

export const renderType = (t: ContestType) => {
	switch (t) {
		case ContestType.OI: return <Tag color="volcano">OI</Tag>
		case ContestType.ICPC: return <Tag color="cyan">ICPC</Tag>
		default: return <Tag>unknown</Tag>
	}
}

class Contests extends React.Component<HistoryProps> {
	public state = {
		loading: true,
		contests: [] as IContest[],
		pagination: { current: 1, pageSize: 50, total: 0 }
	}
	private handleChange = (pagination: any) => {
		const pager = { ...this.state.pagination }
		pager.current = pagination.current
		this.setState({ loading: true, pagination: pager })
		const { pageSize: size, current: page } = pager
		getContests({ page, size })
			.then(({ total, list: contests }) => {
				this.state.pagination.total = total
				this.setState({
					pagination: this.state.pagination,
					loading: false, contests
				})
			})
			.catch((err) => {
				message.error(err)
				this.setState({ loading: false })
			})
	}
	public componentDidMount() {
		updateState({ path: [ 'Contest' ] })
		if (hasToken()) { this.handleChange(this.state.pagination) }
	}
	public render() {
		return <React.Fragment>
			<LoginTip />
			<Card title="Contests" className="list">
				<Table
					rowKey="_id"
					size="middle"
					onRow={({ _id }) => ({
						onClick: () => this.props.history.push(`/contest/${_id}`)
					})}
					loading={this.state.loading}
					dataSource={this.state.contests}
					pagination={this.state.pagination}
					onChange={this.handleChange}
					columns={[
						{ title: 'Title', width: 200, dataIndex: 'title' },
						{ title: 'Type', width: 100, dataIndex: 'type', render: renderType},
						{
							title: 'Start At', width: 200, dataIndex: 'startAt',
							render: (d) => new Date(d).toLocaleString()
						},
						{
							title: 'End At', width: 200, dataIndex: 'endAt',
							render: (d) => new Date(d).toLocaleString()
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
		</React.Fragment>
	}
}

export default withRouter(({ history }) => <Contests history={history} />)
