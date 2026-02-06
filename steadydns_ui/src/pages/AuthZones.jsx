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
  InputNumber
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
      const response = await apiClient.get('/bind-zones/')
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

  // 当编辑区域变化时，更新记录列表
  useEffect(() => {
    if (editingZone && editingZone.records) {
      // 直接使用后端返回的记录，不再生成前端临时ID
      setRecords(editingZone.records);
    } else {
      setRecords([])
    }
  }, [editingZone])

  // 提交表单
  const handleOk = () => {
    form.validateFields().then(values => {
      setLoading(true)
      // 将记录数据添加到提交的数据中
      const submitData = {
        ...values,
        records: records // 添加服务器记录
      }
      if (editingZone) {
        updateAuthZone(editingZone.domain, submitData)
      } else {
        createAuthZone(submitData)
      }
    }).catch(() => {
      message.error(t('authZones.formValidateError', currentLanguage))
      setLoading(false)
    })
  }

  // 创建权威域
  const createAuthZone = async (values) => {
    try {
      const response = await apiClient.post('/bind-zones/', values)
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
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => showModal()}
        >
          {t('authZones.addZone', currentLanguage)}
        </Button>
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
        width={800}
        destroyOnHidden
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{
          domain: '',
          type: 'master',
          allow_query: 'any',
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
                        columns={[
                          {
                            title: t('authZones.name', currentLanguage),
                            dataIndex: 'name',
                            key: 'name',
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
        </Form>
      </Modal>
    </div>
  )
}

export default AuthZones
