import * as React from 'react'

import { Col, Row } from 'antd'

import Code from '../code'
import Markdown from '../markdown'

interface EditorProps {
	value?: string
	sortCode?: boolean
	escapeHtml?: boolean
	onChange?: (value: string) => any
}

export default class extends React.Component<EditorProps> {
	public state = { content: this.props.value }
	private onChange = (content: string) => {
		if (this.props.onChange) {
			this.props.onChange(content)
			this.setState({ content })
		}
	}
	public render() {
		return <Row>
			<Col span={12}>
				<Code
					language="markdown"
					value={this.state.content}
					onChange={this.onChange}
				/>
			</Col>
			<Col span={12}>
				<Markdown
					escapeHtml={this.props.escapeHtml}
					source={this.state.content}
				/>
			</Col>
		</Row>
	}
}
