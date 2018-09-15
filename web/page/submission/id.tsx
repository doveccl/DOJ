import * as React from 'react'
import { withRouter, Link } from 'react-router-dom'

import { message, Card, Checkbox, Timeline } from 'antd'

import Code from '../../component/code'
import LoginTip from '../../component/login-tip'
import { getSubmission, hasToken, putSubmission } from '../../model'
import { isGroup } from '../../util/function'
import { HistoryProps, ISubmission, MatchProps } from '../../util/interface'
import { addListener, globalState, removeListener, updateState } from '../../util/state'
import { renderMemory, renderStatus, renderTime } from './index'

class Submission extends React.Component<HistoryProps & MatchProps> {
	public state = {
		global: globalState,
		submission: {} as ISubmission
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
				.then((submission) => this.setState({ submission }))
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
		const lan = global.languages[language]

		return <React.Fragment>
			<LoginTip />
			<Card
				loading={!_id}
				title={`Submission by ${uname || 'user'}`}
				extra={<Link to={`/problem/${pid}`}>{ptitle}</Link>}
			>
				<Timeline pending="Pending ...">
					<Timeline.Item color="blue">
						[{new Date(createdAt).toLocaleString()}] {uname} submitted the code
					</Timeline.Item>
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
							!isGroup(global.user, 'admin') &&
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
