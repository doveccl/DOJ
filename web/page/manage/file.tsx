import * as React from 'react'

import { message, Button, Card, Divider, Input, Modal, Popconfirm, Table, Tag } from 'antd'
import { WrappedFormUtils } from 'antd/lib/form/Form'

import WrappedUserForm from '../../component/form/user'
import { delFile, getFiles, hasToken, putFile } from '../../model'
import { IFile } from '../../util/interface'
import { updateState } from '../../util/state'

export default class extends React.Component {
	private form: WrappedFormUtils = undefined
	public state = {
		loading: true,
		files: [] as IFile[],
		pagination: { current: 1, pageSize: 50, total: 0 }
	}
	private handleChange = (pagination = this.state.pagination) => {
		const pager = { ...this.state.pagination }
		pager.current = pagination.current
		this.setState({ loading: true, pagination: pager })
		const { pageSize: size, current: page } = pager
		getFiles({ page, size })
			.then(({ total, list: files }) => {
				this.state.pagination.total = total
				this.setState({
					pagination: this.state.pagination,
					loading: false, files
				})
			})
			.catch((err) => {
				message.error(err)
				this.setState({ loading: false })
			})
	}
	private del = (id: any) => {
		delFile(id)
			.then(() => this.handleChange())
			.catch(message.error)
	}
	public componentWillMount() {
		updateState({ path: [ 'Manage', 'File' ] })
		if (hasToken()) { this.handleChange() }
	}
	public render() {
		return <Card title="Files">
			<Table
				rowKey="_id"
				size="middle"
				loading={this.state.loading}
				dataSource={this.state.files}
				pagination={this.state.pagination}
				onChange={this.handleChange}
				columns={[
					{ title: 'ID', dataIndex: '_id' },
					{ title: 'Filename', dataIndex: 'filename' },
					{ title: 'Type', dataIndex: 'metadata.type' },
					{ title: 'Action', key: 'action', render: (t, r) => <React.Fragment>
						<Popconfirm title="Delete this file?" onConfirm={() => this.del(r._id)}>
							<a style={{ color: 'red' }}>Delete</a>
						</Popconfirm>
					</React.Fragment> }
				]}
			/>
		</Card>
	}
}
