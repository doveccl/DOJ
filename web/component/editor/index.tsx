import * as React from 'react'

import { Col, Row } from 'antd'

import { Code } from '../code'
import { MarkDown } from '../markdown'

import './index.less'

interface EditorProps {
	value?: string
	shortCode?: boolean
	escapeHtml?: boolean
	onChange?: (value: string) => any
}

export class Editor extends React.Component<EditorProps> {
	public state = { content: this.props.value }
	private onChange = (content: string) => {
		this.setState({ content })
		if (this.props.onChange) {
			this.props.onChange(content)
		}
	}
	public render() {
		return <Row className="editor" gutter={16}>
			<Col span={12} className="code">
				<Code
					language="markdown"
					value={this.state.content}
					onChange={this.onChange}
				/>
			</Col>
			<Col span={12} className="preview">
				<MarkDown
					shortCode={this.props.shortCode}
					escapeHtml={this.props.escapeHtml}
					source={this.state.content}
				/>
			</Col>
		</Row>
	}
}
