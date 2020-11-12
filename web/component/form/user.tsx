import * as React from 'react'

import { Form, Input, Select } from 'antd'
import { FormInstance } from 'antd/lib/form'

import { Group } from '../../../common/interface'
import { diffGroup } from '../../../common/user'
import { IUser } from '../../util/interface'
import { globalState } from '../../util/state'

interface UserFormProps {
	user?: IUser
	onRefForm: (form: FormInstance<IUser>) => void
}

export class UserForm extends React.Component<UserFormProps> {
	private formRef = React.createRef<FormInstance<IUser>>()

	componentDidMount() {
		this.props.onRefForm(this.formRef.current)
	}

	public render() {
		const { user } = this.props
		const self = globalState.user
		const isSelf = self._id === (user && user._id)
		const formItemLayout = {
			labelCol: { xs: 24, sm: 6, md: 4 },
			wrapperCol: { xs: 24, sm: 18, md: 20 }
		}
		return <Form initialValues={user} ref={this.formRef}>
			<Form.Item label="Name" name="name" rules={[
				{ required: true, message: 'Please input user name' },
				{ pattern: /^[a-zA-Z0-9][a-zA-Z0-9_]{2,14}$/, message: 'Invalid pattern' }
			]} {...formItemLayout}>
				<Input placeholder="Letter, number or underscore (length 3-15)" />
			</Form.Item>
			<Form.Item label="Mail" name="mail" rules={[
				{ required: true, message: 'Please input user mail' },
				{ type: 'email', message: 'Invalid email pattern' }
			]} {...formItemLayout}>
				<Input placeholder="Valid mail (for password retrieve) is required" />
			</Form.Item>
			{!user && <Form.Item label="Password" name="password" rules={[
				{ required: true, message: 'Please input user password' },
				{ min: 6, max: 20, message: 'Length of password should be 6-20' }
			]} {...formItemLayout}>
				<Input placeholder="User password" />
			</Form.Item>}
			{!isSelf && <Form.Item label="Group" name="group" rules={[
				{ required: true, message: 'Please select user group' }
			]} initialValue={0} {...formItemLayout}>
				<Select placeholder="User group">
					<Select.Option value={0} children="Common" />
					{diffGroup(self, Group.admin, 1) && <Select.Option value={1} children="Admin" />}
				</Select>
			</Form.Item>}
			<Form.Item label="Introduction" name="introduction" {...formItemLayout}>
				<Input.TextArea
					rows={5}
					placeholder="User introduction"
				/>
			</Form.Item>
		</Form>
	}
}
