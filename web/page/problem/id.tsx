import * as React from 'react'
import { withRouter } from 'react-router-dom'
import { Card, Input, Button, message } from 'antd'

import * as model from '../../model'
import Markdown from '../../component/markdown'
import LoginTip from '../../component/login-tip'
import { IProblem, HistoryProps, MatchProps } from '../../util/interface'
import { updateState } from '../../util/state'

class Problem extends React.Component<HistoryProps & MatchProps> {
	state = { problem: {} as IProblem }
	componentWillMount() {
		const { params } = this.props.match
		updateState({ path: [
			{ url: '/problem', text: 'Problem' }, params.id
		] })
		if (!model.hasToken()) { return }
		model.getProblem(params.id)
			.then(problem => this.setState({ problem }))
			.catch(err => message.error(err))
	}
	render() {
		return <React.Fragment>
			<LoginTip />
			<Card
				loading={!Boolean(this.state.problem.content)}
				title={this.state.problem.title || 'Problem'}
			>
				<Markdown
					escapeHtml={false}
					source={this.state.problem.content}
				/>
			</Card>
			<div className="divider" />
			<Card title="Submit">
				submit
			</Card>
		</React.Fragment>
	}
}

export default withRouter(({ history, match }) => <Problem history={history} match={match} />)
