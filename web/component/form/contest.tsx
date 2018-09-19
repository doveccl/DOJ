import * as moment from 'moment'
import * as React from 'react'

import { DatePicker, Form, Input, Select } from 'antd'
import { FormComponentProps } from 'antd/lib/form'

import Editor from '../../component/editor'
import { ContestType, IContest } from '../../util/interface'

interface ContestFormProps extends FormComponentProps {
	value?: IContest
}

class ContestForm extends React.Component<ContestFormProps> {
	private adjustFreeze = (t?: any) => {
		const { form } = this.props
		const type = typeof t === 'number' ? t : form.getFieldValue('type')
		const time = Array.isArray(t) ? t : form.getFieldValue('time')
		if (time && time[0] && time[1]) {
			const atStart = moment(time[0])
			const beforeEnd = moment(time[1]).subtract(1, 'hours')
			form.setFieldsValue({ startAt: time[0].format(), endAt: time[1].format() })
			switch (type) {
				case ContestType.OI:
					form.setFieldsValue({ freezeAt: atStart })
					break
				case ContestType.ICPC:
					form.setFieldsValue({ freezeAt: beforeEnd })
			}
		}
	}
	public render() {
		const { value } = this.props
		const { getFieldDecorator } = this.props.form
		const formItemLayout = {
			labelCol: { xs: 24, sm: 6, md: 4 },
			wrapperCol: { xs: 24, sm: 18, md: 20 }
		}
		return <Form>
			<Form.Item label="Title" {...formItemLayout}>
				{getFieldDecorator('title', {
					initialValue: value && value.title,
					rules: [{ required: true, message: 'Please input contest title' }]
				})(
					<Input placeholder="Contest title" />
				)}
			</Form.Item>
			<Form.Item label="Type" {...formItemLayout}>
				{getFieldDecorator('type', {
					initialValue: value ? value.type : ContestType.OI,
					rules: [{ required: true, message: 'Please select type' }]
				})(
					<Select onSelect={this.adjustFreeze}>
						<Select.Option value={ContestType.OI}>OI</Select.Option>
						<Select.Option value={ContestType.ICPC}>ICPC</Select.Option>
					</Select>
				)}
			</Form.Item>
			<Form.Item style={{ display: 'none' }}>
				{getFieldDecorator('startAt', { initialValue: value && value.startAt })(<Input />)}
			</Form.Item>
			<Form.Item style={{ display: 'none' }}>
				{getFieldDecorator('endAt', { initialValue: value && value.endAt })(<Input />)}
			</Form.Item>
			<Form.Item label="Time" {...formItemLayout}>
				{getFieldDecorator('time', {
					rules: [{ required: true, message: 'Please select time' }],
					initialValue: value && [
						moment(value.startAt),
						moment(value.endAt)
					]
				})(
					<DatePicker.RangePicker
						showTime={true}
						onOk={this.adjustFreeze}
						format="YYYY-MM-DD HH:mm:ss"
					/>
				)}
			</Form.Item>
			<Form.Item label="Freeze time" {...formItemLayout}>
				{getFieldDecorator('freezeAt', {
					initialValue: value && moment(value.freezeAt)
				})(
					<DatePicker
						showTime={true}
						format="YYYY-MM-DD HH:mm:ss"
					/>
				)}
			</Form.Item>
			<Form.Item label="Description" {...formItemLayout}>
				{getFieldDecorator('description', {
					initialValue: value && value.description
				})(
					<Editor escapeHtml={false} />
				)}
			</Form.Item>
		</Form>
	}
}

export default Form.create()(ContestForm)
