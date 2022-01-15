import React, { useContext, useEffect, useState } from 'react'
import { message, Button, Card, List } from 'antd'
import { PaginationConfig } from 'antd/lib/pagination'
import { getPosts, postPost } from '../../model'
import { IPost } from '../../interface'
import { GlobalContext } from '../../global'
import { Editor } from '../editor'
import { Post } from './post'

const defaultPage: PaginationConfig = {
	total: 0,
	current: 1,
	pageSize: 5,
	hideOnSinglePage: true
}

export function Discuss({ topic = '' }) {
	const [global] = useContext(GlobalContext)
	const [flag, update] = useState({})
	const [content, setContent] = useState('')
	const [loading, setLoading] = useState(false)
	const [posts, setPosts] = useState([] as IPost[])
	const [pagination, setPagination] = useState<PaginationConfig>({
		...defaultPage,
		onChange(page, size) {
			setPagination({
				...pagination,
				current: page,
				pageSize: size
			})
		}
	})

	const { current, pageSize } = pagination
	useEffect(() => {
		if (global.user) {
			setLoading(true)
			getPosts({
				topic,
				page: current,
				size: pageSize
			}).then(({ total, list }) => {
				setPosts(list)
				setPagination({ ...pagination, total })
			}).catch(e => {
				message.error(e)
			}).finally(() => {
				setLoading(false)
			})
		}
	}, [global.user, topic, current, pageSize, flag])

	return <>
		<List
			loading={loading}
			dataSource={posts}
			grid={{ column: 1 }}
			pagination={pagination}
			renderItem={post => <Post
				{...post}
				onDel={() => update({})}
				onChange={_ => update({})}
			/>}
		/>
		<div className="divider" />
		<Card type="inner" title="Create a new post" extra={
			<Button type="primary" onClick={() => {
				postPost({ topic, content }).then(_ => {
					update({})
					message.success('post success')
				}).catch(message.error)
			}}>Post now</Button>
		}>
			<Editor value={content} onChange={setContent} />
		</Card>
	</>
}
