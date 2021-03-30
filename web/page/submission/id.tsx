import React from 'react'
import { Manager, Socket } from 'socket.io-client'

import { CheckboxChangeEvent } from 'antd/lib/checkbox'
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
		submission: {} as ISubmission,
		edit: false
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
			this.socket.on('step', (data: any) => {
				this.state.submission.cases = data.cases
				this.setState({
					pending: data.pending,
					submission: this.state.submission
				})
			})
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
					id: s._id,
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
	private shareChange = (e: CheckboxChangeEvent) => {
		const s = this.state.submission
		s.open = e.target.checked
		putSubmission(s._id, s).then(() => this.forceUpdate()).catch(message.error)
	}
	private codeChange = (code: string) => {
		this.state.submission.code = code
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
		if (this.socket) this.socket.close()
	}
	public render() {
		const { global, submission, pending, edit } = this.state
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
				<Timeline className="result" pending={pending}>
					<Timeline.Item color="green">
						[{new Date(createdAt).toLocaleString()}] {uname} submitted the code
					</Timeline.Item>
					{scase && cases.map((c, i) => (
						<Timeline.Item key={i} color={color(c)}>
							<Tag>#{i}</Tag>
							{renderStatus(c)}
							<Tag>{parseTime(c.time)}</Tag>
							<Tag>{parseMemory(c.memory)}</Tag>
							{c.extra && (/.\n./.test(c.extra) ?
							  <pre>{c.extra.trim()}</pre> :
								<span>{c.extra}</span>
							)}
						</Timeline.Item>
					))}
					{!scase && smain && <Timeline.Item color={color(result)}>
						{renderStatus(result)}
						<Tag>{parseTime(result.time)}</Tag>
						<Tag>{parseMemory(result.memory)}</Tag>
						{result.extra && (/.\n./.test(result.extra) ?
							<pre>{result.extra.trim()}</pre> :
							<span>{result.extra}</span>
						)}
					</Timeline.Item>}
				</Timeline>
			</Card>
			{code && <React.Fragment>
				<div className="divider" />
				<Card
					title="Code"
					extra={<React.Fragment>
						<Checkbox
							checked={open}
							children="Public"
							onChange={this.shareChange}
							disabled={
								!diffGroup(global.user, Group.admin) &&
								global.user._id !== submission.uid
							}
						/>
						{diffGroup(global.user, Group.root) && <React.Fragment>
							<Divider type="vertical" />
							<Button
								type={edit ? 'primary' : 'default'}
								children={edit ? 'Update' : 'Edit'}
								onClick={() => {
									if (edit) {
										putSubmission(_id, submission).then(() => {
											this.rejudge()
											this.setState({ edit: false })
										}).catch(message.error)
									} else {
										this.setState({ edit: true })
									}
								}}
							/>
						</React.Fragment>}
					</React.Fragment>}
				>
					<Code
						value={code}
						static={!edit}
						language={lan && lan.suffix}
						onChange={this.codeChange}
					/>
				</Card>
			</React.Fragment>}
		</React.Fragment>
	}
}

export default withRouter(({ history, match }) => <Submission history={history} match={match} />)
