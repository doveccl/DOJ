import moment from 'moment'
import React, { useContext, useEffect, useRef, useState } from 'react'
import { message, Button, Card, Checkbox, Divider, Tag, Timeline } from 'antd'
import { Link, useParams } from 'react-router-dom'
import { parseMemory, parseTime } from '../../../common/function'
import { Group, IResult, Status } from '../../../common/interface'
import { diffGroup } from '../../../common/user'
import { Code } from '../../component/code'
import { LoginTip } from '../../component/login-tip'
import { getSubmission, getToken, putSubmission, rejudgeSubmission } from '../../model'
import { ISubmission } from '../../interface'
import { GlobalContext } from '../../global'
import { renderStatus } from './index'

const color = (r?: IResult) => r.status === Status.AC ? 'green' : 'red'

export default function Submission() {
	const { id } = useParams()
	const socket = useRef<WebSocket>(null)
	const [global, setGlobal] = useContext(GlobalContext)
	useEffect(() => setGlobal({ path: [{ url: '/submission', desc: 'Submission' }, id] }), [])

	const [code, setCode] = useState('')
	const [edit, setEdit] = useState(false)
	const [pending, setPending] = useState(null as string)
	const [s, setSubmission] = useState(null as ISubmission)

	useEffect(() => {
		global.user && getSubmission(id)
			.then(setSubmission)
			.catch(message.error)
	}, [global.user, id])

	useEffect(() => setCode(s?.code), [s?.code])

	useEffect(() => {
		if (s?.result?.status === Status.WAIT) {
			setPending('Pending ...')
			const host = location.origin.replace(/^http/, 'ws')
			const ws = socket.current = new WebSocket(`${host}/wss?client`)
			ws.onopen = () => ws.send(JSON.stringify({ id, token: getToken() }))
			ws.onmessage = ({ data }) => {
				const { pending, cases, result } = JSON.parse(data)
				if (pending) {
					setPending(pending)
					setSubmission({ ...s, cases })
				} else {
					setPending(null)
					setSubmission({ ...s, result, cases })
					ws.close()
				}
			}
			return () => ws.close()
		} else {
			setPending(null)
		}
	}, [s?.result?.status])

	function rejudge() {
		rejudgeSubmission({ _id: id }).catch(message.error)
		setSubmission({
			...s,
			cases: [],
			result: {
				time: 0,
				memory: 0,
				status: Status.WAIT
			}
		})
	}

	let cases = s?.cases.length ? s.cases : null
	if (s?.result?.status > Status.WAIT)
		cases = cases ?? [s?.result]

	return <>
		<LoginTip />
		<Card
			loading={!s}
			title={`Submission by ${s?.uname || 'unknown user'}`}
			extra={<>
				<Link to={`/problem/${s?.pid}`}>{s?.ptitle}</Link>
				{!pending && diffGroup(global.user, Group.admin) && <>
					<Divider type="vertical" />
					<Button type="primary" children="Rejudge" onClick={rejudge} />
				</>}
			</>}
		>
			<Timeline className="result" pending={pending}>
				<Timeline.Item color="green">
					{s?.uname} submitted {moment(s?.createdAt).format()}
				</Timeline.Item>
				{cases?.map((c, i) => (
					<Timeline.Item key={i} color={color(c)}>
						<Tag>#{i}</Tag>
						{renderStatus(c)}
						<Tag>{parseTime(c.time)}</Tag>
						<Tag>{parseMemory(c.memory)}</Tag>
						{c.extra && (/.\n./.test(c.extra) ?
							<pre>{c.extra.trim()}</pre> :
							<span>{c.extra}</span>
						)}
					</Timeline.Item>
				))}
			</Timeline>
		</Card>
		{s?.code && <>
			<div className="divider" />
			<Card
				title="Code"
				extra={<>
					{edit || <Checkbox
						checked={s.open}
						children="Public"
						disabled={global.user._id !== s.uid && !diffGroup(global.user, Group.admin)}
						onChange={v => {
							const open = v.target.checked
							setSubmission({ ...s, open })
							putSubmission(id, { open })
						}}
					/>}
					{diffGroup(global.user, Group.root) && <>
						{edit && <Button onClick={() => {
							setEdit(false)
							setCode(s.code)
						}}>Cancel</Button>}
						<Divider type="vertical" />
						<Button
							type={edit ? 'primary' : 'default'}
							children={edit ? 'Update' : 'Edit'}
							onClick={() => {
								setEdit(!edit)
								edit && putSubmission(id, { code }).then(() => {
									s.code = code // no effect
									rejudge()
								}).catch(e => {
									setCode(s.code)
									message.error(e)
								})
							}}
						/>
					</>}
				</>}
			>
				<Code
					value={code}
					static={!edit}
					onChange={setCode}
					language={global.languages?.[s.language].suffix}
				/>
			</Card>
		</>}
	</>
}
