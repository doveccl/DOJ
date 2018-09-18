import * as React from 'react'
import { withRouter, Link } from 'react-router-dom'

import { message, Card, Col, Progress, Row, Table } from 'antd'

import Discuss from '../../component/discuss'
import LoginTip from '../../component/login-tip'
import Markdown from '../../component/markdown'
import { getContest, getProblems, hasToken } from '../../model'
import { parseTime } from '../../util/function'
import { HistoryProps, IContest, IProblem, MatchProps } from '../../util/interface'
import { addListener, globalState, removeListener, updateState } from '../../util/state'
import { renderType } from './index'

class Contest extends React.Component<HistoryProps & MatchProps> {
	private timer = undefined as any
	public state = {
		process: 0,
		status: '',
		tabKey: 'description',
		contest: {} as IContest,
		problems: [] as IProblem[],
		global: globalState
	}
	private refreshProcess = (c?: IContest) => {
		const { startAt, endAt } = c || this.state.contest
		const st = new Date(startAt)
		const et = new Date(endAt)
		const now = new Date()
		const diff = +now - +st
		const status = parseTime(diff)
		let process = 100 * diff / (+et - +st)
		if (process < 0) { process = 0 }
		if (process > 100) { process = 100 }
		this.setState({ process, status })
	}
	public componentWillMount() {
		const { params } = this.props.match
		updateState({ path: [
			{ url: '/contest', text: 'Contest' }, params.id
		] })
		addListener('contest', (global) => {
			this.setState({ global })
		})
		if (hasToken()) {
			getContest(params.id)
				.then((contest) => {
					this.setState({ contest })
					this.refreshProcess(contest)
				})
				.catch(message.error)
			getProblems({ cid: params.id })
				.then(({ list: problems }) => this.setState({ problems }))
				.catch(console.warn)
			this.timer = setInterval(this.refreshProcess, 1000)
		}
	}
	public componentWillReceiveProps(nextProps: HistoryProps & MatchProps) {
		const { hash } = nextProps.history.location
		if (hash) { this.setState({ tabKey: hash.replace('#', '') }) }
	}
	public componentWillUnmount() {
		clearInterval(this.timer)
		removeListener('contest')
	}
	public render() {
		const { process, status, contest, global } = this.state
		const { _id, title, description } = contest
		const { startAt, endAt, type } = contest

		return <React.Fragment>
			<LoginTip />
			<Card
				loading={!_id}
				title={title || 'Contest'}
				extra={renderType(type)}
			>
				<Progress percent={this.state.process} showInfo={false} />
				<Row type="flex" justify="space-between">
					<Col>{new Date(startAt).toLocaleString()}</Col>
					<Col>{process < 100 ? status : 'Ended'}</Col>
					<Col>{new Date(endAt).toLocaleString()}</Col>
				</Row>
			</Card>
			<div className="divider" />
			<Card
				tabList={[
					{ key: 'description', tab: 'Description' },
					{ key: 'problems', tab: 'Problems' },
					{ key: 'discuss', tab: 'Discuss' },
					{ key: 'scoreboard', tab: 'Scoreboard' }
				]}
				activeTabKey={this.state.tabKey}
				onTabChange={(tabKey) => {
					this.setState({ tabKey })
					this.props.history.push(`#${tabKey}`)
				}}
			>
				{this.state.tabKey === 'description' && <Markdown
					escapeHtml={false}
					source={description}
				/>}
				{this.state.tabKey === 'problems' && <Table
					rowKey="_id"
					size="middle"
					pagination={false}
					dataSource={this.state.problems}
					columns={[
						{
							width: 100, sorter: (a, b) => a.contest.key.localeCompare(b.contest.key),
							title: 'Index', dataIndex: 'contest.key', defaultSortOrder: 'ascend'
						},
						{ width: 200, title: 'Title', dataIndex: 'title', render: (t, r) => <Link
							children={t} to={`/problem/${r._id}`}
						/> },
						{ width: 200, title: 'Action', dataIndex: '_id', render: (id) => <a
							children="Open in new tab" href={`/problem/${id}`} target="_blank"
						/> }
					]}
				/>}
				{this.state.tabKey === 'discuss' && global.user && _id && <Discuss topic={_id} />}
			</Card>
		</React.Fragment>
	}
}

export default withRouter(({ history, match }) => <Contest history={history} match={match} />)
