import './index.less'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { Layout } from 'antd'
import { GlobalProvider } from './global'
import { Path } from './component/path'
import { Footer, Header, Sider } from './layout'
import { Router } from './router'

createRoot(document.getElementById('root')!).render(<BrowserRouter>
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
</BrowserRouter>)
