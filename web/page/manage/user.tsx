import * as React from 'react'

import { message, Button, Card, Divider, Input, Modal, Popconfirm, Table, Tag } from 'antd'
import { WrappedFormUtils } from 'antd/lib/form/Form'

import WrappedUserForm from '../../component/form/user'
import { delUser, getUsers, hasToken, inviteUser, postUser, putUser } from '../../model'
import { IUser, UserGroup } from '../../util/interface'
import { updateState } from '../../util/state'

const renderGroup = (g: UserGroup) => {
	switch (g) {
		case UserGroup.admin: return <Tag>Admin</Tag>
		case UserGroup.root: return <Tag>Root</Tag>
		default: return <Tag>Common</Tag>
	}
}

export default class extends React.Component {
	private form: WrappedFormUtils = undefined
	public state = {
		loading: true,
		users: [] as IUser[],
		pagination: { current: 1, pageSize: 50, total: 0 },
		modalTitle: '',
		modalOpen: false,
		modalUser: undefined as IUser
	}
	private handleChange = (pagination = this.state.pagination) => {
		const pager = { ...this.state.pagination }
		pager.current = pagination.current
		this.setState({ loading: true, pagination: pager })
		const { pageSize: size, current: page } = pager
		getUsers({ page, size })
			.then(({ total, list: users }) => {
				this.state.pagination.total = total
				this.setState({
					pagination: this.state.pagination,
					loading: false, users
				})
			})
			.catch((err) => {
				message.error(err)
				this.setState({ loading: false })
			})
	}
	private openModal = (user?: IUser) => {
		this.setState({
			modalOpen: true,
			modalTitle: user ? 'Edit' : 'Create',
			modalUser: user
		})
	}
	private ok = () => {
		this.form.validateFields((error, values) => {
			if (!error) {
				this.setState({ loading: true })
				if (this.state.modalUser) {
					putUser(this.state.modalUser._id, values)
						.then(() => {
							message.success('update success')
							this.setState({ modalOpen: false })
							this.handleChange()
						})
						.catch((err) => {
							message.error(err)
							this.setState({ loading: false })
						})
				} else {
					postUser(values)
						.then(() => {
							message.success('create success')
							this.setState({ modalOpen: false })
							this.handleChange()
						})
						.catch((err) => {
							message.error(err)
							this.setState({ loading: false })
						})
				}
			}
		})
	}
	private invite = (mail: string) => {
		if (mail) {
			const hide: any = message.loading('Sending ...', 0)
			inviteUser({ mail })
				.then(({ mail: m }) => {
					hide()
					message.success(`code has been send to: ${m}`, 10)
				})
				.catch((err) => {
					hide()
					message.error(err)
				})
		}
	}
	private del = (id: any) => {
		delUser(id)
			.then(() => this.handleChange())
			.catch(message.error)
	}
	public componentWillMount() {
		updateState({ path: [ 'Manage', 'User' ] })
		if (hasToken()) { this.handleChange(this.state.pagination) }
	}
	public render() {
		return <Card
			title="Users"
			extra={<React.Fragment>
				<Button
					size="small" onClick={() => this.openModal()}
				>
					Create User
				</Button>
				<Divider type="vertical" />
				<Button
					size="small" type="primary"
					onClick={() => this.invite(prompt('Send invitation mail to:'))}
				>
					Invite User
				</Button>
			</React.Fragment>}
		>
			<Modal
				style={{ minWidth: 650 }}
				destroyOnClose={true}
				visible={this.state.modalOpen}
				title={this.state.modalTitle}
				confirmLoading={this.state.loading}
				onOk={() => this.ok()}
				onCancel={() => this.setState({ modalOpen: false })}
			>
				<WrappedUserForm
					value={this.state.modalUser}
					wrappedComponentRef={(w: any) => {
						this.form = w && w.props.form
					}}
				/>
			</Modal>
			<Table
				rowKey="_id"
				size="middle"
				loading={this.state.loading}
				dataSource={this.state.users}
				pagination={this.state.pagination}
				onChange={this.handleChange}
				columns={[
					{ title: 'ID', dataIndex: '_id' },
					{ title: 'Name', dataIndex: 'name' },
					{ title: 'Mail', dataIndex: 'mail' },
					{ title: 'Group', dataIndex: 'group', render: renderGroup },
					{ title: 'Join', dataIndex: 'createdAt', render: (t) => new Date(t).toLocaleString() },
					{ title: 'Action', key: 'action', render: (t, r) => <React.Fragment>
						<a onClick={() => this.openModal(r)}>Edit</a>
						<Divider type="vertical" />
						<Popconfirm title="Delete this user?" onConfirm={() => this.del(r._id)}>
							<a style={{ color: 'red' }}>Delete</a>
						</Popconfirm>
					</React.Fragment> }
				]}
			/>
		</Card>
	}
}
