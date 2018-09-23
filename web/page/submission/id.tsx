import * as React from 'react'
import * as io from 'socket.io-client'

import { message, Card, Checkbox, Tag, Timeline } from 'antd'
import { withRouter, Link } from 'react-router-dom'

import { parseMemory, parseTime } from '../../../common/function'
import { Group, Status } from '../../../common/interface'
import { diffGroup } from '../../../common/user'
import { Code } from '../../component/code'
import { LoginTip } from '../../component/login-tip'
import { getSubmission, getToken, hasToken, putSubmission } from '../../model'
import { HistoryProps, ISubmission, MatchProps } from '../../util/interface'
import { addListener, globalState, removeListener, updateState } from '../../util/state'
import { renderStatus } from './index'

class Submission extends React.Component<HistoryProps & MatchProps> {
	public state = {
		global: globalState,
		pending: false,
		submission: {} as ISubmission
	}
	private handleSubmission = (s: ISubmission) => {
		this.setState({
			submission: s,
			pending: s.result.status === Status.WAIT ?
				'Pending ...' : false
		})
		if (s.result.status === Status.WAIT) {
			const socket = io('/client')
			socket.on('result', (data: Partial<ISubmission>) => {
				socket.close()
				this.state.submission.result = data.result
				this.state.submission.cases = data.cases
				this.setState({
					pending: false,
					submission: this.state.submission
				})
			})
			socket.on('connect', () => {
				socket.emit('register', { id: s._id, token: getToken() }, (ok: boolean) => {
					if (!ok) { socket.close() }
				})
			})
		}
	}
	public componentWillMount() {
		const { params } = this.props.match
		updateState({ path: [
			{ url: '/submission', text: 'Submission' }, params.id
		] })
		addListener('submission', (global) => {
			this.setState({ global })
		})
		if (hasToken()) {
			getSubmission(params.id)
				.then(this.handleSubmission)
				.catch((err) => message.error(err))
		}
	}
	public componentWillUnmount() {
		removeListener('submission')
	}
	public render() {
		const { global, submission } = this.state
		const { _id, uname, pid, ptitle } = submission
		const { result, cases, code, language } = submission
		const { open, createdAt } = submission
		const show = result && result.status !== Status.WAIT
		const lan = global.languages[language]

		return <React.Fragment>
			<LoginTip />
			<Card
				loading={!_id}
				title={`Submission by ${uname || 'user'}`}
				extra={<Link to={`/problem/${pid}`}>{ptitle}</Link>}
			>
				<Timeline pending={this.state.pending}>
					<Timeline.Item color="green">
						[{new Date(createdAt).toLocaleString()}] {uname} submitted the code
					</Timeline.Item>
					{cases && cases.length > 0 && <Timeline.Item>
						{cases.map((c, i) => <p key={i}>
							{renderStatus(c)}
							<Tag>{parseTime(c.time)}</Tag>
							<Tag>{parseMemory(c.memory)}</Tag>
							<span>{c.extra}</span>
						</p>)}
					</Timeline.Item>}
					{show && <Timeline.Item color={result.status === Status.AC ? 'green' : 'red'}>
						<pre>
							{renderStatus(result)}
							<Tag>{parseTime(result.time)}</Tag>
							<Tag>{parseMemory(result.memory)}</Tag>
						</pre>
						{result.extra && <pre>{result.extra}</pre>}
					</Timeline.Item>}
				</Timeline>
			</Card>
			{code && <React.Fragment>
				<div className="divider" />
				<Card
					title="Code"
					extra={<Checkbox
						checked={open}
						children="Public"
						disabled={
							!diffGroup(global.user, Group.admin) &&
							global.user._id !== submission.uid
						}
						onChange={(e) => {
							const s = { ...submission }
							s.open = e.target.checked
							putSubmission(_id, { open: s.open })
								.then(() => this.setState({ submission: s }))
								.catch(message.error)
						}}
					/>}
				>
					<Code
						value={code}
						static={true}
						language={lan && lan.suffix}
					/>
				</Card>
			</React.Fragment>}
		</React.Fragment>
	}
}

export default withRouter(({ history, match }) => <Submission history={history} match={match} />)
