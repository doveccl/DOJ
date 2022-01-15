import React, { useContext, useEffect, useState } from 'react'
import { message, Button, Card, Divider } from 'antd'
import { Editor } from '../../component/editor'
import { getConfig, putConfig } from '../../model'
import { GlobalContext } from '../../global'

export default function Manage() {
	const [value, setValue] = useState(null as string)
	const [global, setGlobal] = useContext(GlobalContext)
	useEffect(() => setGlobal({ path: ['Manage', 'Setting'] }), [])

	const reload = () => {
		getConfig('notification')
			.then(d => setValue(d.value))
			.catch(message.error)
	}
	const update = () => {
		putConfig('notification', { value })
			.then(() => message.success('update success'))
			.catch(message.error)
	}

	useEffect(() => global.user && reload(), [global.user])

	return <>
		<Card
			title="Set Notification"
			loading={value === null}
			extra={<>
				<Button onClick={reload}>Reset</Button>
				<Divider type="vertical" />
				<Button type="primary" onClick={update}>Update</Button>
			</>}
		>
			<Editor trusted value={value} onChange={setValue} />
		</Card>
	</>
}
