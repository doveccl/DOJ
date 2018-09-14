import * as React from 'react'
import * as ReactDOM from 'react-dom'
import { BrowserRouter } from 'react-router-dom'

import { Layout } from 'antd'

import Footer from './component/footer'
import Header from './component/header'
import Path from './component/path'
import Sider from './component/sider'
import Router from './router'

import './index.less'
import './util/init'

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
