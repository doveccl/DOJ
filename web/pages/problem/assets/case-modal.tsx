import { Form, Input, Modal } from 'antd'

import { useLocale } from '../../../locale'

export type CaseForm = { name: string; input: string; output: string }

export function CaseModal({
  loading,
  onCancel,
  onSave
}: {
  loading: boolean
  onCancel: () => void
  onSave: (values: CaseForm) => void
}) {
  const { text } = useLocale()
  const [form] = Form.useForm<CaseForm>()

  return (
    <Modal
      open
      destroyOnHidden
      title={text.problem.addCase}
      okText={text.common.save}
      cancelText={text.common.cancel}
      confirmLoading={loading}
      onCancel={onCancel}
      onOk={() => form.submit()}
    >
      <Form form={form} preserve={false} layout="vertical" initialValues={{ name: '', input: '', output: '' }} onFinish={onSave}>
        <Form.Item name="name" label={text.problem.caseName}>
          <Input placeholder={text.problem.caseNamePlaceholder} />
        </Form.Item>
        <Form.Item name="input" label={`${text.problem.caseName}.in`}>
          <Input.TextArea rows={5} />
        </Form.Item>
        <Form.Item name="output" label={`${text.problem.caseName}.out`}>
          <Input.TextArea rows={5} />
        </Form.Item>
      </Form>
    </Modal>
  )
}
