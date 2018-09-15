import * as React from 'react'
import { withRouter } from 'react-router-dom'

import { message, Card, Tag } from 'antd'

import WrappedSubmitForm from '../../component/form/submit'
import LoginTip from '../../component/login-tip'
import Markdown from '../../component/markdown'
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
				extra={(problem.tags || []).map((t, i) => <Tag key={i}>{t}</Tag>)}
			>
				<Markdown
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
				{this.state.tabKey === 'submit' && global.user && <WrappedSubmitForm
					callback={(id) => this.props.history.push(`/submission/${id}`)}
					languages={this.state.global.languages}
					uid={this.state.global.user._id}
					pid={this.state.problem._id}
				/>}
			</Card>
		</React.Fragment>
	}
}

export default withRouter(({ history, match }) => <Problem history={history} match={match} />)