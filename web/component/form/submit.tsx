import React from 'react'

import { message, Button, Checkbox, Divider, Form, Input, Select } from 'antd'

import { Status } from '../../../common/interface'
import { Code } from '../../component/code'
import { postSubmission } from '../../model'
import { ILanguage } from '../../util/interface'

interface SubmitFormProps {
	uid: string
	pid: string
	languages: ILanguage[]
	callback: (id: any) => any
}

export class SubmitForm extends React.Component<SubmitFormProps> {
	public state = {
		loading: false,
		language: undefined as string
	}
	private handleSubmit(values: any) {
		this.setState({ loading: true })
		postSubmission(values)
			.then((data) => {
				this.setState({ loading: false })
				message.success('submit success')
				if (data.result.status !== Status.FREEZE) {
					this.props.callback(data._id)
				}
			})
			.catch((err) => {
				this.setState({ loading: false })
				message.error(err)
			})
	}
	public static getDerivedStateFromProps(props: SubmitFormProps) {
		// lan: previous submission language OR undefined
		const lan = props.languages[localStorage.language]
		return { language: lan && lan.suffix }
	}
	public render() {
		return <Form
			onFinish={v => this.handleSubmit(v)}
			className="submit-form"
			initialValues={this.props}
		>
			<Form.Item name="uid" style={{ display: 'none' }}>
				<Input readOnly />
			</Form.Item>
			<Form.Item name="pid" style={{ display: 'none' }}>
				<Input readOnly />
			</Form.Item>
			<Form.Item name="code" rules={[
				{ required: true, message: 'Please input your code' }
			]}>
				<Code options={{ maxLines: 30 }} language={this.state.language} />
			</Form.Item>
			<Form.Item name="language" rules={[
				{ required: true, message: 'Please choose language' }
			]} initialValue={Number(localStorage.language) || 0}>
				<Select
					style={{ width: '50%', minWidth: '200px' }}
					placeholder="Choose language"
					onSelect={(value) => {
						const lan = this.props.languages[Number(value)]
						this.setState({ language: lan.suffix })
						localStorage.language = value
					}}
				>
					{this.props.languages.map((lan, idx) => <Select.Option
						key={idx}
						value={idx}
						children={lan.name}
					/>)}
				</Select>
			</Form.Item>
			<Form.Item>
				<Button type="primary" htmlType="submit" loading={this.state.loading}>
					Submit
				</Button>
				<Divider type="vertical" />
				<Form.Item name="open" valuePropName="checked" initialValue={true} noStyle>
					<Checkbox>Share my code to others</Checkbox>
				</Form.Item>
			</Form.Item>
		</Form>
	}
}
