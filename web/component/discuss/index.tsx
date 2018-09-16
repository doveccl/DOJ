import * as React from 'react'

import { message, List } from 'antd'

import { getPosts } from '../../model'
import { isGroup } from '../../util/function'
import { IPost } from '../../util/interface'
import { globalState } from '../../util/state'
import Post from './post'

interface DiscussProps {
	topic: string
}

export default class extends React.Component<DiscussProps> {
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
	public componentWillMount() {
		this.handleChange()
	}
	public render() {
		const uid = globalState.user._id
		const isAdmin = isGroup(globalState.user, 'admin')
		const { total, current, pageSize } = this.state

		return <React.Fragment>
			<List
				grid={{ column: 1 }}
				loading={this.state.loading}
				dataSource={this.state.posts}
				pagination={{ total, current, pageSize, onChange: this.handleChange }}
				renderItem={(p: IPost) => <Post post={p} action={isAdmin || p.uid === uid} />}
			/>
		</React.Fragment>
	}
}
