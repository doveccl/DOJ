import * as React from 'react'

import { Form, Input, Select } from 'antd'
import { FormComponentProps } from 'antd/lib/form'
import { Group } from '../../../common/interface'
import { diffGroup } from '../../../common/user'
import { IUser } from '../../util/interface'
import { globalState } from '../../util/state'

interface UserFormProps extends FormComponentProps {
	value?: IUser
}

class UserForm extends React.Component<UserFormProps> {
	public render() {
		const { value } = this.props
		const { getFieldDecorator } = this.props.form
		const self = globalState.user
		const isSelf = self._id === (value && value._id)
		const formItemLayout = {
			labelCol: { xs: 24, sm: 6, md: 4 },
			wrapperCol: { xs: 24, sm: 18, md: 20 }
		}
		return <Form>
			<Form.Item label="Name" {...formItemLayout}>
				{getFieldDecorator('name', {
					initialValue: value && value.name,
					rules: [
						{ required: true, message: 'Please input user name' },
						{ pattern: /^[a-zA-Z0-9][a-zA-Z0-9_]{2,14}$/, message: 'Invalid pattern' }
					]
				})(
					<Input placeholder="Letter, number or underscore (length 3-15)" />
				)}
			</Form.Item>
			<Form.Item label="Mail" {...formItemLayout}>
				{getFieldDecorator('mail', {
					initialValue: value && value.mail,
					rules: [
						{ required: true, message: 'Please input user mail' },
						{ type: 'email', message: 'Invalid email pattern' }
					]
				})(
					<Input placeholder="Valid mail (for password retrieve) is required" />
				)}
			</Form.Item>
			{!value && <Form.Item label="Password" {...formItemLayout}>
				{getFieldDecorator('password', {
					rules: [
						{ required: true, message: 'Please input user password' },
						{ min: 6, max: 20, message: 'Length of password should be 6-20' }
					]
				})(
					<Input placeholder="User password" />
				)}
			</Form.Item>}
			{!isSelf && <Form.Item label="Group" {...formItemLayout}>
				{getFieldDecorator('group', {
					initialValue: value ? value.group : 0,
					rules: [{ required: true}]
				})(
					<Select placeholder="User group">
						<Select.Option value={0} children="Common" />
						{diffGroup(self, Group.admin, 1) && <Select.Option value={1} children="Admin" />}
					</Select>
				)}
			</Form.Item>}
			<Form.Item label="Introduction" {...formItemLayout}>
				{getFieldDecorator('introduction', {
					initialValue: value && value.introduction
				})(
					<Input.TextArea
						rows={5}
						placeholder="User introduction"
					/>
				)}
			</Form.Item>
		</Form>
	}
}

export const WrappedUserForm = Form.create()(UserForm)
