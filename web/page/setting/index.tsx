import React, { useContext, useEffect } from 'react'
import { Card } from 'antd'
import { GlobalContext } from '../../global'
import { SettingForm } from '../../component/form/setting'

export default function Setting() {
	const [global, setGlobal] = useContext(GlobalContext)
	useEffect(() => setGlobal({ path: ['Setting'] }), [])

	return <React.Fragment>
		<Card title="Setting" loading={!global.user}>
			<SettingForm user={global.user} onUpdate={user => setGlobal({ user })} />
		</Card>
	</React.Fragment>
}
