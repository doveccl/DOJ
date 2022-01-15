import './index.less'
import React, { useContext } from 'react'
import { Link } from 'react-router-dom'
import { Breadcrumb } from 'antd'
import { GlobalContext } from '../../global'

export function Path() {
	const [global] = useContext(GlobalContext)
	return <Breadcrumb className="breadcrumb">
		<Breadcrumb.Item>DOJ</Breadcrumb.Item>
		{global.path?.map((i, k) => <Breadcrumb.Item key={k}>{
			typeof i === 'string' ? i : <Link to={i.url}>{i.desc}</Link>
		}</Breadcrumb.Item>)}
	</Breadcrumb>
}
