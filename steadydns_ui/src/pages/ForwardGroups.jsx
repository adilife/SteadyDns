import { useState, useEffect, useCallback } from 'react'
import {
  Table,
  Button,
  Modal,
  Form,
  Input,
  Select,
  message,
  Space,
  Popconfirm,
  Tooltip,
  Spin,
  InputNumber,
  Card,
  Switch,
  Row,
  Col
} from 'antd'
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SaveOutlined,
  BarChartOutlined
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { apiClient } from '../utils/apiClient'

const { Option } = Select

const ForwardGroups = () => {
  const { t } = useTranslation()
  const [forwardGroups, setForwardGroups] = useState([])
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editingGroup, setEditingGroup] = useState(null)
  const [form] = Form.useForm()
  const [serverForm] = Form.useForm()
  const [apiLoading, setApiLoading] = useState(false)
  const [isServerModalOpen, setIsServerModalOpen] = useState(false)
  const [editingServer, setEditingServer] = useState(null)
  const [selectedGroupId, setSelectedGroupId] = useState(null)
  const [serverHealth, setServerHealth] = useState({})

  // Load forward groups from API
  const loadForwardGroups = useCallback(async () => {
    setApiLoading(true)
    try {
      const response = await apiClient.get('/forward-groups')
      
      if (response.success) {
        // Normalize data from snake_case to camelCase
        const normalizedGroups = response.data.map(group => ({
          ID: group.id,
          Domain: group.domain,
          Description: group.description,
          Enable: group.enable,
          CreatedAt: group.created_at,
          UpdatedAt: group.updated_at,
          Servers: (group.servers || []).map(server => ({
            ID: server.id,
            GroupID: server.group_id,
            Address: server.address,
            Port: server.port,
            Description: server.description,
            QueueIndex: server.queue_index,
            Priority: server.priority
          }))
        }))
        
        // Sort groups by ID
        normalizedGroups.sort((a, b) => a.ID - b.ID)
        
        setForwardGroups(normalizedGroups)
      } else {
        message.error(response.message || t('forwardGroups.fetchError'))
      }
    } catch (error) {
      console.error('Error loading forward groups:', error)
      // Error already handled by apiClient
    } finally {
      setApiLoading(false)
    }
  }, [t])

  // Load forward groups on component mount
  useEffect(() => {
    loadForwardGroups()
  }, [loadForwardGroups])

  // Show modal for adding/editing forward group
  const showModal = (group = null) => {
    setEditingGroup(group)
    if (group) {
      form.setFieldsValue(group)
    } else {
      form.resetFields()
    }
    setIsModalOpen(true)
  }

  // Cancel modal
  const handleCancel = () => {
    setIsModalOpen(false)
    setEditingGroup(null)
    form.resetFields()
  }

  // Submit form for adding/editing forward group
  const handleOk = () => {
    form.validateFields().then(values => {
      setApiLoading(true)
      if (editingGroup) {
        updateForwardGroup(editingGroup.ID, values)
      } else {
        createForwardGroup(values)
      }
    }).catch(() => {
      message.error(t('forwardGroups.formValidateError'))
      setApiLoading(false)
    })
  }

  // Create forward group
  const createForwardGroup = async (values) => {
    try {
      const response = await apiClient.post('/forward-groups', values)
      
      if (response.success) {
        message.success(response.message || t('forwardGroups.createSuccess'))
        loadForwardGroups()
        setIsModalOpen(false)
        setEditingGroup(null)
        form.resetFields()
      } else {
        message.error(response.message || t('forwardGroups.createError'))
      }
    } catch (error) {
      console.error('Error creating forward group:', error)
      // Error already handled by apiClient
    } finally {
      setApiLoading(false)
    }
  }

  // Update forward group
  const updateForwardGroup = async (id, values) => {
    try {
      const response = await apiClient.put(`/forward-groups/${id}`, values)
      
      if (response.success) {
        message.success(response.message || t('forwardGroups.updateSuccess'))
        loadForwardGroups()
        setIsModalOpen(false)
        setEditingGroup(null)
        form.resetFields()
      } else {
        message.error(response.message || t('forwardGroups.updateError'))
      }
    } catch (error) {
      console.error('Error updating forward group:', error)
      // Error already handled by apiClient
    } finally {
      setApiLoading(false)
    }
  }

  // Show modal for adding/editing server
  const showServerModal = (server = null, groupId = null) => {
    setEditingServer(server)
    setSelectedGroupId(groupId)
    if (server) {
      serverForm.setFieldsValue(server)
    } else {
      serverForm.resetFields()
    }
    setIsServerModalOpen(true)
  }

  // Cancel server modal
  const handleServerCancel = () => {
    setIsServerModalOpen(false)
    setEditingServer(null)
    setSelectedGroupId(null)
    serverForm.resetFields()
  }

  // Submit server form
  const handleServerOk = () => {
    serverForm.validateFields().then(values => {
      setApiLoading(true)
      if (editingServer) {
        updateServer(editingServer.ID, values)
      } else {
        addServer(values)
      }
    }).catch(() => {
      message.error(t('forwardGroups.formValidateError'))
      setApiLoading(false)
    })
  }

  // Add server
  const addServer = async (values) => {
    try {
      // Use Priority as QueueIndex
      const serverData = {
        ...values,
        group_id: selectedGroupId,
        queue_index: values.Priority
      }
      const response = await apiClient.post('/forward-servers?batch=true', [serverData])
      
      if (response.success) {
        message.success(response.message || t('forwardGroups.serverAdded'))
        loadForwardGroups()
      } else {
        message.error(response.message || t('forwardGroups.serverAddError'))
      }
    } catch (error) {
      console.error('Error adding server:', error)
      // Error already handled by apiClient
    } finally {
      setIsServerModalOpen(false)
      setEditingServer(null)
      setSelectedGroupId(null)
      serverForm.resetFields()
      setApiLoading(false)
    }
  }

  // Update server
  const updateServer = async (id, values) => {
    try {
      const serverData = {
        ...values,
        group_id: selectedGroupId,
        queue_index: values.Priority
      }
      const response = await apiClient.put(`/forward-servers/${id}`, serverData)
      
      if (response.success) {
        message.success(response.message || t('forwardGroups.serverUpdated'))
        loadForwardGroups()
      } else {
        message.error(response.message || t('forwardGroups.serverUpdateError'))
      }
    } catch (error) {
      console.error('Error updating server:', error)
      // Error already handled by apiClient
    } finally {
      setIsServerModalOpen(false)
      setEditingServer(null)
      setSelectedGroupId(null)
      serverForm.resetFields()
      setApiLoading(false)
    }
  }

  // Delete server
  const handleDeleteServer = async (serverId) => {
    try {
      const response = await apiClient.delete(`/forward-servers/${serverId}`)
      
      if (response.success) {
        message.success(response.message || t('forwardGroups.serverDeleted'))
        loadForwardGroups()
      } else {
        message.error(response.message || t('forwardGroups.serverDeleteError'))
      }
    } catch (error) {
      console.error('Error deleting server:', error)
      // Error already handled by apiClient
    }
  }

  // Check server health
  const checkServerHealth = useCallback(async (serverId) => {
    try {
      const response = await apiClient.get(`/forward-servers/${serverId}?health=true`)
      
      if (response.success) {
        setServerHealth(prev => ({
          ...prev,
          [serverId]: response.data
        }))
      } else {
        message.error(response.message || t('forwardGroups.healthCheckFailed'))
      }
    } catch (error) {
      console.error('Error checking server health:', error)
      // Error already handled by apiClient
    }
  }, [t])

  // Check health for all servers
  const checkAllServersHealth = useCallback(() => {
    forwardGroups.forEach(group => {
      group.Servers.forEach(server => {
        checkServerHealth(server.ID)
      })
    })
  }, [forwardGroups, checkServerHealth])

  // Auto check health every minute
  useEffect(() => {
    // Check health on mount
    if (forwardGroups.length > 0) {
      checkAllServersHealth()
    }

    // Set up interval for auto health check
    const healthCheckInterval = setInterval(() => {
      if (forwardGroups.length > 0) {
        checkAllServersHealth()
      }
    }, 60000) // 1 minute interval

    return () => {
      clearInterval(healthCheckInterval)
    }
  }, [forwardGroups, checkAllServersHealth])

  // Delete forward group
  const handleDelete = async (id) => {
    // Prevent deletion of default forward group (ID=1)
    if (id === 1) {
      message.warning(t('forwardGroups.defaultGroupCannotDelete'))
      return
    }
    
    setApiLoading(true)
    try {
      const response = await apiClient.delete(`/forward-groups/${id}`)
      
      if (response.success) {
        message.success(response.message || t('forwardGroups.deleteSuccess'))
        loadForwardGroups()
      } else {
        message.error(response.message || t('forwardGroups.deleteError'))
      }
    } catch (error) {
      console.error('Error deleting forward group:', error)
      // Error already handled by apiClient
    } finally {
      setApiLoading(false)
    }
  }

  // Batch delete forward groups
  const handleBatchDelete = async () => {
    // Filter out default group (ID=1)
    const deletableGroups = selectedGroupKeys.filter(id => id !== 1)
    
    if (deletableGroups.length === 0) {
      message.warning(t('forwardGroups.selectToDelete'))
      return
    }
    
    setApiLoading(true)
    try {
      // Use batchOperation method for batch deletion
      await apiClient.batchOperation('/forward-groups?batch=true', deletableGroups, 'DELETE')
      
      message.success(t('forwardGroups.batchDeleteSuccess'))
      loadForwardGroups()
      setSelectedGroupKeys([])
    } catch (error) {
      console.error('Error batch deleting forward groups:', error)
      message.error(t('forwardGroups.batchDeleteError'))
    } finally {
      setApiLoading(false)
    }
  }
  // Table columns
  const columns = [
    {
      title: t('forwardGroups.index'),
      key: 'index',
      width: 60,
      render: (_, __, index) => index + 1
    },
    {
      title: t('forwardGroups.domain'),
      dataIndex: 'Domain',
      key: 'Domain',
      ellipsis: true,
      render: (text, record) => (
        <span style={{ 
          textDecoration: record.Enable ? 'none' : 'line-through',
          color: record.Enable ? 'inherit' : '#999'
        }}>
          {record.ID === 1 && (
            <span style={{ 
              backgroundColor: '#52c41a',
              color: '#fff',
              padding: '2px 8px',
              borderRadius: '10px',
              fontSize: '12px',
              marginRight: '8px',
              fontWeight: 'bold'
            }}>
              {t('forwardGroups.default')}
            </span>
          )}
          {text}
        </span>
      )
    },
    {
      title: t('forwardGroups.description'),
      dataIndex: 'Description',
      key: 'Description',
      ellipsis: true
    },
    {
      title: t('forwardGroups.enableStatus'),
      dataIndex: 'Enable',
      key: 'Enable',
      width: 100,
      render: (enable) => (
        <span style={{ 
          padding: '2px 8px',
          borderRadius: '10px',
          fontSize: '12px',
          backgroundColor: enable ? '#f6ffed' : '#fff2f0',
          color: enable ? '#52c41a' : '#ff4d4f'
        }}>
          {enable ? t('forwardGroups.enabled') : t('forwardGroups.disabled')}
        </span>
      )
    },
    {
      title: t('forwardGroups.servers'),
      dataIndex: 'Servers',
      key: 'Servers',
      ellipsis: true,
      render: (servers, record) => {
        if (!servers || servers.length === 0) {
          return (
            <Space>
              <span>{t('forwardGroups.noServers')}</span>
              <Button
                type="link"
                icon={<PlusOutlined />}
                size="small"
                onClick={() => showServerModal(null, record.ID)}
              >
                {t('forwardGroups.add')}
              </Button>
            </Space>
          )
        }
        return (
          <div>
            <div style={{ marginBottom: 8 }}>
              {servers.length > 0 ? (
                <div>
                  {/* Group servers by priority */}
                  {[1, 2, 3].map(priority => {
                    const priorityServers = servers.filter(server => server.Priority === priority)
                    if (priorityServers.length === 0) return null
                    
                    const priorityText = priority === 1 ? t('forwardGroups.priorityHigh') : 
                                        priority === 2 ? t('forwardGroups.priorityMedium') : 
                                        t('forwardGroups.priorityLow')
                    
                    return (
                      <div key={priority} style={{ marginBottom: 12 }}>
                        <div style={{ 
                          fontSize: '14px', 
                          fontWeight: 'bold', 
                          marginBottom: 8,
                          color: priority === 1 ? '#ff4d4f' : priority === 2 ? '#faad14' : '#1890ff'
                        }}>
                          {t('forwardGroups.priority')} {priorityText} ({priorityServers.length} {t('forwardGroups.serversCount')})
                        </div>
                        <div style={{ marginLeft: 16 }}>
                          {priorityServers.map(server => {
                            const health = serverHealth[server.ID]
                            const isHealthy = health?.is_healthy
                            return (
                              <div key={server.ID} style={{ 
                                marginBottom: 6, 
                                padding: 12, 
                                backgroundColor: '#fafafa', 
                                borderRadius: 4,
                                display: 'flex',
                                justifyContent: 'space-between',
                                alignItems: 'center'
                              }}>
                                <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                                  <span style={{ fontWeight: 'bold' }}>
                                    {server.Address}:{server.Port || 53}
                                  </span>
                                  {server.Description && (
                                    <span style={{ fontSize: '12px', color: '#999' }}>
                                      - {server.Description}
                                    </span>
                                  )}
                                </div>
                                <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                                  {/* Status column */}
                                  {isHealthy !== undefined && (
                                    <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                                      <span style={{ 
                                        width: 12, 
                                        height: 12, 
                                        borderRadius: '50%',
                                        backgroundColor: !isHealthy || health?.response_time > 100 ? '#ff4d4f' : 
                                                         health?.response_time >= 50 ? '#faad14' : '#52c41a'
                                      }}></span>
                                      <span style={{ fontSize: '12px', color: '#666', width: 50, textAlign: 'left' }}>
                                        {health?.response_time > 500 ? ">500ms" : `${health?.response_time}ms`}
                                      </span>
                                    </div>
                                  )}
                                  {/* Action buttons */}
                                  <div style={{ display: 'flex', gap: 8 }}>
                                    <Button
                                      type="link"
                                      size="small"
                                      icon={<EditOutlined />}
                                      onClick={() => showServerModal(server, record.ID)}
                                    />
                                    <Button
                                      type="link"
                                      size="small"
                                      onClick={() => checkServerHealth(server.ID)}
                                    >
                                      {t('forwardGroups.check')}
                                    </Button>
                                    <Popconfirm
                                      title={t('forwardGroups.confirmDeleteServer')}
                                      onConfirm={() => handleDeleteServer(server.ID)}
                                      okText={t('forwardGroups.yes')}
                                      cancelText={t('forwardGroups.no')}
                                    >
                                      <Button
                                        type="link"
                                        size="small"
                                        danger
                                        icon={<DeleteOutlined />}
                                      />
                                    </Popconfirm>
                                  </div>
                                </div>
                              </div>
                            )
                          })}
                        </div>
                      </div>
                    )
                  })}
                </div>
              ) : (
                <Space>
                  <span>{t('forwardGroups.noServers')}</span>
                  <Button
                    type="link"
                    icon={<PlusOutlined />}
                    size="small"
                    onClick={() => showServerModal(null, record.ID)}
                  >
                    {t('forwardGroups.add')}
                  </Button>
                </Space>
              )}
            </div>
            <Button
              type="link"
              icon={<PlusOutlined />}
              size="small"
              onClick={() => showServerModal(null, record.ID)}
            >
              {t('forwardGroups.addServer')}
            </Button>
          </div>
        )
      }
    },
    {
      title: t('forwardGroups.actions'),
      key: 'actions',
      width: 150,
      render: (_, record) => (
        <Space size="middle">
          <Tooltip title={t('forwardGroups.edit')}>
            <Button
              icon={<EditOutlined />}
              size="small"
              onClick={() => showModal(record)}
            />
          </Tooltip>
          <Tooltip title={record.ID === 1 ? t('forwardGroups.defaultGroupCannotDelete') : t('forwardGroups.delete')}>
            <Popconfirm
              title={t('forwardGroups.confirmDelete')}
              onConfirm={() => handleDelete(record.ID)}
              okText={t('forwardGroups.yes')}
              cancelText={t('forwardGroups.no')}
            >
              <Button
                icon={<DeleteOutlined />}
                size="small"
                danger
                disabled={record.ID === 1}
              />
            </Popconfirm>
          </Tooltip>
        </Space>
      ),
    },
  ]

  // State for domain match test
  const [testDomain, setTestDomain] = useState('')
  const [testResult, setTestResult] = useState(null)
  const [testLoading, setTestLoading] = useState(false)
  const [selectedGroupKeys, setSelectedGroupKeys] = useState([])
  
  // Test domain match
  const handleTestDomainMatch = async () => {
    if (!testDomain.trim()) {
      message.warning(t('forwardGroups.enterTestDomain'))
      return
    }
    
    setTestLoading(true)
    try {
      const response = await apiClient.testDomainMatch(testDomain)
      if (response.success) {
        setTestResult(response.data)
      } else {
        message.error(response.message || t('forwardGroups.testFailed'))
        setTestResult(null)
      }
    } catch (error) {
      console.error('Error testing domain match:', error)
      message.error(t('forwardGroups.testFailedRetry'))
      setTestResult(null)
    } finally {
      setTestLoading(false)
    }
  }

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2>{t('forwardGroups.title')}</h2>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => showModal()}
        >
          {t('forwardGroups.addGroup')}
        </Button>
      </div>



      <Spin spinning={apiLoading}>
        <div style={{ marginBottom: 16 }}>
          <Button
            type="danger"
            icon={<DeleteOutlined />}
            onClick={handleBatchDelete}
            disabled={selectedGroupKeys.length === 0}
          >
            {t('forwardGroups.batchDelete')}
          </Button>
          <span style={{ marginLeft: 16 }}>
            {selectedGroupKeys.length > 0 ? 
              t('forwardGroups.selectedCount', { count: selectedGroupKeys.length }) : 
              ''
            }
          </span>
        </div>
        <Table
          columns={columns}
          dataSource={forwardGroups}
          rowKey="ID"
          pagination={{
            showSizeChanger: true,
            pageSizeOptions: ['10', '20', '50'],
            defaultPageSize: 10
          }}
          scroll={{ x: 'max-content' }}
          rowSelection={{
            selectedRowKeys: selectedGroupKeys,
            onChange: (keys) => setSelectedGroupKeys(keys),
            getCheckboxProps: (record) => ({
              disabled: record.ID === 1, // Disable checkbox for default group
            }),
          }}
        />
      </Spin>

      <Modal
        title={editingGroup ? t('forwardGroups.editGroup') : t('forwardGroups.addNewGroup')}
        open={isModalOpen}
        onOk={handleOk}
        onCancel={handleCancel}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            Domain: '',
            Description: '',
            Enable: true
          }}
        >
          <Form.Item
            name="Domain"
            label={t('forwardGroups.domain')}
            rules={[
              { 
                required: true, 
                message: t('forwardGroups.domainValidation.required')
              },
              { 
                max: 255, 
                message: t('forwardGroups.domainValidation.maxLength')
              },
              {
                validator: (_, value) => {
                  // Skip validation for default group (ID=1) as it's disabled
                  if (editingGroup && editingGroup.ID === 1) {
                    return Promise.resolve();
                  }
                  
                  // Basic domain format validation
                  const domainRegex = /^([a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$/;
                  if (!domainRegex.test(value)) {
                    return Promise.reject(new Error(t('forwardGroups.domainValidation.invalidFormat')));
                  }
                  
                  // Check each label length
                  const labels = value.split('.');
                  for (const label of labels) {
                    if (label.length < 1 || label.length > 63) {
                      return Promise.reject(new Error(t('forwardGroups.domainValidation.labelLength')));
                    }
                    if (!/^[a-zA-Z0-9]/.test(label) || !/[a-zA-Z0-9]$/.test(label)) {
                      return Promise.reject(new Error(t('forwardGroups.domainValidation.hyphenStartEnd')));
                    }
                  }
                  
                  return Promise.resolve();
                }
              }
            ]}
            tooltip={editingGroup && editingGroup.ID === 1 ? t('forwardGroups.defaultGroupDomainLocked') : ''}
          >
            <Input 
              placeholder={t('forwardGroups.domainPlaceholder')} 
              disabled={editingGroup && editingGroup.ID === 1}
            />
          </Form.Item>

          <Form.Item
            name="Description"
            label={t('forwardGroups.description')}
            tooltip={editingGroup && editingGroup.ID === 1 ? t('forwardGroups.defaultGroupDescriptionLocked') : ''}
          >
            <Input.TextArea 
              rows={3} 
              placeholder={t('forwardGroups.descriptionPlaceholder')} 
              disabled={editingGroup && editingGroup.ID === 1}
            />
          </Form.Item>

          <Form.Item
            name="Enable"
            label={t('forwardGroups.enableStatus')}
            valuePropName="checked"
          >
            <Switch defaultChecked />
          </Form.Item>

          {/* Server configuration can be added here if needed */}
        </Form>
      </Modal>

      {/* Server Management Modal */}
      <Modal
        title={editingServer ? t('forwardGroups.editServer') : t('forwardGroups.addServer')}
        open={isServerModalOpen}
        onOk={handleServerOk}
        onCancel={handleServerCancel}
        width={600}
      >
        <Form
          form={serverForm}
          layout="vertical"
          initialValues={{
            Address: '',
            Port: 53,
            Description: '',
            Priority: 1
          }}
        >
          <Form.Item
            name="Address"
            label={t('forwardGroups.serverAddress')}
            rules={[{ required: true, message: t('forwardGroups.inputServerAddress') }]}
          >
            <Input placeholder={t('forwardGroups.serverAddressPlaceholder')} />
          </Form.Item>

          <Form.Item
            name="Port"
            label={t('forwardGroups.port')}
            rules={[{ required: true, message: t('forwardGroups.inputPort') }]}
          >
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="Description"
            label={t('forwardGroups.serverDescription')}
          >
            <Input.TextArea rows={3} placeholder={t('forwardGroups.serverDescriptionPlaceholder')} />
          </Form.Item>

          <Form.Item
            name="Priority"
            label={t('forwardGroups.priorityLabel')}
            rules={[{ required: true, message: t('forwardGroups.priorityPlaceholder') }]}
          >
            <Select style={{ width: '100%' }}>
              <Option value={1}>{t('forwardGroups.priorityHighOption')}</Option>
              <Option value={2}>{t('forwardGroups.priorityMediumOption')}</Option>
              <Option value={3}>{t('forwardGroups.priorityLowOption')}</Option>
            </Select>
          </Form.Item>


        </Form>
      </Modal>

      {/* Domain Match Test Section */}
      <Card title={<Space><BarChartOutlined />{t('forwardGroups.domainTest')}</Space>} style={{ marginBottom: 24, marginTop: 24 }}>
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={16}>
            <Input 
              placeholder={t('forwardGroups.testDomainPlaceholder')}
              value={testDomain}
              onChange={(e) => setTestDomain(e.target.value)}
              onPressEnter={handleTestDomainMatch}
              style={{ marginRight: 8 }}
            />
          </Col>
          <Col xs={24} sm={8}>
            <Space>
              <Button 
                type="primary" 
                onClick={handleTestDomainMatch}
                loading={testLoading}
              >
                {t('forwardGroups.testMatch')}
              </Button>
              <Button 
                onClick={() => {
                  setTestDomain('')
                  setTestResult(null)
                }}
              >
                {t('forwardGroups.reset')}
              </Button>
            </Space>
          </Col>
          {testResult && (
            <Col xs={24}>
              <div style={{ marginTop: 16, padding: 12, backgroundColor: '#f6ffed', borderRadius: 4 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                  <strong>{t('forwardGroups.testResult')}:</strong>
                  <span style={{ color: '#52c41a', fontWeight: 'bold' }}>
                    {testResult.matched_group ? testResult.matched_group : t('forwardGroups.noMatch')}
                  </span>
                </div>
                <div style={{ fontSize: '14px', color: '#666' }}>
                  {t('forwardGroups.matchedGroup')}: 
                  <strong>{testResult.matched_group || t('forwardGroups.noMatch')}</strong>
                </div>
              </div>
            </Col>
          )}
        </Row>
      </Card>
    </div>
  )
}

export default ForwardGroups