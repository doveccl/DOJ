import './index.less'
import { Code } from '../code'
import { MarkDown } from '../markdown'
import { EyeOutlined } from '@ant-design/icons'

interface EditorProps {
  value?: string
  trusted?: boolean
  onChange?: (value: string) => void
}

export function Editor(props: EditorProps) {
  return <div className="editor">
    <div className="code">
      <Code
        language="markdown"
        value={props.value}
        onChange={props.onChange}
      />
    </div>
    <div className="preview">
      <EyeOutlined className="button" />
      <div className="view">
        <MarkDown
          children={props.value}
          trusted={props.trusted}
        />
      </div>
    </div>
  </div>
}
