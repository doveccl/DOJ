import * as React from 'react'

import { Avatar, Card, Divider, List } from 'antd'

import { glink } from '../../util/function'
import { IPost } from '../../util/interface'
import Markdown from '../markdown'

interface PostProps {
	post: IPost
	action: boolean
}

export default class extends React.Component<PostProps> {
	public state = {
		content: this.props.post.content,
		content2: this.props.post.content
	}
	private edit = () => {
		console.log('edit')
	}
	private remove = () => {
		console.log('remove')
	}
	public render() {
		const { uname, umail, createdAt } = this.props.post
		const postAtString = new Date(createdAt).toLocaleString()
		return <List.Item>
			<Card
				title={<span>
					<Avatar shape="square" src={glink(umail) }/>
					<div className="hdivider" />
					<span>{uname}</span>
				</span>}
				extra={<React.Fragment>
					<span>Posted at {postAtString}</span>
					{this.props.action && <React.Fragment>
						<Divider type="vertical" />
						<a onClick={this.edit}>Edit</a>
						<Divider type="vertical" />
						<a style={{ color: 'red' }} onClick={this.remove}>Delete</a>
					</React.Fragment>}
				</React.Fragment>}
			>
				<Markdown source={this.state.content} />
			</Card>
		</List.Item>
	}
}
