import React, { useContext, useState } from 'react'
import { message, Avatar, Button, Card, Divider, List, Popconfirm } from 'antd'
import { glink } from '../../../common/function'
import { delPost, putPost } from '../../model'
import { IPost } from '../../interface'
import { Editor } from '../editor'
import { MarkDown } from '../markdown'
import { GlobalContext } from '../../global'
import { diffGroup } from '../../../common/user'
import { Group } from '../../../common/interface'

interface IMut {
	onDel?: () => void,
	onChange?: (s: string) => void,
}

export function Post(post: IPost & IMut) {
	const [edit, setEdit] = useState(false)
	const [content, setContent] = useState(post.content)
	const [global] = useContext(GlobalContext)

	let mutable = global.user._id === post.uid
	mutable = mutable || diffGroup(global.user, Group.admin)

	return <List.Item key={post._id}>
		<Card
			title={<>
				<Avatar shape="square" src={glink(post.umail) }/>
				<div className="hdivider" />
				<span>{post.uname}</span>
			</>}
			extra={<>
				<span>Posted at {new Date(post.createdAt).toLocaleString()}</span>
				{mutable && <>
					<Divider type="vertical" />
					<a onClick={() => setEdit(true)}>Edit</a>
					<Divider type="vertical" />
					<Popconfirm title="Delete this post?" onConfirm={() => {
						delPost(post._id).then(post.onDel).catch(message.error)
					}}>
						<a style={{ color: 'red' }}>Delete</a>
					</Popconfirm>
				</>}
			</>}
		>
			{edit ? <>
				<Editor value={content} onChange={setContent} />
				<div className="divider" />
				<Button type="primary" onClick={() => {
					putPost(post._id, { ...post, content }).then(() => {
						post.onChange?.(content)
					}).catch(e => {
						setContent(post.content)
						message.error(e)
					})
				}}>Update</Button>
				<Divider type="vertical" />
				<Button onClick={() => setEdit(false)}>Cancel</Button>
			</> : <MarkDown children={post.content} />}
		</Card>
	</List.Item>
}
