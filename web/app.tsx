import * as React from 'react'
import { Layout } from 'antd'

export default class extends React.Component {
	render() {
		return <Layout>
			<Layout.Header>
				header
			</Layout.Header>
			<Layout.Content>
				content
			</Layout.Content>
			<Layout.Footer>
				footer
			</Layout.Footer>
		</Layout>
	}
}
