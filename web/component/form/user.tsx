import React, { useContext, useEffect } from 'react'
import { Form, FormInstance, Input, Select } from 'antd'
import { Group } from '../../../common/interface'
import { diffGroup } from '../../../common/user'
import { IUser } from '../../util/interface'
import { GlobalContext } from '../../global'

const formItemLayout = {
	labelCol: { xs: 24, sm: 6, md: 4 },
	wrapperCol: { xs: 24, sm: 18, md: 20 }
}

interface IUserForm {
	user?: IUser
	form?: FormInstance<IUser>
}

export function UserForm({ user, form }: IUserForm) {
	const [global] = useContext(GlobalContext)
	useEffect(() => form.resetFields(), [])
	return <Form initialValues={user} form={form}>
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
		{global.user._id !== user?._id && <Form.Item label="Group" name="group" rules={[
			{ required: true, message: 'Please select user group' }
		]} initialValue={user ? undefined : 0} {...formItemLayout}>
			<Select placeholder="User group">
				<Select.Option value={0} children="Common" />
				{diffGroup(global.user, Group.admin, 1) && <Select.Option value={1} children="Admin" />}
			</Select>
		</Form.Item>}
		<Form.Item label="Introduction" name="introduction" {...formItemLayout}>
			<Input.TextArea rows={5} placeholder="User introduction" />
		</Form.Item>
	</Form>
}
