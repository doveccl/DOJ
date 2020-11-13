import React from 'react'
import { Manager, Socket } from 'socket.io-client'

import { message, Button, Card, Checkbox, Divider, Tag, Timeline } from 'antd'
import { withRouter, Link } from 'react-router-dom'

import { parseMemory, parseTime } from '../../../common/function'
import { Group, IResult, Status } from '../../../common/interface'
import { diffGroup } from '../../../common/user'
import { Code } from '../../component/code'
import { LoginTip } from '../../component/login-tip'
import { getSubmission, getToken, hasToken, putSubmission, rejudgeSubmission } from '../../model'
import { HistoryProps, ISubmission, MatchProps } from '../../util/interface'
import { addListener, globalState, removeListener, updateState } from '../../util/state'
import { renderStatus } from './index'

const color = (r?: IResult) => r.status === Status.AC ? 'green' : 'red'

class Submission extends React.Component<HistoryProps & MatchProps> {
	private socket: Socket
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
			const manager = new Manager()
			this.socket = manager.socket('/client')
			this.socket.on('result', (data: Partial<ISubmission>) => {
				this.socket.close()
				this.state.submission.result = data.result
				this.state.submission.cases = data.cases
				this.setState({
					pending: false,
					submission: this.state.submission
				})
			})
			this.socket.on('connect', () => {
				this.socket.emit('register', { 
					d: s._id,
					token: getToken()
				}, (ok: boolean) => {
					ok || this.socket.close()
				})
			})
		}
	}
	private rejudge = () => {
		const { submission } = this.state
		rejudgeSubmission({ _id: submission._id })
			.then(() => {
				submission.cases = []
				submission.result = {
					time: 0, memory: 0,
					status: Status.WAIT
				}
				this.handleSubmission(submission)
			})
			.catch(message.error)
	}
	public componentDidMount() {
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
		if (this.socket) { this.socket.close() }
	}
	public render() {
		const { global, submission, pending } = this.state
		const { _id, uname, pid, ptitle } = submission
		const { result, cases, code, language } = submission
		const { open, createdAt } = submission
		const scase = cases && cases.length > 0
		const smain = result && result.status !== Status.WAIT
		const lan = global.languages[language]

		return <React.Fragment>
			<LoginTip />
			<Card
				loading={!_id}
				title={`Submission by ${uname || 'user'}`}
				extra={<React.Fragment>
					<Link to={`/problem/${pid}`}>{ptitle}</Link>
					{!pending && diffGroup(global.user, Group.admin) &&
					<React.Fragment>
						<Divider type="vertical" />
						<Button type="primary" children="Rejudge" onClick={this.rejudge} />
					</React.Fragment>}
				</React.Fragment>}
			>
				<Timeline pending={pending}>
					<Timeline.Item color="green">
						[{new Date(createdAt).toLocaleString()}] {uname} submitted the code
					</Timeline.Item>
					{!pending && scase && cases.map((c, i) => (
						<Timeline.Item key={i} color={color(c)}>
							<Tag>#{i}</Tag>
							{renderStatus(c)}
							<Tag>{parseTime(c.time)}</Tag>
							<Tag>{parseMemory(c.memory)}</Tag>
							<span>{c.extra}</span>
						</Timeline.Item>
					))}
					{!scase && smain && <Timeline.Item color={color(result)}>
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
