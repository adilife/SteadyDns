import { useState } from 'react'
import {
  Card,
  List,
  Typography,
  Tag,
  Space,
  Divider,
  Button,
  Alert,
  Collapse
} from 'antd'
import {
  DiffOutlined,
  PlusOutlined,
  MinusOutlined,
  EditOutlined,
  LineChartOutlined,
  DownOutlined,
  UpOutlined,
  CheckOutlined
} from '@ant-design/icons'
import { t } from '../../i18n'


const { Text, Paragraph, Link } = Typography
const { Panel } = Collapse

const DiffViewer = ({ diff }) => {
  const [expandedSections, setExpandedSections] = useState([])

  if (!diff) {
    return (
      <div style={{ padding: 16 }}>
        <Alert
          message={t('diffViewer.noDiffResult')}
          description={t('diffViewer.pleaseViewDiffFirst')}
          type="info"
          showIcon
        />
      </div>
    )
  }

  // 处理 API 返回的数据结构
  let changes = []
  let added = 0
  let removed = 0
  let modified = 0
  let summary = ''

  // 检查是否是 API 返回的格式（包含 lines 和 stats）
  if (diff.lines && diff.stats) {
    // 从 API 返回的数据中提取统计信息
    added = diff.stats.added || 0
    removed = diff.stats.removed || 0
    modified = 0 // API 不返回 modified 统计
    summary = `共 ${diff.stats.total} 行，${diff.stats.unchanged} 行未改变，${added} 行新增，${removed} 行删除`

    // 将 API 返回的 lines 转换为 changes 格式，只保留有变化的行
    changes = diff.lines
      .filter(line => line.type === 'added' || line.type === 'removed') // 只保留有变化的行
      .map((line, index) => {
        let type = 'modify' // 默认类型
        let path = `Line ${line.lineNum}`

        switch (line.type) {
          case 'added':
            type = 'add'
            break
          case 'removed':
            type = 'remove'
            break
          default:
            type = 'modify'
        }

        return {
          type,
          path,
          oldValue: line.type === 'removed' ? line.content : undefined,
          newValue: line.type === 'added' ? line.content : undefined,
          context: [line.content]
        }
      })
  } else {
    // 保持原有逻辑，处理旧格式数据
    changes = diff.changes || []
    added = diff.added || 0
    removed = diff.removed || 0
    modified = diff.modified || 0
    summary = diff.summary || ''
  }

  // 处理区块展开/折叠
  const handleSectionToggle = (index) => {
    setExpandedSections(prev => {
      if (prev.includes(index)) {
        return prev.filter(i => i !== index)
      } else {
        return [...prev, index]
      }
    })
  }

  // 渲染差异项
  const renderDiffItem = (change, index) => {
    const isExpanded = expandedSections.includes(index)
    const {
      type,
      path,
      oldValue,
      newValue,
      context = []
    } = change

    let typeIcon, typeColor, typeText

    switch (type) {
      case 'add':
        typeIcon = <PlusOutlined />;
        typeColor = '#52c41a';
        typeText = '新增';
        break;
      case 'remove':
        typeIcon = <MinusOutlined />;
        typeColor = '#ff4d4f';
        typeText = '删除';
        break;
      case 'modify':
        typeIcon = <EditOutlined />;
        typeColor = '#1890ff';
        typeText = '修改';
        break;
      default:
        typeIcon = <LineChartOutlined />;
        typeColor = '#8c8c8c';
        typeText = '未知';
    }

    return (
      <Card
        key={index}
        title={
          <Space>
            <span style={{ color: typeColor }}>{typeIcon}</span>
            <Text strong>{path || t('diffViewer.unknownPath')}</Text>
            <Tag color={type === 'add' ? 'success' : type === 'remove' ? 'error' : 'blue'}>
              {typeText}
            </Tag>
            <Button
              type="link"
              size="small"
              onClick={() => handleSectionToggle(index)}
              icon={isExpanded ? <UpOutlined /> : <DownOutlined />}
            >
              {isExpanded ? t('diffViewer.collapse') : t('diffViewer.expand')}
            </Button>
          </Space>
        }
        style={{ marginBottom: 16 }}
      >
        {isExpanded && (
          <div>
            {/* 显示旧值和新值 */}
            {(oldValue !== undefined || newValue !== undefined) && (
              <div style={{ marginBottom: 16 }}>
                {oldValue !== undefined && (
                  <div style={{ marginBottom: 8 }}>
                    <Text type="secondary" style={{ marginBottom: 4, display: 'block' }}>
                      {t('diffViewer.oldValue')}
                    </Text>
                    <div style={{ padding: 8, backgroundColor: '#fff1f0', borderRadius: 4, fontFamily: 'monospace' }}>
                      {typeof oldValue === 'object' ? (
                        <pre>{JSON.stringify(oldValue, null, 2)}</pre>
                      ) : (
                        <Text>{oldValue}</Text>
                      )}
                    </div>
                  </div>
                )}
                
                {newValue !== undefined && (
                  <div>
                    <Text type="secondary" style={{ marginBottom: 4, display: 'block' }}>
                      {t('diffViewer.newValue')}
                    </Text>
                    <div style={{ padding: 8, backgroundColor: '#f6ffed', borderRadius: 4, fontFamily: 'monospace' }}>
                      {typeof newValue === 'object' ? (
                        <pre>{JSON.stringify(newValue, null, 2)}</pre>
                      ) : (
                        <Text>{newValue}</Text>
                      )}
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* 显示上下文 */}
            {context.length > 0 && (
              <div>
                <Text type="secondary" style={{ marginBottom: 4, display: 'block' }}>
                  {t('diffViewer.context')}
                </Text>
                <div style={{ padding: 8, backgroundColor: '#f0f2f5', borderRadius: 4 }}>
                  <List
                    size="small"
                    dataSource={context}
                    renderItem={(item, ctxIndex) => (
                      <List.Item key={ctxIndex}>
                        <Text fontFamily="monospace">{item}</Text>
                      </List.Item>
                    )}
                  />
                </div>
              </div>
            )}
          </div>
        )}
      </Card>
    )
  }

  return (
    <div style={{ maxHeight: 600, overflow: 'auto' }}>
      {/* 差异概览 */}
      <Card
        title={
          <Space>
            <DiffOutlined />
            <Text strong>{t('diffViewer.configDiffOverview')}</Text>
          </Space>
        }
        style={{ marginBottom: 16 }}
      >
        <Space wrap style={{ marginBottom: 16 }}>
          <Tag color="success">
            <PlusOutlined /> {t('diffViewer.added')} {added}
          </Tag>
          <Tag color="error">
            <MinusOutlined /> {t('diffViewer.removed')} {removed}
          </Tag>
          <Tag color="blue">
            <EditOutlined /> {t('diffViewer.modified')} {modified}
          </Tag>
          <Tag color="default">
            <CheckOutlined /> {t('diffViewer.total')} {changes.length}
          </Tag>
        </Space>

        {summary && (
          <Paragraph>
            <Text type="secondary">{t('diffViewer.summary')}</Text>
            {summary}
          </Paragraph>
        )}

        {changes.length === 0 && (
          <Alert
            type="success"
            message={t('diffViewer.noDiff')}
            description={t('diffViewer.noDiffDescription')}
            showIcon
            style={{ marginTop: 8 }}
          />
        )}
      </Card>

      {/* 差异详情 */}
      {changes.length > 0 && (
        <div>
          <Text strong style={{ marginBottom: 16, display: 'block' }}>
            {t('diffViewer.diffDetails')}
          </Text>
          {changes.map(renderDiffItem)}
        </div>
      )}

      {/* 空状态 */}
      {changes.length === 0 && (
        <Card>
          <Alert
            type="success"
            message={t('diffViewer.configConsistent')}
            description={t('diffViewer.configConsistentDescription')}
            showIcon
            style={{ margin: 16 }}
          />
        </Card>
      )}
    </div>
  )
}

export default DiffViewer
