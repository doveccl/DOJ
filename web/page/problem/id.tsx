import * as React from 'react'
import { withRouter, Link } from 'react-router-dom'

import { message, Card, Divider, Tag } from 'antd'

import { parseMemory, parseTime } from '../../../common/function'
import { Discuss } from '../../component/discuss'
import { WrappedSubmitForm } from '../../component/form/submit'
import { LoginTip } from '../../component/login-tip'
import { MarkDown } from '../../component/markdown'
import { getProblem, hasToken } from '../../model'
import { HistoryProps, IProblem, MatchProps } from '../../util/interface'
import { addListener, globalState, removeListener, updateState } from '../../util/state'

class Problem extends React.Component<HistoryProps & MatchProps> {
	public state = {
		code: '',
		tabKey: 'submit',
		problem: {} as IProblem,
		global: globalState
	}
	public componentWillMount() {
		const { params } = this.props.match
		updateState({ path: [
			{ url: '/problem', text: 'Problem' }, params.id
		] })
		addListener('problem', (global) => {
			this.setState({ global })
		})
		if (hasToken()) {
			getProblem(params.id)
				.then((problem) => this.setState({ problem }))
				.catch((err) => message.error(err))
		}
	}
	public componentWillUnmount() {
		removeListener('problem')
	}
	public render() {
		const { global, problem } = this.state
		return <React.Fragment>
			<LoginTip />
			<Card
				loading={!problem._id}
				title={problem.title || 'Problem'}
				extra={problem._id && <React.Fragment>
					<Link to={`/submission?pid=${problem._id}`}>
						Solutions ({problem.solve}/{problem.submit})
					</Link>
					<Divider type="vertical" />
					<Tag color="volcano">{parseTime(problem.timeLimit)}</Tag>
					<Tag color="orange">{parseMemory(problem.memoryLimit)}</Tag>
				</React.Fragment>}
			>
				<MarkDown
					shortCode={true}
					escapeHtml={false}
					source={problem.content}
				/>
			</Card>
			<div className="divider" />
			<Card
				tabList={[
					{ key: 'submit', tab: 'Submit' },
					{ key: 'discuss', tab: 'Discuss' }
				]}
				activeTabKey={this.state.tabKey}
				onTabChange={(tabKey) => this.setState({ tabKey })}
				loading={!global.user || !problem._id}
			>
				{this.state.tabKey === 'submit' && global.languages.length > 0 && <WrappedSubmitForm
					languages={global.languages} uid={global.user._id} pid={problem._id}
					callback={(id) => this.props.history.push(`/submission/${id}`)}
				/>}
				{this.state.tabKey === 'discuss' && global.user && <Discuss topic={problem._id} />}
			</Card>
		</React.Fragment>
	}
}

export default withRouter(({ history, match }) => <Problem history={history} match={match} />)
