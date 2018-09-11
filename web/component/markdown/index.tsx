import * as React from 'react'
import * as Markdown from 'react-markdown'
import * as math from 'remark-math'
import * as shortcodes from 'remark-shortcodes'

import { renderToString } from 'katex'
import { highlightAuto } from 'highlight.js'

import PDF from '../pdf'

import 'katex/dist/katex.min.css'
import 'highlight.js/styles/default.css'

const renderMath = (math: string, displayMode = false) => {
	try {
		const __html = renderToString(math, { displayMode })
		return <span dangerouslySetInnerHTML={{ __html }} />
	} catch {
		return <span style={{ color: 'red' }}>error</span>
	}
}

export default class extends React.Component<Markdown.ReactMarkdownProps> {
	render() {
		return <Markdown
			{ ...this.props }
			plugins={[ math, shortcodes ]}
			renderers={{
				code: ({ value, language }) => {
					if (!language) { return <pre className="hljs">{value}</pre> }
					const { value: __html } = highlightAuto(value, [ language ])
					return <pre className="hljs" dangerouslySetInnerHTML={{ __html }} />
				},
				shortcode: ({ identifier, attributes }) => {
					let { id, url } = attributes
					url = url ? url : `/api/file/${id}`
					switch (identifier.toLowerCase()) {
						case 'pdf': return <PDF file={url} />
						case 'img': return <img src={url} />
						default: return <span>Unsupported: {identifier}</span>
					}
				},
				math: ({ value }) => renderMath(value, true),
				inlineMath: ({ value }) => renderMath(value)
			}}
		/>
	}
}
