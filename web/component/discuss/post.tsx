import * as React from 'react'

import { message, Avatar, Button, Card, Divider, List, Popconfirm } from 'antd'

import { glink } from '../../../common/function'
import { delPost, putPost } from '../../model'
import { IPost } from '../../util/interface'
import { Editor } from '../editor'
import { MarkDown } from '../markdown'

interface PostProps {
	post: IPost
	action?: boolean
	callback?: () => any
}

export class Post extends React.Component<PostProps> {
	public state = {
		edit: false,
		text: this.props.post.content
	}
	private update = () => {
		const { post, callback } = this.props
		putPost(post._id, { content: this.state.text })
			.then(() => {
				message.success('update success')
				this.setState({ edit: false })
				if (callback) { callback() }
			})
			.catch(message.error)
	}
	private remove = () => {
		const { post, callback } = this.props
		delPost(post._id)
			.then(() => {
				message.success('delete success')
				if (callback) { callback() }
			})
			.catch(message.error)
	}
	public render() {
		const { _id, uname, umail, content, createdAt } = this.props.post
		const postAtString = new Date(createdAt).toLocaleString()
		return <List.Item key={_id}>
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
						<a onClick={() => this.setState({ edit: true })}>Edit</a>
						<Divider type="vertical" />
						<Popconfirm title="Are you sure delete this post?" onConfirm={this.remove}>
							<a style={{ color: 'red' }}>Delete</a>
						</Popconfirm>
					</React.Fragment>}
				</React.Fragment>}
			>
				{this.state.edit ? <React.Fragment>
					<Editor value={content} onChange={(text) => this.setState({ text })} />
					<div className="divider" />
					<Button type="primary" onClick={this.update}>Update</Button>
					<Divider type="vertical" />
					<Button onClick={() => this.setState({ edit: false })}>Cancel</Button>
				</React.Fragment> : <MarkDown source={content} />}
			</Card>
		</List.Item>
	}
}
