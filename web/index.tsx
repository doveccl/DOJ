import './index.less'
import React from 'react'
import { render } from 'react-dom'
import { BrowserRouter } from 'react-router-dom'
import { Layout } from 'antd'
import { GlobalProvider } from './global'
import { Path } from './component/path'
import { Footer, Header, Sider } from './layout'
import { Router } from './router'

const container = document.createElement('div')
document.body.appendChild(container)

render(<BrowserRouter>
	<GlobalProvider>
		<Layout className="container">
				<Sider />
				<Layout>
					<Header />
					<Layout.Content className="content">
						<Path />
						<Router />
					</Layout.Content>
					<Footer />
				</Layout>
		</Layout>
	</GlobalProvider>
</BrowserRouter>, container)
