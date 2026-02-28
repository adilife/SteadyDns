import { useState, useEffect, useCallback } from 'react'
import {
  Table,
  Button,
  Modal,
  Form,
  Input,
  message,
  Space,
  Popconfirm,
  Tooltip,
  Spin,
  Card,
  Row,
  Col,
  Tabs,
  Select,
  InputNumber,
  Alert
} from 'antd'
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ReloadOutlined,
  DatabaseOutlined
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { apiClient } from '../utils/apiClient'

const { TabPane } = Tabs

/**
 * 权威域管理组件
 * 用于管理BIND权威域配置，包括域的增删改查、记录管理等
 */
const AuthZones = () => {
  const { t } = useTranslation()
  const [authZones, setAuthZones] = useState([])
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editingZone, setEditingZone] = useState(null)
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)
  const [activeTab, setActiveTab] = useState('basic')
  // 服务器记录管理状态
  const [records, setRecords] = useState([])
  const [editingRecord, setEditingRecord] = useState(null)
  const [isRecordModalOpen, setIsRecordModalOpen] = useState(false)
  const [recordForm] = Form.useForm()
  const [selectedRecordType, setSelectedRecordType] = useState('A')
  
  // Operation history state
  const [historyModalVisible, setHistoryModalVisible] = useState(false)
  const [historyRecords, setHistoryRecords] = useState([])
  const [historyLoading, setHistoryLoading] = useState(false)
  
  // Confirm modal state
  const [confirmModalVisible, setConfirmModalVisible] = useState(false)
  const [confirmModalTitle, setConfirmModalTitle] = useState('')
  const [confirmModalContent, setConfirmModalContent] = useState('')
  const [currentAction, setCurrentAction] = useState('')
  const [currentActionParams, setCurrentActionParams] = useState(null)
  
  // Plugin status state
  const [pluginEnabled, setPluginEnabled] = useState(true)
  const [checkingPluginStatus, setCheckingPluginStatus] = useState(true)

  // 记录类型与默认TTL的映射表
  const recordTypeTTLMap = {
    'A': 3600,
    'AAAA': 3600,
    'NS': 86400,
    'MX': 86400,
    'CNAME': 3600,
    'TXT': 3600,
    'SRV': 3600,
    'PTR': 86400,
    'CAA': 3600,
    'NAPTR': 3600,
    'DS': 3600,
    'DNSKEY': 3600,
    'RRSIG': 3600,
    'NSEC': 3600,
    'NSEC3': 3600,
    'SSHFP': 3600,
    'TLSA': 3600,
    'OPENPGPKEY': 3600,
    'SMIMEA': 3600
  }

  // 标准DNS记录类型列表
  const standardRecordTypes = Object.keys(recordTypeTTLMap)

  // 检查插件状态
  const checkPluginStatus = useCallback(async () => {
    setCheckingPluginStatus(true)
    try {
      const response = await apiClient.getPluginsStatus()
      if (response.success) {
        const bindPlugin = response.data.plugins.find(plugin => plugin.name === 'bind')
        setPluginEnabled(bindPlugin?.enabled || false)
      } else {
        console.error('Failed to check plugin status:', response.error)
        setPluginEnabled(false)
      }
    } catch (error) {
      console.error('Error checking plugin status:', error)
      setPluginEnabled(false)
    } finally {
      setCheckingPluginStatus(false)
    }
  }, [])

  // 加载权威域列表
  const loadAuthZones = useCallback(async () => {
    if (!pluginEnabled) return
    
    setLoading(true)
    try {
      const response = await apiClient.get('/bind-zones')
      if (response.success) {
        setAuthZones(response.data)
      } else {
        message.error(response.error || t('authZones.fetchError'))
      }
    } catch (error) {
      console.error('Error loading auth zones:', error)
      // 检查是否是404错误（插件禁用）
      if (error.message.includes('404')) {
        setPluginEnabled(false)
        message.error(t('plugins.bindNotEnabledOperate'))
      } else {
        message.error(t('authZones.fetchError'))
      }
    } finally {
      setLoading(false)
    }
  }, [pluginEnabled, t])

  // 组件挂载时加载数据
  useEffect(() => {
    // 先检查插件状态
    checkPluginStatus()
  }, [checkPluginStatus])

  // 插件状态变化时加载数据
  useEffect(() => {
    if (pluginEnabled) {
      loadAuthZones()
    }
  }, [pluginEnabled, loadAuthZones])

  // 实现30秒自动刷新数据
  useEffect(() => {
    if (!pluginEnabled) return
    
    // 设置30秒定时器
    const timer = setInterval(() => {
      loadAuthZones()
    }, 30000)
    
    // 组件卸载时清除定时器
    return () => {
      if (timer) {
        clearInterval(timer)
      }
    }
  }, [loadAuthZones, pluginEnabled])

  // 显示添加/编辑模态框
  const showModal = (zone = null) => {
    setEditingZone(zone)
    // 设置默认TAB：修改模式显示服务器记录，添加模式显示基本信息
    setActiveTab(zone ? 'records' : 'basic')
    setIsModalOpen(true)
    // 延迟设置字段值或重置表单，确保Form元素已渲染
    setTimeout(() => {
      if (zone) {
        form.setFieldsValue(zone)
      } else {
        form.resetFields()
      }
    }, 0)
  }

  // 关闭模态框
  const handleCancel = () => {
    setIsModalOpen(false)
    setEditingZone(null)
    // 在模态框关闭后重置表单，避免useForm实例未连接的警告
    setTimeout(() => {
      form.resetFields()
    }, 300)
  }

  // 记录管理相关功能
  // 显示添加/编辑记录模态框
  const showRecordModal = (record = null) => {
    setEditingRecord(record)
    
    if (record) {
      setSelectedRecordType(record.type)
    } else {
      setSelectedRecordType('A')
    }
    
    setIsRecordModalOpen(true)
    
    // 延迟设置字段值，确保Modal和Form元素已渲染
    setTimeout(() => {
      if (record) {
        // 确保编辑时的TTL值有效
        const validTTL = record.ttl && record.ttl > 0 ? record.ttl : (recordTypeTTLMap[record.type] || 3600);
        recordForm.setFieldsValue({
          ...record,
          ttl: validTTL
        })
      } else {
        // 设置默认TTL
        recordForm.setFieldsValue({
          type: 'A',
          ttl: recordTypeTTLMap['A'] || 3600
        })
      }
    }, 0)
  }

  // 关闭记录模态框
  const handleRecordCancel = () => {
    setIsRecordModalOpen(false)
    setEditingRecord(null)
    recordForm.resetFields()
  }

  // 记录类型变化时自动设置默认TTL
  const handleRecordTypeChange = (value) => {
    setSelectedRecordType(value)
    // 设置默认TTL
    recordForm.setFieldsValue({
      ttl: recordTypeTTLMap[value] || 3600
    })
  }

  // 添加或编辑记录
  const handleRecordOk = () => {
    recordForm.validateFields().then(values => {
      let updatedValues = { ...values };
      
      // 处理自定义记录类型
      if (updatedValues.type === 'custom') {
        updatedValues.type = updatedValues.customType;
        delete updatedValues.customType;
      }
      
      // 确保TTL值存在且符合要求
      if (!updatedValues.ttl || updatedValues.ttl <= 0) {
        updatedValues.ttl = recordTypeTTLMap[updatedValues.type] || 3600;
      }
      
      let updatedRecords = [...records]
      if (editingRecord) {
        // 编辑现有记录 - 通过ID定位
        const index = updatedRecords.findIndex(record => record.id === editingRecord.id)
        if (index !== -1) {
          updatedRecords[index] = { ...updatedValues, id: editingRecord.id }
        }
      } else {
        // 添加新记录 - 不生成前端ID，让后端自动生成
        updatedRecords.push(updatedValues)
      }
      setRecords(updatedRecords)
      setIsRecordModalOpen(false)
      setEditingRecord(null)
      recordForm.resetFields()
    }).catch(() => {
      message.error(t('authZones.formValidateError'))
    })
  }

  // 删除记录
  const handleDeleteRecord = (recordId) => {
    const updatedRecords = records.filter(record => record.id !== recordId)
    setRecords(updatedRecords)
  }

  /**
   * 加载操作历史记录
   * 从后端获取BIND区域操作历史
   */
  const loadHistoryRecords = useCallback(async () => {
    if (!pluginEnabled) {
      message.error(t('plugins.bindNotEnabledOperate'))
      setHistoryLoading(false)
      return
    }
    
    setHistoryLoading(true)
    try {
      const response = await apiClient.getBindZonesHistory()
      if (response.success) {
        // Sort history records by ID in descending order
        const sortedRecords = (response.data || []).sort((a, b) => b.id - a.id)
        setHistoryRecords(sortedRecords)
      } else {
        message.error(response.error || t('history.loadHistoryFailed'))
      }
    } catch (error) {
      console.error('Error loading history records:', error)
      if (error.message.includes('404')) {
        setPluginEnabled(false)
        message.error(t('plugins.bindNotEnabledOperate'))
      } else {
        message.error(t('history.loadHistoryFailed'))
      }
    } finally {
      setHistoryLoading(false)
    }
  }, [pluginEnabled, t])

  /**
   * 处理从历史记录恢复的确认操作
   * @param {number} historyId - 历史记录ID
   */
  const confirmRestoreFromHistory = (historyId) => {
    setConfirmModalTitle(t('authZones.restoreFromHistory'))
    setConfirmModalContent(t('authZones.confirmRestoreMessage'))
    setCurrentAction('restoreFromHistory')
    setCurrentActionParams(historyId)
    setConfirmModalVisible(true)
  }

  /**
   * 处理确认操作
   * 执行用户确认后的操作
   */
  const handleConfirmAction = async () => {
    setConfirmModalVisible(false)
    
    try {
      switch (currentAction) {
        case 'restoreFromHistory':
          await handleRestoreFromHistory(currentActionParams)
          break
        default:
          break
      }
    } catch (error) {
      console.error('Error executing action:', error)
      message.error(t('authZones.error'))
    }
  }

  // 当编辑区域变化时，更新记录列表
  useEffect(() => {
    if (editingZone && editingZone.records) {
      // 直接使用后端返回的记录，不再生成前端临时ID
      setRecords(editingZone.records);
    } else {
      setRecords([])
    }
  }, [editingZone])

  // 当操作历史模态框打开时，加载历史记录
  useEffect(() => {
    if (historyModalVisible) {
      loadHistoryRecords()
    }
  }, [historyModalVisible, loadHistoryRecords])

  /**
   * 准备提交数据，移除后端自动维护的字段
   * @param {object} values - 表单数据
   * @returns {object} 处理后的提交数据
   */
  const prepareSubmitData = (values) => {
    // eslint-disable-next-line no-unused-vars
    const { domain, file, ...restData } = values

    // 移除SOA中的serial字段（如果存在）
    const soa = restData.soa ? { ...restData.soa } : null
    if (soa && soa.serial !== undefined) {
      delete soa.serial
    }

    return {
      ...restData,
      soa,
      records: records
    }
  }

  /**
   * 提交表单
   * 处理权威域的创建或更新
   */
  const handleOk = () => {
    form.validateFields().then(values => {
      setLoading(true)
      // 准备提交数据，移除后端自动维护的字段
      const submitData = prepareSubmitData(values, !!editingZone)

      if (editingZone) {
        updateAuthZone(editingZone.domain, submitData)
      } else {
        // 创建时需要domain字段
        const createData = {
          domain: values.domain,
          ...submitData
        }
        createAuthZone(createData)
      }
    }).catch(() => {
      message.error(t('authZones.formValidateError'))
      setLoading(false)
    })
  }

  /**
   * 创建权威域
   * @param {object} values - 权威域数据
   */
  const createAuthZone = async (values) => {
    if (!pluginEnabled) {
      message.error(t('plugins.bindNotEnabledOperate'))
      setLoading(false)
      return
    }
    
    try {
      const response = await apiClient.post('/bind-zones', values)
      if (response.success) {
        message.success(response.data.message || t('authZones.createSuccess'))
        loadAuthZones()
        setIsModalOpen(false)
        setEditingZone(null)
        form.resetFields()
      } else {
        message.error(response.error || t('authZones.createError'))
      }
    } catch (error) {
      console.error('Error creating auth zone:', error)
      if (error.message.includes('404')) {
        setPluginEnabled(false)
        message.error(t('plugins.bindNotEnabledOperate'))
      } else {
        message.error(t('authZones.createError'))
      }
    } finally {
      setLoading(false)
    }
  }

  /**
   * 更新权威域
   * @param {string} domain - 域名
   * @param {object} values - 更新数据
   */
  const updateAuthZone = async (domain, values) => {
    if (!pluginEnabled) {
      message.error(t('plugins.bindNotEnabledOperate'))
      setLoading(false)
      return
    }
    
    try {
      const response = await apiClient.put(`/bind-zones/${domain}`, values)
      if (response.success) {
        message.success(response.data.message || t('authZones.updateSuccess'))
        loadAuthZones()
        setIsModalOpen(false)
        setEditingZone(null)
        form.resetFields()
      } else {
        message.error(response.error || t('authZones.updateError'))
      }
    } catch (error) {
      console.error('Error updating auth zone:', error)
      if (error.message.includes('404')) {
        setPluginEnabled(false)
        message.error(t('plugins.bindNotEnabledOperate'))
      } else {
        message.error(t('authZones.updateError'))
      }
    } finally {
      setLoading(false)
    }
  }

  /**
   * 删除权威域
   * @param {string} domain - 域名
   */
  const deleteAuthZone = async (domain) => {
    if (!pluginEnabled) {
      message.error(t('plugins.bindNotEnabledOperate'))
      return
    }
    
    try {
      const response = await apiClient.delete(`/bind-zones/${domain}`)
      if (response.success) {
        message.success(response.data.message || t('authZones.deleteSuccess'))
        loadAuthZones()
      } else {
        message.error(response.error || t('authZones.deleteError'))
      }
    } catch (error) {
      console.error('Error deleting auth zone:', error)
      if (error.message.includes('404')) {
        setPluginEnabled(false)
        message.error(t('plugins.bindNotEnabledOperate'))
      } else {
        message.error(t('authZones.deleteError'))
      }
    }
  }

  /**
   * 刷新权威域配置
   * @param {string} domain - 域名
   */
  const reloadAuthZone = async (domain) => {
    if (!pluginEnabled) {
      message.error(t('plugins.bindNotEnabledOperate'))
      return
    }
    
    try {
      const response = await apiClient.post(`/bind-zones/${domain}/reload`)
      if (response.success) {
        message.success(response.data.message || t('authZones.reloadSuccess'))
      } else {
        message.error(response.error || t('authZones.reloadError'))
      }
    } catch (error) {
      console.error('Error reloading auth zone:', error)
      if (error.message.includes('404')) {
        setPluginEnabled(false)
        message.error(t('plugins.bindNotEnabledOperate'))
      } else {
        message.error(t('authZones.reloadError'))
      }
    }
  }

  /**
   * 从历史记录恢复
   * @param {number} historyId - 历史记录ID
   */
  const handleRestoreFromHistory = async (historyId) => {
    if (!pluginEnabled) {
      message.error(t('plugins.bindNotEnabledOperate'))
      setHistoryLoading(false)
      return
    }
    
    try {
      setHistoryLoading(true)
      const response = await apiClient.restoreBindZoneFromHistory(historyId)
      if (response.success) {
        message.success(response.data.message || t('history.restoreSuccess'))
        // Reload auth zones after restore
        setTimeout(loadAuthZones, 1000)
        setHistoryModalVisible(false)
      } else {
        message.error(response.error || t('history.restoreFailed'))
      }
    } catch (error) {
      console.error('Error restoring from history:', error)
      if (error.message.includes('404')) {
        setPluginEnabled(false)
        message.error(t('plugins.bindNotEnabledOperate'))
      } else {
        message.error(t('history.restoreFailed'))
      }
    } finally {
      setHistoryLoading(false)
    }
  }

  /**
   * 表格列配置
   * 定义权威域列表的列信息
   */
  const columns = [
    {
      title: t('authZones.domain'),
      dataIndex: 'domain',
      key: 'domain',
      ellipsis: true,
      render: (text) => (
        <span style={{ fontWeight: 'bold' }}>{text}</span>
      )
    },
    {
      title: t('authZones.type'),
      dataIndex: 'type',
      key: 'type',
      width: 120
    },
    {
      title: t('authZones.allowQuery'),
      dataIndex: 'allow_query',
      key: 'allow_query',
      width: 150
    },
    {
      title: t('authZones.comment'),
      dataIndex: 'comment',
      key: 'comment',
      width: 300,
      ellipsis: true,
      render: (text) => (
        <Tooltip title={text}>
          <span>{text}</span>
        </Tooltip>
      )
    },
    {
      title: t('authZones.records'),
      key: 'records',
      width: 120,
      render: (_, record) => {
        // 计算总记录数，不包含SOA记录
        const totalRecords = record.records?.length || 0;
        return (
          <div>
            {t('authZones.totalRecords')}: {totalRecords}
          </div>
        );
      }
    },
    {
      title: t('authZones.actions'),
      key: 'actions',
      width: 180,
      render: (_, record) => (
        <Space size="middle">
          <Tooltip title={t('authZones.edit')}>
            <Button
              icon={<EditOutlined />}
              size="small"
              onClick={() => showModal(record)}
            />
          </Tooltip>
          <Tooltip title={t('authZones.reload')}>
            <Button
              icon={<ReloadOutlined />}
              size="small"
              onClick={() => reloadAuthZone(record.domain)}
            />
          </Tooltip>
          <Tooltip title={t('authZones.delete')}>
            <Popconfirm
              title={t('authZones.confirmDelete')}
              onConfirm={() => deleteAuthZone(record.domain)}
              okText={t('authZones.yes')}
              cancelText={t('authZones.no')}
            >
              <Button
                icon={<DeleteOutlined />}
                size="small"
                danger
              />
            </Popconfirm>
          </Tooltip>
        </Space>
      )
    }
  ]

  /**
   * 当插件未启用时显示的提示信息
   */
  if (!pluginEnabled) {
    return (
      <div style={{ textAlign: 'center', padding: '60px 20px' }}>
        <div style={{ maxWidth: '600px', margin: '0 auto' }}>
          <Alert
            message={t('plugins.bindNotEnabled')}
            description={
              <div>
                <p style={{ marginBottom: '16px' }}>{t('plugins.bindNotEnabledDescription')}</p>
                <p style={{ marginBottom: '8px' }}><strong>{t('plugins.enableMethod')}</strong></p>
                <p>{t('plugins.enableStep1')}</p>
                <p>{t('plugins.enableStep2')}</p>
                <p>{t('plugins.enableStep3')}</p>
              </div>
            }
            type="warning"
            showIcon
            style={{ marginBottom: '24px' }}
          />
        </div>
      </div>
    )
  }

  /**
   * 当检查插件状态时显示加载状态
   */
  if (checkingPluginStatus) {
    return (
      <div style={{ textAlign: 'center', padding: '60px' }}>
        <Spin size="large" tip={t('plugins.checkingPluginStatus')}>
          <div style={{ padding: 50 }} />
        </Spin>
      </div>
    )
  }

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2>
          <Space>
            <DatabaseOutlined />
            {t('authZones.title')}
          </Space>
        </h2>
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => setHistoryModalVisible(true)}
          >
            {t('authZones.operationHistory')}
          </Button>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => showModal()}
          >
            {t('authZones.addZone')}
          </Button>
        </Space>
      </div>

      <Spin spinning={loading}>
        <Table
          columns={columns}
          dataSource={authZones}
          rowKey="domain"
          pagination={{
            showSizeChanger: true,
            pageSizeOptions: ['10', '20', '50'],
            defaultPageSize: 10
          }}
          scroll={{ x: 'max-content' }}
        />
      </Spin>

      {/* 添加/编辑权威域模态框 */}
      <Modal
        title={editingZone ? t('authZones.editZone') : t('authZones.addNewZone')}
        open={isModalOpen}
        onOk={handleOk}
        onCancel={handleCancel}
        width={1200}
        destroyOnHidden
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{
          domain: '',
          type: 'master',
          file: '',
          allow_query: 'any',
          comment: '',
          soa: {
            primary_ns: '',
            admin_email: '',
            refresh: '3600',
            retry: '1800',
            expire: '604800',
            minimum_ttl: '86400'
          },
          records: []
        }}
        >
          <Tabs
            activeKey={activeTab}
            onChange={setActiveTab}
            items={[
              {
                key: 'basic',
                label: t('authZones.basic'),
                children: (
                  <>
                    <Form.Item
                      name="domain"
                      label={t('authZones.domain')}
                      rules={[
                        { required: true, message: t('authZones.domainRequired') },
                        { type: 'string', max: 255, message: t('authZones.domainMax') }
                      ]}
                    >
                      <Input placeholder={t('authZones.domainPlaceholder')} />
                    </Form.Item>

                    <Form.Item
                      name="type"
                      label={t('authZones.type')}
                      rules={[{ required: true, message: t('authZones.typeRequired') }]}
                    >
                      <Input placeholder={t('authZones.typePlaceholder')} />
                    </Form.Item>

                    <Form.Item
                      name="allow_query"
                      label={t('authZones.allowQuery')}
                    >
                      <Input placeholder={t('authZones.allowQueryPlaceholder')} />
                    </Form.Item>

                    <Form.Item
                      name="comment"
                      label={t('authZones.comment')}
                    >
                      <Input.TextArea 
                        rows={4} 
                        placeholder={t('authZones.commentPlaceholder')} 
                      />
                    </Form.Item>

                  </>
                )
              },
              {
                key: 'soa',
                label: t('authZones.soaRecord'),
                children: (
                  <Card title={t('authZones.soaRecord')} size="small">
                    <Form.Item
                      name={['soa', 'primary_ns']}
                      label={t('authZones.primaryNs')}
                      rules={[{ required: true, message: t('authZones.primaryNsRequired') }]}
                    >
                      <Input placeholder={t('authZones.primaryNsPlaceholder')} />
                    </Form.Item>

                    <Form.Item
                      name={['soa', 'admin_email']}
                      label={t('authZones.adminEmail')}
                      rules={[{ required: true, message: t('authZones.adminEmailRequired') }]}
                    >
                      <Input placeholder={t('authZones.adminEmailPlaceholder')} />
                    </Form.Item>

                    <Row gutter={16}>
                      <Col xs={12}>
                        <Form.Item
                          name={['soa', 'refresh']}
                          label={t('authZones.refresh')}
                          rules={[{ required: true, message: t('authZones.refreshRequired') }]}
                        >
                          <Input placeholder={t('authZones.refreshPlaceholder')} />
                        </Form.Item>
                      </Col>
                      <Col xs={12}>
                        <Form.Item
                          name={['soa', 'retry']}
                          label={t('authZones.retry')}
                          rules={[{ required: true, message: t('authZones.retryRequired') }]}
                        >
                          <Input placeholder={t('authZones.retryPlaceholder')} />
                        </Form.Item>
                      </Col>
                      <Col xs={12}>
                        <Form.Item
                          name={['soa', 'expire']}
                          label={t('authZones.expire')}
                          rules={[{ required: true, message: t('authZones.expireRequired') }]}
                        >
                          <Input placeholder={t('authZones.expirePlaceholder')} />
                        </Form.Item>
                      </Col>
                      <Col xs={12}>
                        <Form.Item
                          name={['soa', 'minimum_ttl']}
                          label={t('authZones.minimumTtl')}
                          rules={[{ required: true, message: t('authZones.minimumTtlRequired') }]}
                        >
                          <Input placeholder={t('authZones.minimumTtlPlaceholder')} />
                        </Form.Item>
                      </Col>
                    </Row>
                  </Card>
                )
              },
              {
                key: 'records',
                label: t('authZones.records'),
                children: (
                  <>
                    <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'flex-end' }}>
                      <Button
                        type="primary"
                        icon={<PlusOutlined />}
                        onClick={() => showRecordModal()}
                      >
                        {t('authZones.addRecord')}
                      </Button>
                    </div>

                    <Card title={t('authZones.recordsList')} size="small">
                      <Table
                        dataSource={records}
                        rowKey="id"
                        pagination={{
                          showSizeChanger: true,
                          pageSizeOptions: ['10', '20', '50'],
                          defaultPageSize: 10
                        }}
                        scroll={{ x: 'max-content' }}
                        columns={[
                          {
                            title: t('authZones.name'),
                            dataIndex: 'name',
                            key: 'name',
                            width: 150,
                            ellipsis: true
                          },
                          {
                            title: t('authZones.type'),
                            dataIndex: 'type',
                            key: 'type',
                            width: 100
                          },
                          {
                            title: t('authZones.value'),
                            dataIndex: 'value',
                            key: 'value',
                            width: 200,
                            ellipsis: true
                          },
                          {
                            title: t('authZones.priority'),
                            dataIndex: 'priority',
                            key: 'priority',
                            width: 100
                          },
                          {
                            title: t('authZones.ttl'),
                            dataIndex: 'ttl',
                            key: 'ttl',
                            width: 100
                          },
                          {
                            title: t('authZones.comment'),
                            dataIndex: 'comment',
                            key: 'comment',
                            width: 300,
                            ellipsis: true
                          },
                          {
                            title: t('authZones.actions'),
                            key: 'actions',
                            width: 80,
                            render: (_, record) => (
                              <Space size="middle">
                                <Tooltip title={t('authZones.edit')}>
                                  <Button
                                    type="link"
                                    size="small"
                                    icon={<EditOutlined />}
                                    onClick={() => showRecordModal(record)}
                                  />
                                </Tooltip>
                                <Tooltip title={t('authZones.delete')}>
                                  <Button
                                    type="link"
                                    size="small"
                                    danger
                                    icon={<DeleteOutlined />}
                                    onClick={() => handleDeleteRecord(record.id)}
                                  />
                                </Tooltip>
                              </Space>
                            )
                          }
                        ]}
                      />
                    </Card>
                  </>
                )
              }
            ]}
          />
        </Form>
      </Modal>

      {/* 添加/编辑记录模态框 */}
      <Modal
        title={editingRecord ? t('authZones.editRecord') : t('authZones.addRecord')}
        open={isRecordModalOpen}
        onOk={handleRecordOk}
        onCancel={handleRecordCancel}
        width={600}
        destroyOnHidden
      >
        <Form
          form={recordForm}
          layout="vertical"
        >
          <Form.Item
            name="name"
            label={t('authZones.name')}
            rules={[{ required: true, message: t('authZones.nameRequired') }]}
          >
            <Input placeholder={t('authZones.namePlaceholder')} />
          </Form.Item>

          <Form.Item
            name="type"
            label={t('authZones.type')}
            rules={[{ required: true, message: t('authZones.typeRequired') }]}
          >
            <Select
              value={selectedRecordType}
              onChange={handleRecordTypeChange}
              placeholder={t('authZones.typePlaceholder')}
              allowClear={false}
            >
              {standardRecordTypes.map(type => (
                <Select.Option key={type} value={type}>
                  {type}
                </Select.Option>
              ))}
              <Select.Option value="custom">{t('authZones.customType')}</Select.Option>
            </Select>
          </Form.Item>

          {/* 自定义记录类型输入框 */}
          {selectedRecordType === 'custom' && (
            <Form.Item
              name="customType"
              label={t('authZones.customType')}
              rules={[{ required: true, message: t('authZones.customTypeRequired') }]}
            >
              <Input placeholder={t('authZones.customTypePlaceholder')} />
            </Form.Item>
          )}

          <Form.Item
            name="value"
            label={t('authZones.value')}
            rules={[{ required: true, message: t('authZones.valueRequired') }]}
          >
            <Input.TextArea
              rows={3}
              placeholder={t('authZones.valuePlaceholder')}
            />
          </Form.Item>

          {/* MX记录优先级 */}
          {selectedRecordType === 'MX' && (
            <Form.Item
              name="priority"
              label={t('authZones.priority')}
              rules={[{ required: true, message: t('authZones.priorityRequired') }]}
            >
              <InputNumber
                min={0}
                max={65535}
                placeholder={t('authZones.priorityPlaceholder')}
                style={{ width: '100%' }}
              />
            </Form.Item>
          )}

          <Form.Item
            name="ttl"
            label={t('authZones.ttl')}
            rules={[{ required: true, message: t('authZones.ttlRequired') }]}
          >
            <InputNumber
              min={0}
              max={2147483647}
              placeholder={t('authZones.ttlPlaceholder')}
              style={{ width: '100%' }}
            />
          </Form.Item>

          <Form.Item
            name="comment"
            label={t('authZones.comment')}
          >
            <Input.TextArea
              rows={2}
              placeholder={t('authZones.commentPlaceholder')}
            />
          </Form.Item>
        </Form>
      </Modal>
      
      {/* Operation History Modal */}
      <Modal
        title={t('authZones.operationHistoryTitle')}
        open={historyModalVisible}
        onOk={() => setHistoryModalVisible(false)}
        onCancel={() => setHistoryModalVisible(false)}
        okText={t('authZones.close')}
        cancelText={t('authZones.cancel')}
        width={1000}
        styles={{ body: { maxHeight: 600, overflow: 'auto' } }}
      >
        <div>
          <h3 style={{ marginBottom: 16 }}>{t('authZones.operationHistoryRecords')}</h3>
          <Spin spinning={historyLoading}>
            {historyRecords.length > 0 ? (
              <Table
                dataSource={historyRecords}
                rowKey="id"
                pagination={{
                  showSizeChanger: true,
                  pageSizeOptions: ['10', '20', '50'],
                  defaultPageSize: 10
                }}
                scroll={{ x: 'max-content' }}
                columns={[
                  {
                    title: t('authZones.id'),
                    dataIndex: 'id',
                    key: 'id',
                    width: 80
                  },
                  {
                    title: t('authZones.operationType'),
                    dataIndex: 'operation',
                    key: 'operation',
                    width: 120,
                    render: (operation) => {
                      const operationMap = {
                        create: t('authZones.create'),
                        update: t('authZones.update'),
                        delete: t('authZones.deleteOperation'),
                        rollback: t('authZones.rollback')
                      }
                      return operationMap[operation] || operation
                    }
                  },
                  {
                    title: t('authZones.domain'),
                    dataIndex: 'domain',
                    key: 'domain',
                    width: 200
                  },
                  {
                    title: t('authZones.operationTime'),
                    dataIndex: 'timestamp',
                    key: 'timestamp',
                    width: 200,
                    render: (timestamp) => {
                      return new Date(timestamp).toLocaleString()
                    }
                  },
                  {
                    title: t('authZones.actions'),
                    key: 'actions',
                    width: 100,
                    render: (_, record) => (
                      <Button
                        type="primary"
                        size="small"
                        onClick={() => confirmRestoreFromHistory(record.id)}
                      >
                        {t('authZones.restore')}
                      </Button>
                    )
                  }
                ]}
              />
            ) : (
              <Alert
                title={t('authZones.noOperationHistory')}
                description={t('authZones.noOperationHistoryDescription')}
                type="info"
                showIcon
              />
            )}
          </Spin>
        </div>
      </Modal>
      
      {/* Confirm Modal */}
      <Modal
        title={confirmModalTitle}
        open={confirmModalVisible}
        onOk={handleConfirmAction}
        onCancel={() => setConfirmModalVisible(false)}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
      >
        <div>
          <Alert
            title={t('common.operationConfirm')}
            description={confirmModalContent}
            type="warning"
            showIcon
            style={{ marginBottom: 16 }}
          />
          <p>{t('common.confirmOperationPrompt')}</p>
        </div>
      </Modal>
    </div>
  )
}

export default AuthZones
