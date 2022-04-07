import './index.less'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { Layout } from 'antd'
import { GlobalProvider } from './global'
import { Path } from './component/path'
import { Footer, Header, Sider } from './layout'
import { Router } from './router'

const container = document.createElement('div')
document.body.appendChild(container)
const root = createRoot(container)

root.render(<BrowserRouter>
	<Layout className="container">
		<GlobalProvider>
			<Sider />
			<Layout>
				<Header />
				<Layout.Content className="content">
					<Path />
					<Router />
				</Layout.Content>
				<Footer />
			</Layout>
		</GlobalProvider>
	</Layout>
</BrowserRouter>)
