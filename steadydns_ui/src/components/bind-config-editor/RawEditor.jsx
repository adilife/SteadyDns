import { useState, useEffect } from 'react'
import {
  Card,
  Input,
  Button,
  Space,
  Alert,
  Typography
} from 'antd'
import {
  CopyOutlined,
  CheckOutlined,
  HighlightOutlined
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'


const { TextArea } = Input
const { Text } = Typography

const RawEditor = ({ content, onContentChange }) => {
  // 国际化
  const { t } = useTranslation()
  
  console.log('RawEditor组件渲染，接收到的props:', {
    content: content ? `有数据 (${content.length} 字符)` : '空',
    onContentChange: typeof onContentChange === 'function' ? '有函数' : '无'
  });
  
  const [copied, setCopied] = useState(false)
  const [lineCount, setLineCount] = useState(0)

  // 处理内容变更
  const handleContentChange = (e) => {
    const newContent = e.target.value
    onContentChange(newContent)
    updateLineCount(newContent)
  }

  // 更新行数
  const updateLineCount = (text) => {
    const count = text ? text.split('\n').length : 0
    setLineCount(count)
  }

  // 复制内容
  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(content)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (error) {
      console.error('Failed to copy content:', error)
    }
  }

  // 初始化行数
  useEffect(() => {
    updateLineCount(content)
  }, [content])

  return (
    <Card
      title={
        <Space>
          <Text strong>{t('rawEditor.rawEditMode')}</Text>
          <Text type="secondary">{lineCount} {t('rawEditor.lines')}</Text>
          <Button
            icon={copied ? <CheckOutlined /> : <CopyOutlined />}
            size="small"
            onClick={handleCopy}
          >
            {copied ? t('rawEditor.copied') : t('rawEditor.copy')}
          </Button>
        </Space>
      }
      extra={
        <Alert
          message={t('rawEditor.tip')}
          description={t('rawEditor.tipDescription')}
          type="info"
          size="small"
          showIcon
        />
      }
    >
      <TextArea
        value={content}
        onChange={handleContentChange}
        style={{
          height: 500,
          fontFamily: 'Monaco, Menlo, Consolas, "Courier New", monospace',
          fontSize: 14,
          lineHeight: 1.5
        }}
        autoSize={false}
        placeholder={t('rawEditor.enterBindConfig')}
        spellCheck={false}
        autoComplete="off"
      />
      <div style={{ marginTop: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <Space>
            <HighlightOutlined />
            <Text type="secondary">{t('rawEditor.syntaxHighlight')}</Text>
          </Space>
        </div>
        <div>
          <Text type="secondary">{t('rawEditor.saveShortcut')}</Text>
        </div>
      </div>
    </Card>
  )
}

export default RawEditor
