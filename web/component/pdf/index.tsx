import React from 'react'

interface PDFProps { file: string }

export class PDF extends React.Component<PDFProps> {
	public render() {
		return <p style={{
			width: '100%',
			height: '100vh'
		}}>
			<object
				type="application/pdf"
				width="100%"
				height="100%"
				data={this.props.file}
			>
				<a target="_blank" href={this.props.file}>
					click here to download the PDF
				</a>
			</object>
		</p>
	}
}
