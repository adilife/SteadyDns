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
import { t } from '../i18n'
import { apiClient } from '../utils/apiClient'

const { TabPane } = Tabs

const AuthZones = ({ currentLanguage }) => {
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

  // 加载权威域列表
  const loadAuthZones = useCallback(async () => {
    setLoading(true)
    try {
      const response = await apiClient.get('/bind-zones')
      if (response.success) {
        setAuthZones(response.data)
      } else {
        message.error(response.error || t('authZones.fetchError', currentLanguage))
      }
    } catch (error) {
      console.error('Error loading auth zones:', error)
      message.error(t('authZones.fetchError', currentLanguage))
    } finally {
      setLoading(false)
    }
  }, [currentLanguage])

  // 组件挂载时加载数据
  useEffect(() => {
    loadAuthZones()
  }, [loadAuthZones])

  // 实现30秒自动刷新数据
  useEffect(() => {
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
  }, [loadAuthZones])

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
      message.error(t('authZones.formValidateError', currentLanguage))
    })
  }

  // 删除记录
  const handleDeleteRecord = (recordId) => {
    const updatedRecords = records.filter(record => record.id !== recordId)
    setRecords(updatedRecords)
  }

  // Load operation history records
  const loadHistoryRecords = async () => {
    setHistoryLoading(true)
    try {
      const response = await apiClient.getBindZonesHistory()
      if (response.success) {
        // Sort history records by ID in descending order
        const sortedRecords = (response.data || []).sort((a, b) => b.id - a.id)
        setHistoryRecords(sortedRecords)
      } else {
        message.error(response.error || '加载操作历史失败')
      }
    } catch (error) {
      console.error('Error loading history records:', error)
      message.error('加载操作历史失败')
    } finally {
      setHistoryLoading(false)
    }
  }

  // Handle restore from history
  const handleRestoreFromHistory = async (historyId) => {
    try {
      setHistoryLoading(true)
      const response = await apiClient.restoreBindZoneFromHistory(historyId)
      if (response.success) {
        message.success(response.data.message || '从历史记录恢复成功')
        // Reload auth zones after restore
        setTimeout(loadAuthZones, 1000)
        setHistoryModalVisible(false)
      } else {
        message.error(response.error || '从历史记录恢复失败')
      }
    } catch (error) {
      console.error('Error restoring from history:', error)
      message.error('从历史记录恢复失败')
    } finally {
      setHistoryLoading(false)
    }
  }

  // Handle restore from history with confirmation
  const confirmRestoreFromHistory = (historyId) => {
    setConfirmModalTitle('从历史记录恢复')
    setConfirmModalContent('确定要从历史记录恢复吗？这将覆盖当前的配置。')
    setCurrentAction('restoreFromHistory')
    setCurrentActionParams(historyId)
    setConfirmModalVisible(true)
  }

  // Handle confirm action
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
      message.error(t('authZones.error', currentLanguage))
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
  }, [historyModalVisible])

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

  // 提交表单
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
      message.error(t('authZones.formValidateError', currentLanguage))
      setLoading(false)
    })
  }

  // 创建权威域
  const createAuthZone = async (values) => {
    try {
      const response = await apiClient.post('/bind-zones', values)
      if (response.success) {
        message.success(response.data.message || t('authZones.createSuccess', currentLanguage))
        loadAuthZones()
        setIsModalOpen(false)
        setEditingZone(null)
        form.resetFields()
      } else {
        message.error(response.error || t('authZones.createError', currentLanguage))
      }
    } catch (error) {
      console.error('Error creating auth zone:', error)
      message.error(t('authZones.createError', currentLanguage))
    } finally {
      setLoading(false)
    }
  }

  // 更新权威域
  const updateAuthZone = async (domain, values) => {
    try {
      const response = await apiClient.put(`/bind-zones/${domain}`, values)
      if (response.success) {
        message.success(response.data.message || t('authZones.updateSuccess', currentLanguage))
        loadAuthZones()
        setIsModalOpen(false)
        setEditingZone(null)
        form.resetFields()
      } else {
        message.error(response.error || t('authZones.updateError', currentLanguage))
      }
    } catch (error) {
      console.error('Error updating auth zone:', error)
      message.error(t('authZones.updateError', currentLanguage))
    } finally {
      setLoading(false)
    }
  }

  // 删除权威域
  const deleteAuthZone = async (domain) => {
    try {
      const response = await apiClient.delete(`/bind-zones/${domain}`)
      if (response.success) {
        message.success(response.data.message || t('authZones.deleteSuccess', currentLanguage))
        loadAuthZones()
      } else {
        message.error(response.error || t('authZones.deleteError', currentLanguage))
      }
    } catch (error) {
      console.error('Error deleting auth zone:', error)
      message.error(t('authZones.deleteError', currentLanguage))
    }
  }

  // 刷新权威域配置
  const reloadAuthZone = async (domain) => {
    try {
      const response = await apiClient.post(`/bind-zones/${domain}/reload`)
      if (response.success) {
        message.success(response.data.message || t('authZones.reloadSuccess', currentLanguage))
      } else {
        message.error(response.error || t('authZones.reloadError', currentLanguage))
      }
    } catch (error) {
      console.error('Error reloading auth zone:', error)
      message.error(t('authZones.reloadError', currentLanguage))
    }
  }

  // 表格列配置
  const columns = [
    {
      title: t('authZones.domain', currentLanguage),
      dataIndex: 'domain',
      key: 'domain',
      ellipsis: true,
      render: (text) => (
        <span style={{ fontWeight: 'bold' }}>{text}</span>
      )
    },
    {
      title: t('authZones.type', currentLanguage),
      dataIndex: 'type',
      key: 'type',
      width: 120
    },
    {
      title: t('authZones.allowQuery', currentLanguage),
      dataIndex: 'allow_query',
      key: 'allow_query',
      width: 150
    },
    {
      title: t('authZones.comment', currentLanguage),
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
      title: t('authZones.records', currentLanguage),
      key: 'records',
      width: 120,
      render: (_, record) => {
        // 计算总记录数，不包含SOA记录
        const totalRecords = record.records?.length || 0;
        return (
          <div>
            {t('authZones.totalRecords', currentLanguage)}: {totalRecords}
          </div>
        );
      }
    },
    {
      title: t('authZones.actions', currentLanguage),
      key: 'actions',
      width: 180,
      render: (_, record) => (
        <Space size="middle">
          <Tooltip title={t('authZones.edit', currentLanguage)}>
            <Button
              icon={<EditOutlined />}
              size="small"
              onClick={() => showModal(record)}
            />
          </Tooltip>
          <Tooltip title={t('authZones.reload', currentLanguage)}>
            <Button
              icon={<ReloadOutlined />}
              size="small"
              onClick={() => reloadAuthZone(record.domain)}
            />
          </Tooltip>
          <Tooltip title={t('authZones.delete', currentLanguage)}>
            <Popconfirm
              title={t('authZones.confirmDelete', currentLanguage)}
              onConfirm={() => deleteAuthZone(record.domain)}
              okText={t('authZones.yes', currentLanguage)}
              cancelText={t('authZones.no', currentLanguage)}
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

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2>
          <Space>
            <DatabaseOutlined />
            {t('authZones.title', currentLanguage)}
          </Space>
        </h2>
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => setHistoryModalVisible(true)}
          >
            操作历史
          </Button>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => showModal()}
          >
            {t('authZones.addZone', currentLanguage)}
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
        title={editingZone ? t('authZones.editZone', currentLanguage) : t('authZones.addNewZone', currentLanguage)}
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
                label: t('authZones.basic', currentLanguage),
                children: (
                  <>
                    <Form.Item
                      name="domain"
                      label={t('authZones.domain', currentLanguage)}
                      rules={[
                        { required: true, message: t('authZones.domainRequired', currentLanguage) },
                        { type: 'string', max: 255, message: t('authZones.domainMax', currentLanguage) }
                      ]}
                    >
                      <Input placeholder={t('authZones.domainPlaceholder', currentLanguage)} />
                    </Form.Item>

                    <Form.Item
                      name="type"
                      label={t('authZones.type', currentLanguage)}
                      rules={[{ required: true, message: t('authZones.typeRequired', currentLanguage) }]}
                    >
                      <Input placeholder={t('authZones.typePlaceholder', currentLanguage)} />
                    </Form.Item>

                    <Form.Item
                      name="allow_query"
                      label={t('authZones.allowQuery', currentLanguage)}
                    >
                      <Input placeholder={t('authZones.allowQueryPlaceholder', currentLanguage)} />
                    </Form.Item>

                    <Form.Item
                      name="comment"
                      label={t('authZones.comment', currentLanguage)}
                    >
                      <Input.TextArea 
                        rows={4} 
                        placeholder={t('authZones.commentPlaceholder', currentLanguage)} 
                      />
                    </Form.Item>

                  </>
                )
              },
              {
                key: 'soa',
                label: t('authZones.soaRecord', currentLanguage),
                children: (
                  <Card title={t('authZones.soaRecord', currentLanguage)} size="small">
                    <Form.Item
                      name={['soa', 'primary_ns']}
                      label={t('authZones.primaryNs', currentLanguage)}
                      rules={[{ required: true, message: t('authZones.primaryNsRequired', currentLanguage) }]}
                    >
                      <Input placeholder={t('authZones.primaryNsPlaceholder', currentLanguage)} />
                    </Form.Item>

                    <Form.Item
                      name={['soa', 'admin_email']}
                      label={t('authZones.adminEmail', currentLanguage)}
                      rules={[{ required: true, message: t('authZones.adminEmailRequired', currentLanguage) }]}
                    >
                      <Input placeholder={t('authZones.adminEmailPlaceholder', currentLanguage)} />
                    </Form.Item>

                    <Row gutter={16}>
                      <Col xs={12}>
                        <Form.Item
                          name={['soa', 'refresh']}
                          label={t('authZones.refresh', currentLanguage)}
                          rules={[{ required: true, message: t('authZones.refreshRequired', currentLanguage) }]}
                        >
                          <Input placeholder={t('authZones.refreshPlaceholder', currentLanguage)} />
                        </Form.Item>
                      </Col>
                      <Col xs={12}>
                        <Form.Item
                          name={['soa', 'retry']}
                          label={t('authZones.retry', currentLanguage)}
                          rules={[{ required: true, message: t('authZones.retryRequired', currentLanguage) }]}
                        >
                          <Input placeholder={t('authZones.retryPlaceholder', currentLanguage)} />
                        </Form.Item>
                      </Col>
                      <Col xs={12}>
                        <Form.Item
                          name={['soa', 'expire']}
                          label={t('authZones.expire', currentLanguage)}
                          rules={[{ required: true, message: t('authZones.expireRequired', currentLanguage) }]}
                        >
                          <Input placeholder={t('authZones.expirePlaceholder', currentLanguage)} />
                        </Form.Item>
                      </Col>
                      <Col xs={12}>
                        <Form.Item
                          name={['soa', 'minimum_ttl']}
                          label={t('authZones.minimumTtl', currentLanguage)}
                          rules={[{ required: true, message: t('authZones.minimumTtlRequired', currentLanguage) }]}
                        >
                          <Input placeholder={t('authZones.minimumTtlPlaceholder', currentLanguage)} />
                        </Form.Item>
                      </Col>
                    </Row>
                  </Card>
                )
              },
              {
                key: 'records',
                label: t('authZones.records', currentLanguage),
                children: (
                  <>
                    <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'flex-end' }}>
                      <Button
                        type="primary"
                        icon={<PlusOutlined />}
                        onClick={() => showRecordModal()}
                      >
                        {t('authZones.addRecord', currentLanguage)}
                      </Button>
                    </div>

                    <Card title={t('authZones.recordsList', currentLanguage)} size="small">
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
                            title: t('authZones.name', currentLanguage),
                            dataIndex: 'name',
                            key: 'name',
                            width: 150,
                            ellipsis: true
                          },
                          {
                            title: t('authZones.type', currentLanguage),
                            dataIndex: 'type',
                            key: 'type',
                            width: 100
                          },
                          {
                            title: t('authZones.value', currentLanguage),
                            dataIndex: 'value',
                            key: 'value',
                            width: 200,
                            ellipsis: true
                          },
                          {
                            title: t('authZones.priority', currentLanguage),
                            dataIndex: 'priority',
                            key: 'priority',
                            width: 100
                          },
                          {
                            title: t('authZones.ttl', currentLanguage),
                            dataIndex: 'ttl',
                            key: 'ttl',
                            width: 100
                          },
                          {
                            title: t('authZones.comment', currentLanguage),
                            dataIndex: 'comment',
                            key: 'comment',
                            width: 300,
                            ellipsis: true
                          },
                          {
                            title: t('authZones.actions', currentLanguage),
                            key: 'actions',
                            width: 80,
                            render: (_, record) => (
                              <Space size="middle">
                                <Tooltip title={t('authZones.edit', currentLanguage)}>
                                  <Button
                                    type="link"
                                    size="small"
                                    icon={<EditOutlined />}
                                    onClick={() => showRecordModal(record)}
                                  />
                                </Tooltip>
                                <Tooltip title={t('authZones.delete', currentLanguage)}>
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
        title={editingRecord ? t('authZones.editRecord', currentLanguage) : t('authZones.addRecord', currentLanguage)}
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
            label={t('authZones.name', currentLanguage)}
            rules={[{ required: true, message: t('authZones.nameRequired', currentLanguage) }]}
          >
            <Input placeholder={t('authZones.namePlaceholder', currentLanguage)} />
          </Form.Item>

          <Form.Item
            name="type"
            label={t('authZones.type', currentLanguage)}
            rules={[{ required: true, message: t('authZones.typeRequired', currentLanguage) }]}
          >
            <Select
              value={selectedRecordType}
              onChange={handleRecordTypeChange}
              placeholder={t('authZones.typePlaceholder', currentLanguage)}
              allowClear={false}
            >
              {standardRecordTypes.map(type => (
                <Select.Option key={type} value={type}>
                  {type}
                </Select.Option>
              ))}
              <Select.Option value="custom">{t('authZones.customType', currentLanguage)}</Select.Option>
            </Select>
          </Form.Item>

          {/* 自定义记录类型输入框 */}
          {selectedRecordType === 'custom' && (
            <Form.Item
              name="customType"
              label={t('authZones.customType', currentLanguage)}
              rules={[{ required: true, message: t('authZones.customTypeRequired', currentLanguage) }]}
            >
              <Input placeholder={t('authZones.customTypePlaceholder', currentLanguage)} />
            </Form.Item>
          )}

          <Form.Item
            name="value"
            label={t('authZones.value', currentLanguage)}
            rules={[{ required: true, message: t('authZones.valueRequired', currentLanguage) }]}
          >
            <Input.TextArea
              rows={3}
              placeholder={t('authZones.valuePlaceholder', currentLanguage)}
            />
          </Form.Item>

          {/* MX记录优先级 */}
          {selectedRecordType === 'MX' && (
            <Form.Item
              name="priority"
              label={t('authZones.priority', currentLanguage)}
              rules={[{ required: true, message: t('authZones.priorityRequired', currentLanguage) }]}
            >
              <InputNumber
                min={0}
                max={65535}
                placeholder={t('authZones.priorityPlaceholder', currentLanguage)}
                style={{ width: '100%' }}
              />
            </Form.Item>
          )}

          <Form.Item
            name="ttl"
            label={t('authZones.ttl', currentLanguage)}
            rules={[{ required: true, message: t('authZones.ttlRequired', currentLanguage) }]}
          >
            <InputNumber
              min={0}
              max={2147483647}
              placeholder={t('authZones.ttlPlaceholder', currentLanguage)}
              style={{ width: '100%' }}
            />
          </Form.Item>

          <Form.Item
            name="comment"
            label={t('authZones.comment', currentLanguage)}
          >
            <Input.TextArea
              rows={2}
              placeholder={t('authZones.commentPlaceholder', currentLanguage)}
            />
          </Form.Item>
        </Form>
      </Modal>
      
      {/* Operation History Modal */}
      <Modal
        title="权威域操作历史"
        open={historyModalVisible}
        onOk={() => setHistoryModalVisible(false)}
        onCancel={() => setHistoryModalVisible(false)}
        okText="关闭"
        cancelText="取消"
        width={1000}
        styles={{ body: { maxHeight: 600, overflow: 'auto' } }}
      >
        <div>
          <h3 style={{ marginBottom: 16 }}>操作历史记录</h3>
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
                    title: 'ID',
                    dataIndex: 'id',
                    key: 'id',
                    width: 80
                  },
                  {
                    title: '操作类型',
                    dataIndex: 'operation',
                    key: 'operation',
                    width: 120,
                    render: (operation) => {
                      const operationMap = {
                        create: '创建',
                        update: '更新',
                        delete: '删除',
                        rollback: '回滚操作'
                      }
                      return operationMap[operation] || operation
                    }
                  },
                  {
                    title: '域名',
                    dataIndex: 'domain',
                    key: 'domain',
                    width: 200
                  },
                  {
                    title: '操作时间',
                    dataIndex: 'timestamp',
                    key: 'timestamp',
                    width: 200,
                    render: (timestamp) => {
                      return new Date(timestamp).toLocaleString()
                    }
                  },
                  {
                    title: '操作',
                    key: 'actions',
                    width: 100,
                    render: (_, record) => (
                      <Button
                        type="primary"
                        size="small"
                        onClick={() => confirmRestoreFromHistory(record.id)}
                      >
                        恢复
                      </Button>
                    )
                  }
                ]}
              />
            ) : (
              <Alert
                title="没有操作历史记录"
                description="当前没有权威域的操作历史记录"
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
        okText="确认"
        cancelText="取消"
      >
        <div>
          <Alert
            title="操作确认"
            description={confirmModalContent}
            type="warning"
            showIcon
            style={{ marginBottom: 16 }}
          />
          <p>请确认是否执行此操作。</p>
        </div>
      </Modal>
    </div>
  )
}

export default AuthZones
