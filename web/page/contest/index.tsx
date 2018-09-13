import * as React from 'react'
import { withRouter } from 'react-router-dom'
import { Card, Table, Tag, message } from 'antd'

import * as model from '../../model'
import LoginTip from '../../component/login-tip'
import { IContest, ContestType, HistoryProps } from '../../util/interface'
import { updateState } from '../../util/state'

class Contests extends React.Component<HistoryProps> {
	state = {
		loading: true,
		contests: [] as IContest[],
		pagination: { current: 1, pageSize: 50, total: 0 }
	}
	componentWillMount() {
		updateState({ path: [ 'Contest' ] })
		if (!model.hasToken()) { return }
		this.handleChange(this.state.pagination)
	}
	handleChange = (pagination: any) => {
		const pager = { ...this.state.pagination }
		pager.current = pagination.current
		this.setState({ loading: true, pagination: pager })
		const { pageSize: size, current: page } = pager
		model.getContests({ page, size })
			.then(({ total, list: contests }) => {
				this.state.pagination.total = total
				this.setState({
					pagination: this.state.pagination,
					loading: false, contests
				})
			})
			.catch(err => {
				message.error(err)
				this.setState({ loading: false })
			})
	}
	render() {
		return <React.Fragment>
			<LoginTip />
			<Card title="Contests" className="contests">
				<Table
					rowKey="_id"
					onRow={({ _id }) => ({
						onClick: () => this.props.history.push(`/contests/${_id}`)
					})}
					loading={this.state.loading}
					dataSource={this.state.contests}
					pagination={this.state.pagination}
					onChange={this.handleChange}
					columns={[
						{ title: 'Title', width: 20, dataIndex: 'title' },
						{ title: 'Type', width: 10, dataIndex: 'type', render: t => {
							switch (t) {
								case ContestType.OI: return <Tag color="volcano">OI</Tag>
								case ContestType.ICPC: return <Tag color="cyan">ICPC</Tag>
								default: return <Tag>unknown</Tag>
							}
						}},
						{
							title: 'Start', width: 20, dataIndex: 'startAt',
							render: d => new Date(d).toLocaleString()
						},
						{
							title: 'End', width: 20, dataIndex: 'endAt',
							render: d => new Date(d).toLocaleString()
						},
						{ title: 'Status', width: 10, key: 'status', render: (t, r) => {
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
