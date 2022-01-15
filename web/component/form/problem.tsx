import React, { useEffect } from 'react'
import { Form, FormInstance, Input, Select } from 'antd'
import { Editor } from '../../component/editor'
import { Number } from '../../component/number'
import { IProblem } from '../../interface'

const formItemLayout = {
	labelCol: { xs: 24, sm: 6, md: 4 },
	wrapperCol: { xs: 24, sm: 18, md: 20 }
}

const defaults = {
	timeLimit: 1.0,
	memoryLimit: 32 * 1024 * 1024
}

interface IProblemForm {
	problem?: IProblem
	form?: FormInstance<IProblem>
}

export function ProblemForm({ problem, form }: IProblemForm) {
	useEffect(() => form.resetFields(), [])
	return <Form initialValues={{ ...defaults, ...problem }} form={form}>
		<Form.Item label="Title" name="title" rules={[
			{ required: true, message: 'Please input problem title' }
		]} {...formItemLayout}>
			<Input placeholder="Problem title" />
		</Form.Item>
		<Form.Item label="Tags" name="tags" {...formItemLayout}>
			<Select tokenSeparators={[',']} mode="tags" placeholder="Problem tags" />
		</Form.Item>
		<Form.Item label="Time" name="timeLimit" rules={[
			{ required: true, message: 'Please input time limit' }
		]} {...formItemLayout}>
			<Number options={[
				{ name: 'ms', scale: 0.001 },
				{ name: 's', scale: 1 }
			]} default={0} />
		</Form.Item>
		<Form.Item label="Memory" name="memoryLimit" rules={[
			{ required: true, message: 'Please input memory limit' }
		]} {...formItemLayout}>
			<Number options={[
				{ name: 'Bytes', scale: 1 },
				{ name: 'KiB', scale: 1024 },
				{ name: 'MiB', scale: 1024 * 1024 },
				{ name: 'GiB', scale: 1024 * 1024 * 1024 }
			]} default={2} />
		</Form.Item>
		<Form.Item label="Data" name="data" {...formItemLayout}>
			<Input placeholder="Problem data id (use Manage/File to upload data)" />
		</Form.Item>
		<Form.Item label="Content" name="content" rules={[
			{ required: true, message: 'Please input problem content' }
		]} {...formItemLayout}>
			<Editor trusted />
		</Form.Item>
	</Form>
}
