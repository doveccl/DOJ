import React, { useContext, useEffect, useRef, useState } from 'react'
import { message, Button, Card, Checkbox, Divider, Tag, Timeline } from 'antd'
import { Link, useParams } from 'react-router-dom'
import { Manager, Socket } from 'socket.io-client'

import { parseMemory, parseTime } from '../../../common/function'
import { Group, IResult, Status } from '../../../common/interface'
import { diffGroup } from '../../../common/user'
import { Code } from '../../component/code'
import { LoginTip } from '../../component/login-tip'
import { getSubmission, getToken, putSubmission, rejudgeSubmission } from '../../model'
import { ISubmission } from '../../util/interface'
import { GlobalContext } from '../../global'
import { renderStatus } from './index'

const color = (r?: IResult) => r.status === Status.AC ? 'green' : 'red'

export default function Submission() {
	const { id } = useParams()
	const socket = useRef<Socket>(null)
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
		if (s?.result.status === Status.WAIT) {
			setPending('Pending ...')
			socket.current = new Manager().socket('/client').on('connect', () => {
				socket.current.emit('register', { id, token: getToken() })
			}).on('setp', ({ pending, cases }) => {
				setPending(pending)
				setSubmission({ ...s, cases })
			}).on('result', ({ result, cases }) => {
				setPending(null)
				setSubmission({ ...s, result, cases })
				socket.current.close()
			})
			return () => { socket.current.close() }
		} else {
			setPending(null)
		}
	}, [s?.result.status])

	function rejudge() {
		rejudgeSubmission({ id }).then(() => setSubmission({
			...s,
			cases: [],
			result: {
				time: 0,
				memory: 0,
				status: Status.WAIT
			}
		})).catch(message.error)
	}

	let cases = s?.cases.length ? s.cases : [s?.result]
	if (s?.result.status === Status.WAIT) cases = null
	if (!s?.result) cases = null

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
					[{new Date(s?.createdAt).toLocaleString()}] {s?.uname} submitted the code
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
