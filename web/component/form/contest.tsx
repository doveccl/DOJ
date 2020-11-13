import moment from 'moment'
import * as React from 'react'

import { DatePicker, Form, Input, Select } from 'antd'
import { FormInstance } from 'antd/lib/form'

import { ContestType } from '../../../common/interface'
import { Editor } from '../../component/editor'
import { IContest } from '../../util/interface'

interface ContestFormProps {
	contest?: IContest
	onRefForm: (form: FormInstance<IContest>) => void
}

export class ContestForm extends React.Component<ContestFormProps> {
	private formRef = React.createRef<FormInstance>()
	private adjustFreeze = (t?: Array<moment.Moment> | number) => {
		const form = this.formRef.current
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

	componentDidMount() {
		this.props.onRefForm(this.formRef.current)
	}

	public render() {
		const formItemLayout = {
			labelCol: { xs: 24, sm: 6, md: 4 },
			wrapperCol: { xs: 24, sm: 18, md: 20 }
		}

		const contest = Object.assign({ time: null }, this.props.contest)
		contest.freezeAt = contest.freezeAt && moment(contest.freezeAt) as any
		if (contest.startAt && contest.endAt) {
			contest.time = [
				moment(this.props.contest.startAt),
				moment(this.props.contest.endAt)				
			]
		}

		return <Form initialValues={contest} ref={this.formRef}>
			<Form.Item label="Title" name="title" rules={[
				{ required: true, message: 'Please input contest title' }
			]} {...formItemLayout}>
				<Input placeholder="Contest title" />
			</Form.Item>
			<Form.Item label="Type" name="type" rules={[
				{ required: true, message: 'Please select type' }
			]} {...formItemLayout}>
				<Select onSelect={t => this.adjustFreeze(Number(t))}>
					<Select.Option value={ContestType.OI}>OI</Select.Option>
					<Select.Option value={ContestType.ICPC}>ICPC</Select.Option>
				</Select>
			</Form.Item>
			<Form.Item name="startAt" style={{ display: 'none' }}>
				<Input />
			</Form.Item>
			<Form.Item name="endAt" style={{ display: 'none' }}>
				<Input />
			</Form.Item>
			<Form.Item label="Time" name="time" rules={[
				{ required: true, message: 'Please select time' }
			]} {...formItemLayout}>
				<DatePicker.RangePicker
					showTime={true}
					onChange={this.adjustFreeze}
					format="YYYY-MM-DD HH:mm:ss"
				/>
			</Form.Item>
			<Form.Item label="Freeze time" name="freezeAt" {...formItemLayout}>
				<DatePicker
					showTime={true}
					format="YYYY-MM-DD HH:mm:ss"
				/>
			</Form.Item>
			<Form.Item label="Description" name="description" {...formItemLayout}>
				<Editor escapeHtml={false} />
			</Form.Item>
		</Form>
	}
}
