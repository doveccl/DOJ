import * as React from 'react'
import * as ReactDOM from 'react-dom'

import { Layout } from 'antd'
import { BrowserRouter } from 'react-router-dom'

import Router from './router'
import Path from './component/path'
import Sider from './component/sider'
import Header from './component/header'
import Footer from './component/footer'

import './util/init'
import './index.less'

const app = <BrowserRouter>
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
</BrowserRouter>

const container = document.createElement('div')
document.body.appendChild(container)
ReactDOM.render(app, container)
