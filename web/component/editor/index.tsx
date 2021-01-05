import React from 'react'

import { Col, Row } from 'antd'

import { Code } from '../code'
import { MarkDown } from '../markdown'

import './index.less'

interface EditorProps {
	value?: string
	shortCode?: boolean
	allowDangerousHtml?: boolean
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
					children={this.state.content}
					shortCode={this.props.shortCode}
					allowDangerousHtml={this.props.allowDangerousHtml}
				/>
			</Col>
		</Row>
	}
}
