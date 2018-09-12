import * as React from 'react'
import { withRouter } from 'react-router-dom'
import {
	Card, Form, Input, Select,
	Checkbox, Button, Divider, message
} from 'antd'
import { FormComponentProps } from 'antd/lib/form'

import * as model from '../../model'
import * as state from '../../util/state'
import Code from '../../component/code'
import Markdown from '../../component/markdown'
import LoginTip from '../../component/login-tip'
import { IProblem, ILanguage, HistoryProps, MatchProps } from '../../util/interface'

interface SubmitFormProps {
	uid: string
	pid: string
	languages: ILanguage[]
}

class SubmitForm extends React.Component<SubmitFormProps & FormComponentProps> {
	state = {
		loading: false,
		language: undefined as string
	}
	handleSubmit = (e: React.FormEvent) => {
		e.preventDefault()
		this.props.form.validateFields((err, values) => {
			if (!err) {
				console.log(values)
			}
		})
	}
	render() {
		const { uid, pid } = this.props
		const { getFieldDecorator } = this.props.form
		return <Form onSubmit={this.handleSubmit} className="submit-form">
			<Form.Item style={{ display: 'none' }}>
				{getFieldDecorator('uid', { initialValue: uid })(<Input readOnly />)}
			</Form.Item>
			<Form.Item style={{ display: 'none' }}>
				{getFieldDecorator('pid', { initialValue: pid })(<Input readOnly />)}
			</Form.Item>
			<Form.Item>
				{getFieldDecorator('code', {
					rules: [{ required: true, message: 'Please input your code' }]
				})(
					<Code options={{ maxLines: 30 }} language={this.state.language} />
				)}
			</Form.Item>
			<Form.Item>
				{getFieldDecorator('language', {
					rules: [{ required: true, message: 'Please choose language' }]
				})(
					<Select
						placeholder="Choose language"
						onSelect={value => {
							const lan = this.props.languages[Number(value)]
							this.setState({ language: lan.suffix })
						}}
					>
						{this.props.languages.map((lan, idx) => <Select.Option
							key={idx}
							children={lan.name}
						/>)}
					</Select>
				)}
			</Form.Item>
			<Form.Item>
				<Button type="primary" htmlType="submit" loading={this.state.loading}>
					Submit
				</Button>
				<Divider type="vertical" />
				{getFieldDecorator('open', {
					valuePropName: 'checked',
					initialValue: true
				})(
					<Checkbox>Share my code to others</Checkbox>
				)}
			</Form.Item>
		</Form>
	}
}

const WrappedSubmitForm = Form.create()(SubmitForm)

class Problem extends React.Component<HistoryProps & MatchProps> {
	state = {
		code: '',
		problem: {} as IProblem,
		global: state.globalState
	}
	componentWillMount() {
		const { params } = this.props.match
		state.updateState({ path: [
			{ url: '/problem', text: 'Problem' }, params.id
		] })
		state.addListener('problem', global => this.setState({ global }))
		if (!model.hasToken()) { return }
		model.getProblem(params.id)
			.then(problem => this.setState({ problem }))
			.catch(err => message.error(err))
	}
	componentWillUnmount() {
		state.removeListener('problem')
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
			<Card
				title="Submit"
				loading={
					!this.state.global.user ||
					!this.state.problem._id ||
					this.state.global.languages.length === 0
				}
			>
				{this.state.global.user && <WrappedSubmitForm
					languages={this.state.global.languages}
					uid={this.state.global.user._id}
					pid={this.state.problem._id}
				/>}
			</Card>
		</React.Fragment>
	}
}

export default withRouter(({ history, match }) => <Problem history={history} match={match} />)
