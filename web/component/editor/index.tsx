import React from 'react'
import { Code } from '../code'
import { MarkDown } from '../markdown'
import { EyeOutlined } from '@ant-design/icons'

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
		return <div className="editor">
			<div className="code">
				<Code
					language="markdown"
					value={this.state.content}
					onChange={this.onChange}
				/>
			</div>
			<div className="preview">
				<EyeOutlined className="button" />
				<div className="view">
					<MarkDown
						children={this.state.content}
						shortCode={this.props.shortCode}
						allowDangerousHtml={this.props.allowDangerousHtml}
					/>
				</div>
			</div>
		</div>
	}
}
