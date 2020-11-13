import React from 'react'
import Markdown from 'react-markdown'
import math from 'remark-math'
import shortcodes from 'remark-shortcodes'

import { Code } from '../code'
import { PDF } from '../pdf'

import { renderToString } from 'katex'

const renderMath = (tex: string, displayMode = false) => {
	try {
		const html = renderToString(tex, { displayMode })
		return <span dangerouslySetInnerHTML={{ __html: html }} />
	} catch {
		return <span style={{ color: 'red' }}>error</span>
	}
}

type MarkdownProps = Markdown.ReactMarkdownProps & {
	shortCode?: boolean
}

export class MarkDown extends React.Component<MarkdownProps> {
	public shouldComponentUpdate(nextPorps: MarkdownProps) {
		return nextPorps.source !== this.props.source
	}
	public render() {
		const plugins = [ math ]
		if (this.props.shortCode) {
			plugins.push(shortcodes)
		}
		return <Markdown
			{ ...this.props }
			plugins={plugins}
			renderers={{
				code: ({ value, language }) => <Code
					static value={value} language={language}
				/>,
				shortcode: ({ identifier, attributes }) => {
					const { id, url } = attributes
					const link = url || `/api/file/${id}`
					switch (identifier.toLowerCase()) {
						case 'pdf': return <PDF file={link} />
						case 'img': return <img src={link} />
						default: return <span>Unsupported: {identifier}</span>
					}
				},
				math: ({ value }) => renderMath(value, true),
				inlineMath: ({ value }) => renderMath(value)
			}}
		/>
	}
}
