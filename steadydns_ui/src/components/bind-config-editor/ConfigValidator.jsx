import { useState } from 'react'
import {
  Card,
  List,
  Typography,
  Alert,
  Tag,
  Button,
  Space,
  Divider,
  Collapse
} from 'antd'
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  AlertOutlined,
  LineChartOutlined,
  DownOutlined,
  UpOutlined
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'


const { Text, Paragraph } = Typography
const { Panel } = Collapse

const ConfigValidator = ({ result }) => {
  // 国际化
  const { t } = useTranslation()
  
  const [expandedErrors, setExpandedErrors] = useState([])

  if (!result) {
    return (
      <div style={{ padding: 16 }}>
        <Alert
          message={t('configValidator.noValidationResult')}
          description={t('configValidator.pleaseValidateFirst')}
          type="info"
          showIcon
        />
      </div>
    )
  }

  // 处理 API 返回的错误和输出字段
  const valid = result?.valid === true
  const errors = []
  const warnings = []
  const info = []
  const details = {}
  
  // 如果有 error 字段，添加到 errors 数组
  if (!valid && result?.error) {
    errors.push({ message: result.error })
  }
  
  // 如果有 output 字段，添加到 details
  if (result?.output) {
    details.output = result.output
  }
  
  // 保留原有的错误、警告和信息
  if (Array.isArray(result?.errors)) {
    errors.push(...result.errors)
  }
  
  if (Array.isArray(result?.warnings)) {
    warnings.push(...result.warnings)
  }
  
  if (Array.isArray(result?.info)) {
    info.push(...result.info)
  }
  
  if (typeof result?.details === 'object' && result.details !== null) {
    Object.assign(details, result.details)
  }

  // 处理错误展开/折叠
  const handleErrorToggle = (index) => {
    setExpandedErrors(prev => {
      if (prev.includes(index)) {
        return prev.filter(i => i !== index)
      } else {
        return [...prev, index]
      }
    })
  }

  // 渲染错误项
  const renderErrorItem = (error, index) => {
    const isExpanded = expandedErrors.includes(index)
    
    return (
      <List.Item key={index}>
        <div style={{ width: '100%' }}>
          <Space style={{ marginBottom: 8 }}>
            <CloseCircleOutlined style={{ color: '#ff4d4f' }} />
            <Text strong style={{ color: '#ff4d4f' }}>{error.message || t('configValidator.error')}</Text>
            <Tag color="error">{t('configValidator.error')}</Tag>
            {error.line && (
              <Tag color="default">{t('configValidator.line')} {error.line}</Tag>
            )}
          </Space>
          
          {(error.details || error.context) && (
            <div>
              <Button
                type="link"
                size="small"
                onClick={() => handleErrorToggle(index)}
                icon={isExpanded ? <UpOutlined /> : <DownOutlined />}
              >
                {isExpanded ? t('configValidator.toggleDetailsCollapsed') : t('configValidator.toggleDetails')}
              </Button>
              
              {isExpanded && (
                <div style={{ marginLeft: 24, marginTop: 8, padding: 12, backgroundColor: '#fff1f0', borderRadius: 4 }}>
                  {error.details && (
                    <Paragraph>
                      <Text type="secondary">{t('configValidator.details')}：</Text>
                      {error.details}
                    </Paragraph>
                  )}
                  {error.context && (
                    <Paragraph>
                      <Text type="secondary">{t('configValidator.context')}：</Text>
                      <Text code>{error.context}</Text>
                    </Paragraph>
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      </List.Item>
    )
  }

  // 渲染警告项
  const renderWarningItem = (warning, index) => {
    return (
      <List.Item key={index}>
        <Space>
            <AlertOutlined style={{ color: '#faad14' }} />
            <Text>{warning.message || t('configValidator.warning')}</Text>
            <Tag color="warning">{t('configValidator.warning')}</Tag>
            {warning.line && (
              <Tag color="default">{t('configValidator.line')} {warning.line}</Tag>
            )}
          </Space>
      </List.Item>
    )
  }

  // 渲染信息项
  const renderInfoItem = (infoItem, index) => {
    return (
      <List.Item key={index}>
        <Space>
            <LineChartOutlined style={{ color: '#1890ff' }} />
            <Text>{infoItem.message || t('configValidator.info')}</Text>
            <Tag color="blue">{t('configValidator.info')}</Tag>
          </Space>
      </List.Item>
    )
  }

  return (
    <div style={{ maxHeight: 500, overflow: 'auto' }}>
      {/* 验证状态概览 */}
      <Card
        title={
          <Space>
            {valid ? (
              <>
                <CheckCircleOutlined style={{ color: '#52c41a' }} />
                <Text strong>{t('configValidator.validationSuccess')}</Text>
              </>
            ) : (
              <>
                <CloseCircleOutlined style={{ color: '#ff4d4f' }} />
                <Text strong>{t('configValidator.validationFailed')}</Text>
              </>
            )}
          </Space>
        }
        style={{ marginBottom: 16 }}
      >
        <Space>
          <Alert
            type={valid ? 'success' : 'error'}
            title={valid ? t('configValidator.configValid') : t('configValidator.configInvalid')}
            description={
              valid 
                ? t('configValidator.configValidDescription') 
                : t('configValidator.configInvalidDescription', { errors: errors.length, warnings: warnings.length })
            }
            showIcon
          />
        </Space>

        {/* 验证详情 */}
        {Object.keys(details).length > 0 && (
          <Collapse style={{ marginTop: 16 }}>
            <Panel header={t('configValidator.validationDetails')} key="details">
              <List
                size="small"
                dataSource={Object.entries(details)}
                renderItem={([key, value]) => (
                  <List.Item>
                    <Text strong>{key}:</Text>
                    <Text style={{ marginLeft: 8 }}>{value}</Text>
                  </List.Item>
                )}
              />
            </Panel>
          </Collapse>
        )}
      </Card>

      {/* 错误列表 */}
      {errors.length > 0 && (
        <Card
          title={
          <Space>
            <CloseCircleOutlined style={{ color: '#ff4d4f' }} />
            <Text strong>{t('configValidator.error')} ({errors.length})</Text>
          </Space>
        }
          style={{ marginBottom: 16 }}
        >
          <List
            size="small"
            bordered
            dataSource={errors}
            renderItem={renderErrorItem}
          />
        </Card>
      )}

      {/* 警告列表 */}
      {warnings.length > 0 && (
        <Card
          title={
          <Space>
            <AlertOutlined style={{ color: '#faad14' }} />
            <Text strong>{t('configValidator.warning')} ({warnings.length})</Text>
          </Space>
        }
          style={{ marginBottom: 16 }}
        >
          <List
            size="small"
            bordered
            dataSource={warnings}
            renderItem={renderWarningItem}
          />
        </Card>
      )}

      {/* 信息列表 */}
      {info.length > 0 && (
        <Card
          title={
          <Space>
            <LineChartOutlined style={{ color: '#1890ff' }} />
            <Text strong>{t('configValidator.info')} ({info.length})</Text>
          </Space>
        }
        >
          <List
            size="small"
            bordered
            dataSource={info}
            renderItem={renderInfoItem}
          />
        </Card>
      )}

      {/* 成功提示 */}
      {valid && errors.length === 0 && warnings.length === 0 && (
        <Card>
          <Alert
            type="success"
            message={t('configValidator.validationPassed')}
            description={t('configValidator.validationPassedDescription')}
            showIcon
            style={{ margin: 16 }}
          />
        </Card>
      )}
    </div>
  )
}

export default ConfigValidator
