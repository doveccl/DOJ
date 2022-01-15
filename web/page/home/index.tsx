import React, { useContext, useEffect, useState } from 'react'
import { message, Card } from 'antd'
import { LoginTip } from '../../component/login-tip'
import { MarkDown } from '../../component/markdown'
import { getConfig } from '../../model'
import { GlobalContext } from '../../global'

export default function Home() {
	const [value, setValue] = useState(null as string)
	const [global, setGlobal] = useContext(GlobalContext)
	useEffect(() => setGlobal({ path: ['Home'] }), [])
	useEffect(() => {
		global.user && getConfig('notification')
			.then(d => setValue(d.value))
			.catch(message.error)
	}, [global.user])

	return <>
		<LoginTip />
		<Card title="Welcome to DOJ" loading={value === null}>
			<MarkDown children={value} trusted />
		</Card>
	</>
}
