import React, { useContext, useEffect, useState } from 'react'
import { message, Button, Card, List } from 'antd'

import { Group } from '../../../common/interface'
import { diffGroup } from '../../../common/user'
import { getPosts, postPost } from '../../model'
import { IPost } from '../../util/interface'
import { GlobalContext } from '../../global'
import { Editor } from '../editor'
import { Post } from './post'

export function Discuss({ topic = '' }) {
	const [flag, update] = useState({})
	const [page, setPage] = useState(1)
	const [size, setSize] = useState(5)
	const [total, setTotal] = useState(0)
	const [content, setContent] = useState('')
	const [posts, setPosts] = useState(null as IPost[])
	const [global] = useContext(GlobalContext)

	useEffect(() => {
		global.user && getPosts({ topic, page, size }).then(({ total, list }) => {
			setTotal(total)
			setPosts(list)
		}).catch(message.error)
	}, [global.user, topic, page, size, flag])

	return <>
		<List
			loading={!posts}
			dataSource={posts ?? []}
			grid={{ column: 1 }}
			pagination={{
				total,
				current: page,
				pageSize: size,
				hideOnSinglePage: true,
				onChange: (page, size) => {
					setPage(page)
					setSize(size)
				}
			}}
			renderItem={p => <Post
				post={p} callback={() => update({})}
				action={diffGroup(global.user, Group.admin) || global.user._id === p.uid}
			/>}
		/>
		<div className="divider" />
		<Card type="inner" title="Create a new post" actions={[
			<Button type="primary" onClick={() => {
				postPost({ topic, content }).then(() => {
					update({})
					message.success('post success')
				}).catch(message.error)
			}}>Post</Button>
		]}>
			<Editor onChange={setContent} />
		</Card>
	</>
}
