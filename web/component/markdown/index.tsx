import './index.less'
import hljs from 'highlight.js'
import React, { useEffect, useRef } from 'react'
import { marked } from 'marked'
import { message } from 'antd'
import { sanitize } from 'dompurify'
import { renderToString } from 'katex'

const mathRegExp = /(\${1,2})([\s\S]+?)\1/g
const codeRegExp = /\s*([`~]{3})(\w*)\n([\s\S]+?)\1/g
const shortCodeRegExp = /\[\[\s*(\w+)(?:\s+(\w+)="(\S+)")+\s*\]\]/g

export function MarkDown({ trusted = false, children = '' }) {
	const ref = useRef<HTMLDivElement>()

	useEffect(() => {
		const codes: string[] = []
		const noCodeStr = children
			.replaceAll('\r', '')
			// replace code blocks with empty string
			// avoid sanitize: '#include<xxx>' => '#include'
			.replace(codeRegExp, (_, tok, lan, code) => {
				codes.push(code)
				return `\n${tok}${lan}\n${tok}`
			})
			// ensure an empty line before title
			.replace(/\s*\n(#{1,6})/g, '\n\n$1')
			.trimStart()

		// trusted ? parse shortcode : sanitize it
		const safeStr = trusted ? noCodeStr.replace(shortCodeRegExp, (_, tag, ...args) => {
			let id = '', src = ''
			for (let i = 1; i < args.length; i += 2) {
				switch (args[i - 1]) {
					case 'id': id = args[i]; break
					case 'src': src = args[i]; break
				}
			}
			if (id) src = `/api/file/${id}`
			switch (src && tag.toLowerCase()) {
				case 'pdf': return `<object class="pdf" data=${src} />`
				case 'img': return `<img src=${src} />`
				default: return `<red>[${tag}]</red>`
			}
		}) : sanitize(noCodeStr, { ALLOWED_TAGS: [] })

		// parse math block $...$, $$...$$ with katex
		const result = safeStr.replace(mathRegExp, (_, tok, math) => {
			const displayMode = tok === '$$'
			const tag = displayMode ? 'div' : 'span'
			try {
				return renderToString(math, { displayMode })
			} catch {
				const el = document.createElement(tag)
				el.style.color = 'red'
				el.textContent = math
				return el.outerHTML
			}
		})

		ref.current.innerHTML = marked(result)
		ref.current.querySelectorAll<HTMLElement>('pre>code:not(.hljs)').forEach((el, idx) => {
			el.textContent = codes[idx]
			el.addEventListener('click', () => {
				navigator.clipboard.writeText(codes[idx])
				message.success('Copied')
			})
			hljs.highlightElement(el)
			// @ts-ignore no typedef
			hljs.lineNumbersBlock?.(el)
		})
	}, [trusted, children])

	return <div ref={ref} className="markdown-body" />
}
