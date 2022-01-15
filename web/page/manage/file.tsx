import moment from 'moment'
import React, { useContext, useEffect, useState } from 'react'
import { message, Card, Divider, Popconfirm, Table, Upload, TablePaginationConfig } from 'antd'
import { LoadingOutlined, CloudUploadOutlined } from '@ant-design/icons'
import { UploadChangeParam } from 'antd/lib/upload'
import { delFile, getFiles, putFile } from '../../model'
import { IFile } from '../../interface'
import { GlobalContext } from '../../global'
import { parseMemory } from '../../../common/function'

const renderUsage = (f: IFile) => {
	if (/\.pdf$/.test(f.filename))
		return `[[ PDF id="${f._id}" ]]`
	else if (/\.(png|jpe?g|gif)$/.test(f.filename))
		return `[[ IMG id="${f._id}" ]]`
	return `${f._id}`
}

const defaultPage: TablePaginationConfig = {
	total: 0,
	current: 1,
	pageSize: 50
}

export default function ManageFile() {
	const [global, setGlobal] = useContext(GlobalContext)
	useEffect(() => setGlobal({ path: ['Manage', 'Problem'] }), [])

	const [loading, setLoading] = useState(true)
	const [uploading, setUploading] = useState(false)
	const [files, setFiles] = useState([] as IFile[])
	const [pagination, setPagination] = useState(defaultPage)

	const { current, pageSize } = pagination
	useEffect(() => {
		if (global.user) {
			setLoading(true)
			getFiles({
				page: current,
				size: pageSize
			}).then(({ total, list }) => {
				setFiles(list)
				setPagination({ ...pagination, total })
			}).catch(e => {
				message.error(e)
			}).finally(() => {
				setLoading(false)
			})
		}
	}, [global.user, current, pageSize])

	function onUploadChange({ file }: UploadChangeParam) {
		console.log(file.status)
		setUploading(file.status === 'uploading')
		if (file.status === 'error')
			message.error('upload fail')
		else if (file.response?.success)
			setFiles([{
				_id: file.response.data[0],
				filename: file.name,
				length: file.size,
				uploadDate: moment().format()
			}, ...files])
	}

	return <Card title="Files">
		<Upload.Dragger
			action="/api/file"
			accept="image/*,.zip,.pdf"
			onChange={onUploadChange}
			multiple={false}
		>
			<p className="ant-upload-drag-icon">
				{uploading ? <LoadingOutlined /> : <CloudUploadOutlined />}
			</p>
			<p className="ant-upload-text">Click or drag file to this area to upload</p>
			<p className="ant-upload-hint">PDF, ZIP, Images are supported</p>
		</Upload.Dragger>
		<div className="divider" />
		<Table
			rowKey="_id"
			size="middle"
			loading={loading}
			dataSource={files}
			pagination={pagination}
			onChange={setPagination}
			columns={[
				{ title: 'Filename', dataIndex: 'filename', render: (t, r) => (
					<a href={`/api/file/${r._id}`} download={t}>{t}</a>
				) },
				{ title: 'Usage', key: 'usage', render: (_, r) => renderUsage(r) },
				{ title: 'Size', dataIndex: 'length', render: t => parseMemory(t) },
				{ title: 'Time', dataIndex: 'uploadDate', render: t => moment(t).fromNow() },
				{ title: 'File Type', dataIndex: ['metadata', 'type'] },
				{ title: 'Action', key: 'action', render: (_, r) => <>
					<a onClick={() => {
						const filename = prompt('New name:', r.filename)
						putFile(r._id, { filename }).then(() => {
							setFiles(files.map(f => f._id === r._id ? { ...f, filename } : f))
						}).catch(message.error)
					}}>Rename</a>
					<Divider type="vertical" />
					<Popconfirm title="Delete this file?" onConfirm={() => {
						delFile(r._id).then(() => {
							setFiles(files.filter(f => f._id !== r._id))
						}).catch(message.error)
					}}>
						<a style={{ color: 'red' }}>Delete</a>
					</Popconfirm>
				</> }
			]}
		/>
	</Card>

}
