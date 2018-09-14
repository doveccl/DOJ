import * as React from 'react'
import * as Markdown from 'react-markdown'
import * as math from 'remark-math'
import * as shortcodes from 'remark-shortcodes'

import Code from '../code'
import PDF from '../pdf'

import { renderToString } from 'katex'

const renderMath = (tex: string, displayMode = false) => {
	try {
		const html = renderToString(tex, { displayMode })
		return <span dangerouslySetInnerHTML={{ __html: html }} />
	} catch {
		return <span style={{ color: 'red' }}>error</span>
	}
}

export default class extends React.Component<Markdown.ReactMarkdownProps> {
	public render() {
		return <Markdown
			{ ...this.props }
			plugins={[ math, shortcodes ]}
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
