import React from 'react'

import { message, Card, Divider, Popconfirm, Table, Upload } from 'antd'
import { LoadingOutlined, CloudUploadOutlined } from '@ant-design/icons'
import { UploadChangeParam } from 'antd/lib/upload'

import { delFile, getFiles, hasToken, putFile } from '../../model'
import { IFile } from '../../util/interface'
import { updateState } from '../../util/state'

const renderUsage = (f: IFile) => {
	if (/\.pdf$/.test(f.filename))
		return `[[ PDF id="${f._id}" ]]`
	else if (/\.(png|jpe?g|gif)$/.test(f.filename))
		return `[[ IMG id="${f._id}" ]]`
	return `${f._id}`
}

export default class extends React.Component {
	public state = {
		loading: true,
		uploading: false,
		files: [] as IFile[],
		pagination: { current: 1, pageSize: 50, total: 0 }
	}
	private handleChange = (pagination = this.state.pagination) => {
		const pager = { ...this.state.pagination }
		pager.current = pagination.current
		pager.pageSize = pagination.pageSize
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
	private onUploadChange = ({ file }: UploadChangeParam) => {
		const { status, response: r } = file
		if (status === 'uploading') {
			this.setState({ uploading: true })
		} else if (status === 'removed') {
			this.setState({ uploading: false })
		} else { // status is 'done' or 'error'
			this.setState({ uploading: false })
			if (r.success) {
				this.handleChange()
			} else {
				message.error(r.message || r)
			}
		}
	}
	private rename = (id: any, filename: string) => {
		if (!filename) { return }
		putFile(id, { filename })
			.then(() => this.handleChange())
			.catch(message.error)
	}
	private del = (id: any) => {
		delFile(id)
			.then(() => this.handleChange())
			.catch(message.error)
	}
	public componentDidMount() {
		updateState({ path: [ 'Manage', 'File' ] })
		if (hasToken()) { this.handleChange() }
	}
	public render() {
		return <Card title="Files">
			<Upload.Dragger
				action="/api/file"
				accept="image/*,.zip,.pdf"
				onChange={this.onUploadChange}
				multiple={false}
			>
				<p className="ant-upload-drag-icon">
					{this.state.uploading ? <LoadingOutlined /> : <CloudUploadOutlined />}
				</p>
				<p className="ant-upload-text">
					Click or drag file to this area to upload
				</p>
				<p className="ant-upload-hint">
					PDF, ZIP, Images are supported
				</p>
			</Upload.Dragger>
			<div className="divider" />
			<Table
				rowKey="_id"
				size="middle"
				loading={this.state.loading}
				dataSource={this.state.files}
				pagination={this.state.pagination}
				onChange={this.handleChange}
				columns={[
					{ title: 'Filename', dataIndex: 'filename', render: (t, r) => (
						<a href={`/api/file/${r._id}`} download={t}>{t}</a>
					) },
					{ title: 'Usage', key: 'usage', render: (t, r) => renderUsage(r) },
					{ title: 'File Type', dataIndex: ['metadata', 'type'] },
					{ title: 'Action', key: 'action', render: (t, r) => <React.Fragment>
						<a onClick={() => {
							this.rename(r._id, prompt('New name:', r.filename))
						}}>Rename</a>
						<Divider type="vertical" />
						<Popconfirm title="Delete this file?" onConfirm={() => this.del(r._id)}>
							<a style={{ color: 'red' }}>Delete</a>
						</Popconfirm>
					</React.Fragment> }
				]}
			/>
		</Card>
	}
}
