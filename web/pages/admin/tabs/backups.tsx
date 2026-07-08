import { CloudUploadOutlined, DeleteOutlined, DownloadOutlined } from '@ant-design/icons'
import { AutoComplete, Button, Flex, Form, InputNumber, Popconfirm, Space, Switch, Table, Tooltip, Typography } from 'antd'

import type { BackupItem, BackupList, BackupSettings } from '../../../client'
import { localeCode } from '../../../locale'
import type { Lang } from '../../../locale'
import { ErrorBlock, LoadingBlock } from '../../../components/state'
import { formatBytes } from '../../../utils/format'
import type { BackupSettingsForm } from '../types'
import type { Option } from './shared'
import { useAdminText } from './shared'

export function BackupsTab({
  settings,
  backups,
  cronOptions,
  createLoading,
  settingsSaveLoading,
  downloadName,
  deleteName,
  onSaveSettings,
  onCreate,
  onDownload,
  onDelete,
  lang
}: {
  settings: { isLoading: boolean; isError: boolean; error: unknown; data?: BackupSettings }
  backups: { isLoading: boolean; isError: boolean; error: unknown; data?: BackupList }
  cronOptions: Option<string>[]
  createLoading: boolean
  settingsSaveLoading: boolean
  downloadName?: string
  deleteName?: string
  onSaveSettings: (values: BackupSettingsForm) => void
  onCreate: () => void
  onDownload: (name: string) => void
  onDelete: (name: string) => void
  lang: Lang
}) {
  const text = useAdminText()
  const [form] = Form.useForm<BackupSettingsForm>()
  const saveSettings = (patch?: Partial<BackupSettingsForm>) => {
    if (!settings.data) {
      return
    }
    const next = { ...settings.data, ...form.getFieldsValue(true), ...patch }
    if (next.enabled === settings.data.enabled && next.cron === settings.data.cron && next.keep === settings.data.keep) {
      return
    }
    onSaveSettings(next)
  }
  return (
    <Flex vertical gap={16}>
      <Flex className="tableToolbar" justify="space-between" align="center" gap={12} wrap>
        {settings.isLoading ? <LoadingBlock /> : settings.isError ? <ErrorBlock error={settings.error} /> : settings.data ? (
          <Form<BackupSettingsForm>
            form={form}
            className="tableToolbarForm"
            layout="inline"
            initialValues={settings.data}
            key={`${settings.data.enabled}:${settings.data.cron}:${settings.data.keep}`}
          >
            <Form.Item name="enabled" label={text.admin.backupEnabled} valuePropName="checked">
              <Switch loading={settingsSaveLoading} onChange={(enabled) => saveSettings({ enabled })} />
            </Form.Item>
            <Form.Item name="cron">
              <AutoComplete options={cronOptions} placeholder={text.admin.backupCron} disabled={settingsSaveLoading} style={{ width: 220 }} onBlur={() => saveSettings()} />
            </Form.Item>
            <Form.Item name="keep">
              <InputNumber min={1} max={100} prefix={text.admin.backupKeep} disabled={settingsSaveLoading} style={{ width: 180 }} onBlur={() => saveSettings()} />
            </Form.Item>
          </Form>
        ) : null}
        <Button type="primary" icon={<CloudUploadOutlined />} loading={createLoading || !!backups.data?.running} onClick={onCreate}>
          {text.admin.backupNow}
        </Button>
      </Flex>
      {backups.isLoading ? <LoadingBlock /> : backups.isError ? <ErrorBlock error={backups.error} /> : (
        <Table<BackupItem>
          rowKey="name"
          pagination={{ defaultPageSize: 10, hideOnSinglePage: true }}
          scroll={{ x: 760 }}
          dataSource={backups.data?.items ?? []}
          columns={[
            { title: text.admin.backupFile, dataIndex: 'name', width: 360, render: (name: string) => <Typography.Text ellipsis={{ tooltip: name }}>{name}</Typography.Text> },
            { title: text.admin.createdAt, dataIndex: 'createdAt', render: (value: string) => new Date(value).toLocaleString(localeCode(lang)) },
            { title: text.admin.backupSize, dataIndex: 'size', render: (value: number) => formatBytes(value) },
            {
              align: 'right',
              render: (_, row) => (
                <Space size={4}>
                  <Tooltip title={text.common.download}>
                    <Button type="text" icon={<DownloadOutlined />} loading={downloadName === row.name} onClick={() => onDownload(row.name)} />
                  </Tooltip>
                  <Popconfirm title={text.common.confirmDelete} okText={text.common.delete} cancelText={text.common.cancel} onConfirm={() => onDelete(row.name)}>
                    <Button type="text" danger icon={<DeleteOutlined />} loading={deleteName === row.name} />
                  </Popconfirm>
                </Space>
              )
            }
          ]}
        />
      )}
    </Flex>
  )
}
