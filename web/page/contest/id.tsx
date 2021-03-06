import React from 'react'
import { withRouter, Link } from 'react-router-dom'

import { message, Card, Col, Progress, Row, Table } from 'antd'

import { parseCount } from '../../../common/function'
import { Discuss } from '../../component/discuss'
import { LoginTip } from '../../component/login-tip'
import { MarkDown } from '../../component/markdown'
import { Scoreboard } from '../../component/scoreboard'
import { getContest, getProblems, hasToken } from '../../model'
import { HistoryProps, IContest, IProblem, MatchProps } from '../../util/interface'
import { addListener, globalState, removeListener, updateState } from '../../util/state'
import { renderType } from './index'

class Contest extends React.Component<HistoryProps & MatchProps> {
	private timer = undefined as any
	public state = {
		process: 0,
		status: '',
		global: globalState,
		contest: {} as IContest,
		problems: [] as IProblem[],
		tabKey: this.props.history.location.hash.replace('#', '') || 'description'
	}
	private refreshProcess = (c?: IContest) => {
		const { startAt, endAt } = c || this.state.contest
		const st = new Date(startAt)
		const et = new Date(endAt)
		const now = new Date()
		const diff = +now - +st
		const status = parseCount(diff)
		let process = 100 * diff / (+et - +st)
		if (process < 0) { process = 0 }
		if (process > 100) { process = 100 }
		this.setState({ process, status })
	}
	public componentDidMount() {
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
				<Row justify="space-between">
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
				{this.state.tabKey === 'description' && <MarkDown
					children={description}
					allowDangerousHtml={true}
				/>}
				{this.state.tabKey === 'problems' && <Table
					rowKey="_id"
					size="middle"
					pagination={false}
					dataSource={this.state.problems}
					columns={[
						{
							width: 100, sorter: (a, b) => a.contest.key.localeCompare(b.contest.key),
							title: 'Index', dataIndex: ['contest', 'key'], defaultSortOrder: 'ascend'
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
				{this.state.tabKey === 'scoreboard' && _id && <Scoreboard id={_id} />}
			</Card>
		</React.Fragment>
	}
}

export default withRouter(({ history, match }) => <Contest history={history} match={match} />)
