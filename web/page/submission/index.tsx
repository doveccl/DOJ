import * as React from 'react'
import { withRouter, Link } from 'react-router-dom'

import { message, Button, Card, Col, Icon, Input, Row, Table, Tag } from 'antd'

import LoginTip from '../../component/login-tip'
import { getSubmissions, hasToken } from '../../model'
import { renderMemory, renderTime } from '../../util/function'
import { HistoryProps, IResult, ISubmission, Status } from '../../util/interface'
import { addListener, globalState, removeListener, updateState } from '../../util/state'

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
		default: return <Tag>unknown</Tag>
	}
}

class Submissions extends React.Component<HistoryProps> {
	public state = {
		loading: true,
		uname: '',
		pid: '',
		cid: '',
		global: globalState,
		submissions: [] as ISubmission[],
		pagination: { current: 1, pageSize: 50, total: 0 }
	}
	private handleChange = (pagination: any) => {
		const { uname, pid, cid } = this.state
		const pager = { ...this.state.pagination }
		pager.current = pagination.current
		this.setState({ loading: true, pagination: pager })
		const { pageSize: size, current: page } = pager
		getSubmissions({ page, size, uname, pid, cid })
			.then(({ total, list: submissions }) => {
				this.state.pagination.total = total
				this.setState({
					pagination: this.state.pagination,
					loading: false, submissions
				})
			})
			.catch((err) => {
				message.error(err)
				this.setState({ loading: false })
			})
	}
	private handleSearch = () => {
		const pagination = { ...this.state.pagination }
		pagination.current = 1
		this.handleChange(pagination)
	}
	public componentWillMount() {
		updateState({ path: [ 'Submission' ] })
		addListener('submissions', (global) => this.setState({ global }))
		if (hasToken()) { this.handleChange(this.state.pagination) }
	}
	public componentWillUnmount() {
		removeListener('submissions')
	}
	public render() {
		const { languages } = this.state.global
		const { pageSize, current, total } = this.state.pagination
		return <React.Fragment>
			<LoginTip />
			<Card title="Submissions">
				<Row gutter={16}>
					<Col span={7}>
						<Input
							placeholder="Username"
							onChange={(e) => this.setState({ uname: e.target.value })}
							prefix={<Icon type="user" style={{ color: 'rgba(0,0,0,.25)' }} />}
						/>
					</Col>
					<Col span={7}>
						<Input
							placeholder="Problem ID"
							onChange={(e) => this.setState({ pid: e.target.value })}
							prefix={<Icon type="file-text" style={{ color: 'rgba(0,0,0,.25)' }} />}
						/>
					</Col>
					<Col span={7}>
						<Input
							placeholder="Contest ID"
							onChange={(e) => this.setState({ cid: e.target.value })}
							prefix={<Icon type="code" style={{ color: 'rgba(0,0,0,.25)' }} />}
						/>
					</Col>
					<Col span={3}>
						<Button block type="primary" onClick={this.handleSearch}>Search</Button>
					</Col>
				</Row>
				<div className="divider" />
				<Table
					rowKey="_id"
					size="middle"
					loading={this.state.loading}
					dataSource={this.state.submissions}
					pagination={this.state.pagination}
					onChange={this.handleChange}
					columns={[
						{ title: 'Index', dataIndex: '_id', render: (t, r, i) => <Link
							children={total - pageSize * (current - 1) - i} to={`/submission/${t}`}
						/> },
						{ title: 'User', dataIndex: 'uname' },
						{ title: 'Problem', dataIndex: 'ptitle', render: (t, r) => <Link
							children={t} to={`/problem/${r.pid}`}
						/> },
						{ title: 'Status', dataIndex: 'result', render: renderStatus, onCell: (r) => ({
							onClick: () => this.props.history.push(`/submission/${r._id}`)
						}) },
						{ title: 'Time', align: 'center', dataIndex: 'result.time', render: renderTime },
						{ title: 'Memory', align: 'center', dataIndex: 'result.memory', render: renderMemory },
						{ title: 'Language', align: 'center', dataIndex: 'language', render: (l) => (
							languages[l] ? languages[l].name : 'unknown'
						) },
						{ title: 'Submit At', align: 'center', dataIndex: 'createdAt', render: (t) => (
							new Date(t).toLocaleString()
						) }
					]}
				/>
			</Card>
		</React.Fragment>
	}
}

export default withRouter(({ history }) => <Submissions history={history} />)
