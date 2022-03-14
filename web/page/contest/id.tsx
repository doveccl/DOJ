import moment from 'moment'
import React, { useContext, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { message, Card, Col, Progress, Row, Table } from 'antd'
import { renderType } from './index'
import { parseCount } from '../../../common/function'
import { Discuss } from '../../component/discuss'
import { LoginTip } from '../../component/login-tip'
import { MarkDown } from '../../component/markdown'
import { Scoreboard } from '../../component/scoreboard'
import { getContest, getProblems } from '../../model'
import { IContest, IProblem } from '../../interface'
import { GlobalContext } from '../../global'

export default function Contest() {
	const { id } = useParams()
	const [status, setStatus] = useState('')
	const [process, setProcess] = useState(0)
	const [tab, setTab] = useState(location.hash || '#description')
	const [problems, setProblems] = useState([] as IProblem[])
	const [contest, setContest] = useState(null as IContest)
	const [global, setGlobal] = useContext(GlobalContext)

	useEffect(() => setGlobal({
		path: [
			{ url: '/contest', desc: 'Contest' },
			contest?.title ?? id
		]
	}), [contest?.title])

	useEffect(() => {
		global.user && Promise.all([
			getContest(id),
			getProblems({ cid: id })
		]).then(([contest, { list }]) => {
			setContest(contest)
			setProblems(list)
		}).catch(message.error)
	}, [global.user])

	useEffect(() => {
		const timer = setInterval(() => {
			const now = Date.now()
			const st = +new Date(contest?.startAt ?? 0)
			const et = +new Date(contest?.endAt ?? 0)
			setProcess(100 * (now - st) / (et - st))
			setStatus(now > et ? 'Ended' : parseCount(now - st))
		}, 200)
		return () => clearInterval(timer)
	}, [])

	return <>
		<LoginTip />
		<Card
			loading={!contest}
			title={contest?.title ?? 'Contest'}
			extra={renderType(contest?.type)}
		>
			<Progress percent={process} showInfo={false} />
			<Row justify="space-between">
				<Col>{moment(contest?.startAt).format()}</Col>
				<Col>{status}</Col>
				<Col>{moment(contest?.endAt).format()}</Col>
			</Row>
		</Card>
		<div className="divider" />
		<Card
			tabList={[
				{ key: '#description', tab: 'Description' },
				{ key: '#problems', tab: 'Problems' },
				{ key: '#discuss', tab: 'Discuss' },
				{ key: '#scoreboard', tab: 'Scoreboard' }
			]}
			activeTabKey={tab}
			onTabChange={t => setTab(location.hash = t)}
		>
			{tab === '#discuss' && <Discuss topic={id} />}
			{tab === '#scoreboard' && <Scoreboard id={id} />}
			{tab === '#description' && <MarkDown trusted>{contest?.description}</MarkDown>}
			{tab === '#problems' && <Table
				rowKey="_id"
				size="middle"
				pagination={false}
				dataSource={problems}
				columns={[
					{
						width: 100, sorter: (a, b) => a.contest.key.localeCompare(b.contest.key),
						title: 'Index', dataIndex: ['contest', 'key'], defaultSortOrder: 'ascend'
					},
					{ width: 200, title: 'Title', dataIndex: 'title', render: (t, r) => <Link
						children={t} to={`/problem/${r._id}`}
					/> },
					{ width: 200, title: 'Action', dataIndex: '_id', render: (id) => <a
						children="Open in new tab" href={`/problem/${id}`} target="_blank"
					/> }
				]}
			/>}
		</Card>
	</>
}
