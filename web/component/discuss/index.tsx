import * as React from 'react'

import { message, Button, Card, List } from 'antd'

import { Group } from '../../../common/interface'
import { diffGroup } from '../../../common/user'
import { getPosts, postPost } from '../../model'
import { IPost } from '../../util/interface'
import { globalState } from '../../util/state'
import { Editor } from '../editor'
import { Post } from './post'

interface DiscussProps {
	topic: string
}

export class Discuss extends React.Component<DiscussProps> {
	public state = {
		total: 0,
		current: 1,
		pageSize: 5,
		loading: true,
		content: '',
		posts: [] as IPost[]
	}
	private handleChange = (page?: number, pageSize?: number) => {
		this.setState({ loading: true })
		const p = page || this.state.current
		const s = pageSize || this.state.pageSize
		getPosts({ page: p, size: s, topic: this.props.topic })
			.then(({ total, list }) => {
				this.setState({
					total, current: p, pageSize: s,
					posts: list, loading: false
				})
			})
			.catch((err) => {
				message.error(err)
				this.setState({ loading: false })
			})
	}
	private handleClick = () => {
		const { topic } = this.props
		const { content } = this.state
		postPost({ topic, content })
			.then(() => {
				message.success('post success')
				this.handleChange(1)
			})
			.catch(message.error)
	}
	public componentWillMount() {
		this.handleChange()
	}
	public render() {
		const uid = globalState.user._id
		const isAdmin = diffGroup(globalState.user, Group.admin)
		const { total, current, pageSize } = this.state

		return <React.Fragment>
			<List
				grid={{ column: 1 }}
				loading={this.state.loading}
				dataSource={this.state.posts}
				pagination={{
					total, current, pageSize,
					hideOnSinglePage: true,
					onChange: this.handleChange
				}}
				renderItem={(p: IPost) => <Post
					post={p} callback={this.handleChange}
					action={isAdmin || p.uid === uid}
				/>}
			/>
			<div className="divider" />
			<Card type="inner" title="Creat a new post" actions={[
				<Button type="primary" onClick={this.handleClick}>Post</Button>
			]}>
				<Editor onChange={(content) => this.setState({ content })} />
			</Card>
		</React.Fragment>
	}
}
