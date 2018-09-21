import * as ace from 'ace-builds'

function escapeHtml(unsafe?: string) {
	if (!unsafe) { return '' }
	return unsafe
		.replace(/&/g, '&amp;')
		.replace(/</g, '&lt;')
		.replace(/>/g, '&gt;')
		.replace(/"/g, '&quot;')
		.replace(/'/g, '&apos;')
}

const loadModule = ace.require('ace/config').loadModule
const TextLayer = ace.require('ace/layer/text').Text
const EditSession = ace.require('ace/edit_session').EditSession

class Element {
	private type: string
	public text = ''
	public className: string
	constructor(type: string) {
		this.type = type
	}
	public cloneNode() {
		return this
	}
	public appendChild(child: Element | HTMLElement) {
		if (child instanceof HTMLElement) {
			this.text += child.outerHTML
		} else {
			this.text += child.toString()
		}
	}
	public toString() {
		let result = ''
		if (this.type !== 'fragment') {
			let c = this.className
			c = c ? ` class="${c}"` : ''
			result = `<${this.type}${c}>`
		}
		if (this.text) {
			result += this.text
		}
		if (this.type !== 'fragment') {
			result += `</${this.type}>`
		}
		return result
	}
}

const simpleDom = {
	createTextNode: (text: string) => escapeHtml(text),
	createElement: (type: string) => new Element(type),
	createFragment: () => new Element('fragment')
}

const SimpleTextLayer: any = class {
	private config = {}
	private dom = simpleDom
}
SimpleTextLayer.prototype = TextLayer.prototype

const cacheMode: any = {}
const getMode = (mode: string, callback: (mode: any) => any) => {
	if (cacheMode[mode]) { return callback(cacheMode[mode]) }
	loadModule([ 'mode', mode ], (m: any) => {
		callback(cacheMode[mode] = new m.Mode())
	})
}

export default (el: HTMLElement, opts: Partial<ace.Ace.EditorOptions>) => {
	opts.value = opts.value || ''
	opts.theme = opts.theme || 'ace/theme/textmate'
	if (!opts.mode) { return el.textContent = opts.value }
	const style = ace.require(opts.theme).cssClass

	getMode(opts.mode, (mode) => {
		const session = new EditSession(opts.value, mode)
		session.setUseWorker(false)

		const textLayer = new SimpleTextLayer()
		textLayer.setSession(session)
		for (const k in textLayer.$tabStrings) {
			if (typeof textLayer.$tabStrings[k] === 'string') {
				const ele = simpleDom.createFragment()
				ele.text = textLayer.$tabStrings[k]
				textLayer.$tabStrings[k] = ele
			}
		}

		const length = session.getLength()
		const wrapper = simpleDom.createElement('div')
		wrapper.className = `ace_static_highlight ${style}`

		for (let i = 0; i < length; i++) {
			const line = simpleDom.createElement('div')
			line.className = 'ace_line'
			textLayer.$renderLine(line, i, false)
			wrapper.appendChild(line)
		}

		el.innerHTML = wrapper.toString()
	})
}
