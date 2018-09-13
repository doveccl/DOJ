import * as React from 'react'
import { withRouter, Link } from 'react-router-dom'
import { Card, Row, Col, Progress, Table, message } from 'antd'

import * as model from '../../model'
import * as state from '../../util/state'
import Markdown from '../../component/markdown'
import LoginTip from '../../component/login-tip'
import { renderType } from './index'
import { parseTime } from '../../util/function'
import { IProblem, IContest, HistoryProps, MatchProps } from '../../util/interface'

class Contest extends React.Component<HistoryProps & MatchProps> {
	private timer = undefined as any
	state = {
		process: 0,
		tabKey: 'description',
		contest: {} as IContest,
		problems: [] as IProblem[],
		global: state.globalState
	}
	componentWillMount() {
		const { params } = this.props.match
		state.updateState({ path: [
			{ url: '/contest', text: 'Contest' }, params.id
		] })
		state.addListener('contest', global => this.setState({ global }))
		this.componentWillReceiveProps(this.props)
		if (!model.hasToken()) { return }
		model.getContest(params.id)
			.then(contest => {
				this.setState({ contest })
				this.refreshProcess(contest)
			})
			.catch(err => message.error(err))
		model.getProblems({ cid: params.id })
			.then(({ list: problems }) => this.setState({ problems }))
			.catch(err => console.warn(err))
		this.timer = setInterval(this.refreshProcess, 1000)
	}
	componentWillUnmount() {
		clearInterval(this.timer)
		state.removeListener('contest')
	}
	componentWillReceiveProps(nextProps: HistoryProps & MatchProps) {
		const { hash } = nextProps.history.location
		if (hash) { this.setState({ tabKey: hash.replace('#', '') }) }
	}
	refreshProcess = (c?: IContest) => {
		const { startAt, endAt } = c || this.state.contest
		const st = new Date(startAt), et = new Date(endAt)
		const now = new Date(), diff = +now - +st
		let process = 100 * diff / (+et - +st)
		if (process < 0) { process = 0 }
		if (process > 100) { process =100 }
		this.setState({ process })
	}
	render() {
		const { startAt, endAt, type } = this.state.contest
		const { _id, title, description } = this.state.contest
		const st = new Date(startAt), et = new Date(endAt)
		const now = new Date(), diff = parseTime(+now - +st)
		return <React.Fragment>
			<LoginTip />
			<Card
				loading={!_id}
				title={title || 'Contest'}
				extra={renderType(type)}
			>
				<Progress percent={this.state.process} showInfo={false} />
				<Row type="flex" justify="space-between">
					<Col>{st.toLocaleString()}</Col>
					<Col>{now < et ? diff : 'Ended'}</Col>
					<Col>{et.toLocaleString()}</Col>
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
				onTabChange={tabKey => {
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
						{ width: 200, title: 'Action', dataIndex: '_id', render: _id => <a
							children="Open in new tab" href={`/problem/${_id}`} target="_blank"
						/> }
					]}
				/>}
      </Card>
		</React.Fragment>
	}
}

export default withRouter(({ history, match }) => <Contest history={history} match={match} />)
