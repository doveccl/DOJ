import * as React from 'react'
import { Document, Page, setOptions } from 'react-pdf'

import './index.less'

setOptions({ workerSrc: '/pdf.worker.js' })

export default class extends React.Component<any> {
	state = { numPages: 0 }
	render() {
		return <div>
			<Document
				{ ...this.props }
				className="pdf-document"
				onLoadSuccess={({ numPages }: any) => this.setState({ numPages })}
			>
				{new Array(this.state.numPages).fill(0).map((_, i) => <Page
					key={`p_${i}`}
					renderMode="svg"
					className="pdf-page"
					pageNumber={i + 1}
				/>)}
			</Document>
		</div>
	}
}
