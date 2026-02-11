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
import { t } from '../i18n'
import { apiClient } from '../utils/apiClient'

const { Option } = Select

const ForwardGroups = ({ currentLanguage }) => {
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
        message.error(response.message || (currentLanguage === 'zh-CN' ? '获取转发组失败' : 'Failed to get forward groups'))
      }
    } catch (error) {
      console.error('Error loading forward groups:', error)
      // Error already handled by apiClient
    } finally {
      setApiLoading(false)
    }
  }, [currentLanguage])

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
      message.error(currentLanguage === 'zh-CN' ? '请检查表单字段' : 'Please check form fields')
      setApiLoading(false)
    })
  }

  // Create forward group
  const createForwardGroup = async (values) => {
    try {
      const response = await apiClient.post('/forward-groups', values)
      
      if (response.success) {
        message.success(response.message || (currentLanguage === 'zh-CN' ? '转发组创建成功' : 'Forward group created successfully'))
        loadForwardGroups()
        setIsModalOpen(false)
        setEditingGroup(null)
        form.resetFields()
      } else {
        message.error(response.message || (currentLanguage === 'zh-CN' ? '创建转发组失败' : 'Failed to create forward group'))
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
        message.success(response.message || (currentLanguage === 'zh-CN' ? '转发组更新成功' : 'Forward group updated successfully'))
        loadForwardGroups()
        setIsModalOpen(false)
        setEditingGroup(null)
        form.resetFields()
      } else {
        message.error(response.message || (currentLanguage === 'zh-CN' ? '更新转发组失败' : 'Failed to update forward group'))
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
      message.error(currentLanguage === 'zh-CN' ? '请检查表单字段' : 'Please check form fields')
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
        message.success(response.message || (currentLanguage === 'zh-CN' ? '服务器添加成功' : 'Server added successfully'))
        loadForwardGroups()
      } else {
        message.error(response.message || (currentLanguage === 'zh-CN' ? '添加服务器失败' : 'Failed to add server'))
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
        message.success(response.message || (currentLanguage === 'zh-CN' ? '服务器更新成功' : 'Server updated successfully'))
        loadForwardGroups()
      } else {
        message.error(response.message || (currentLanguage === 'zh-CN' ? '更新服务器失败' : 'Failed to update server'))
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
        message.success(response.message || (currentLanguage === 'zh-CN' ? '服务器删除成功' : 'Server deleted successfully'))
        loadForwardGroups()
      } else {
        message.error(response.message || (currentLanguage === 'zh-CN' ? '删除服务器失败' : 'Failed to delete server'))
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
        message.error(response.message || (currentLanguage === 'zh-CN' ? '健康检查失败' : 'Health check failed'))
      }
    } catch (error) {
      console.error('Error checking server health:', error)
      // Error already handled by apiClient
    }
  }, [currentLanguage])

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
      message.warning(currentLanguage === 'zh-CN' ? '默认转发组不可删除' : 'Default forward group cannot be deleted')
      return
    }
    
    setApiLoading(true)
    try {
      const response = await apiClient.delete(`/forward-groups/${id}`)
      
      if (response.success) {
        message.success(response.message || (currentLanguage === 'zh-CN' ? '转发组删除成功' : 'Forward group deleted successfully'))
        loadForwardGroups()
      } else {
        message.error(response.message || (currentLanguage === 'zh-CN' ? '删除转发组失败' : 'Failed to delete forward group'))
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
      message.warning(currentLanguage === 'zh-CN' ? '请选择要删除的转发组' : 'Please select forward groups to delete')
      return
    }
    
    setApiLoading(true)
    try {
      // Use batchOperation method for batch deletion
      await apiClient.batchOperation('/forward-groups?batch=true', deletableGroups, 'DELETE')
      
      message.success(currentLanguage === 'zh-CN' ? '转发组批量删除成功' : 'Forward groups deleted successfully')
      loadForwardGroups()
      setSelectedGroupKeys([])
    } catch (error) {
      console.error('Error batch deleting forward groups:', error)
      message.error(currentLanguage === 'zh-CN' ? '批量删除转发组失败' : 'Failed to delete forward groups')
    } finally {
      setApiLoading(false)
    }
  }
  // Table columns
  const columns = [
    {
      title: currentLanguage === 'zh-CN' ? '序号' : 'No.',
      key: 'index',
      width: 60,
      render: (_, __, index) => index + 1
    },
    {
      title: t('forwardGroups.domain', currentLanguage),
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
              {t('forwardGroups.default', currentLanguage)}
            </span>
          )}
          {text}
        </span>
      )
    },
    {
      title: t('forwardGroups.description', currentLanguage),
      dataIndex: 'Description',
      key: 'Description',
      ellipsis: true
    },
    {
      title: currentLanguage === 'zh-CN' ? '启用状态' : 'Enable Status',
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
          {enable ? (currentLanguage === 'zh-CN' ? '启用' : 'Enabled') : (currentLanguage === 'zh-CN' ? '禁用' : 'Disabled')}
        </span>
      )
    },
    {
      title: t('forwardGroups.servers', currentLanguage),
      dataIndex: 'Servers',
      key: 'Servers',
      ellipsis: true,
      render: (servers, record) => {
        if (!servers || servers.length === 0) {
          return (
            <Space>
              <span>{t('forwardGroups.noServers', currentLanguage)}</span>
              <Button
                type="link"
                icon={<PlusOutlined />}
                size="small"
                onClick={() => showServerModal(null, record.ID)}
              >
                {currentLanguage === 'zh-CN' ? '添加' : 'Add'}
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
                    
                    const priorityText = priority === 1 ? (currentLanguage === 'zh-CN' ? '高' : 'High') : 
                                        priority === 2 ? (currentLanguage === 'zh-CN' ? '中' : 'Medium') : 
                                        (currentLanguage === 'zh-CN' ? '低' : 'Low')
                    
                    return (
                      <div key={priority} style={{ marginBottom: 12 }}>
                        <div style={{ 
                          fontSize: '14px', 
                          fontWeight: 'bold', 
                          marginBottom: 8,
                          color: priority === 1 ? '#ff4d4f' : priority === 2 ? '#faad14' : '#1890ff'
                        }}>
                          {currentLanguage === 'zh-CN' ? '优先级 ' : 'Priority '}{priorityText} ({priorityServers.length} {currentLanguage === 'zh-CN' ? '个服务器' : 'servers'})
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
                                      {currentLanguage === 'zh-CN' ? '检查' : 'Check'}
                                    </Button>
                                    <Popconfirm
                                      title={currentLanguage === 'zh-CN' ? '确定要删除此服务器吗？' : 'Are you sure to delete this server?'}
                                      onConfirm={() => handleDeleteServer(server.ID)}
                                      okText={t('forwardGroups.yes', currentLanguage)}
                                      cancelText={t('forwardGroups.no', currentLanguage)}
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
                  <span>{t('forwardGroups.noServers', currentLanguage)}</span>
                  <Button
                    type="link"
                    icon={<PlusOutlined />}
                    size="small"
                    onClick={() => showServerModal(null, record.ID)}
                  >
                    {currentLanguage === 'zh-CN' ? '添加' : 'Add'}
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
              {currentLanguage === 'zh-CN' ? '添加服务器' : 'Add Server'}
            </Button>
          </div>
        )
      }
    },
    {
      title: t('forwardGroups.actions', currentLanguage),
      key: 'actions',
      width: 150,
      render: (_, record) => (
        <Space size="middle">
          <Tooltip title={t('forwardGroups.edit', currentLanguage)}>
            <Button
              icon={<EditOutlined />}
              size="small"
              onClick={() => showModal(record)}
            />
          </Tooltip>
          <Tooltip title={record.ID === 1 ? (currentLanguage === 'zh-CN' ? '默认转发组不可删除' : 'Default group cannot be deleted') : t('forwardGroups.delete', currentLanguage)}>
            <Popconfirm
              title={t('forwardGroups.confirmDelete', currentLanguage)}
              onConfirm={() => handleDelete(record.ID)}
              okText={t('forwardGroups.yes', currentLanguage)}
              cancelText={t('forwardGroups.no', currentLanguage)}
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
      message.warning(currentLanguage === 'zh-CN' ? '请输入测试域名' : 'Please enter a test domain')
      return
    }
    
    setTestLoading(true)
    try {
      const response = await apiClient.testDomainMatch(testDomain)
      if (response.success) {
        setTestResult(response.data)
      } else {
        message.error(response.message || (currentLanguage === 'zh-CN' ? '测试失败' : 'Test failed'))
        setTestResult(null)
      }
    } catch (error) {
      console.error('Error testing domain match:', error)
      message.error(currentLanguage === 'zh-CN' ? '测试失败，请稍后重试' : 'Test failed, please try again later')
      setTestResult(null)
    } finally {
      setTestLoading(false)
    }
  }

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2>{t('forwardGroups.title', currentLanguage)}</h2>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => showModal()}
        >
          {t('forwardGroups.addGroup', currentLanguage)}
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
            {t('forwardGroups.batchDelete', currentLanguage)}
          </Button>
          <span style={{ marginLeft: 16 }}>
            {selectedGroupKeys.length > 0 ? 
              (currentLanguage === 'zh-CN' ? `已选择 ${selectedGroupKeys.length} 个转发组` : `Selected ${selectedGroupKeys.length} forward groups`) : 
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
        title={editingGroup ? t('forwardGroups.editGroup', currentLanguage) : t('forwardGroups.addNewGroup', currentLanguage)}
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
            label={t('forwardGroups.domain', currentLanguage)}
            rules={[
              { 
                required: true, 
                message: t('forwardGroups.domainValidation.required', currentLanguage) 
              },
              { 
                max: 255, 
                message: t('forwardGroups.domainValidation.maxLength', currentLanguage) 
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
                    return Promise.reject(new Error(t('forwardGroups.domainValidation.invalidFormat', currentLanguage)));
                  }
                  
                  // Check each label length
                  const labels = value.split('.');
                  for (const label of labels) {
                    if (label.length < 1 || label.length > 63) {
                      return Promise.reject(new Error(t('forwardGroups.domainValidation.labelLength', currentLanguage)));
                    }
                    if (!/^[a-zA-Z0-9]/.test(label) || !/[a-zA-Z0-9]$/.test(label)) {
                      return Promise.reject(new Error(t('forwardGroups.domainValidation.hyphenStartEnd', currentLanguage)));
                    }
                  }
                  
                  return Promise.resolve();
                }
              }
            ]}
            tooltip={editingGroup && editingGroup.ID === 1 ? t('forwardGroups.defaultGroupDomainLocked', currentLanguage) : ''}
          >
            <Input 
              placeholder={currentLanguage === 'zh-CN' ? '请输入转发组域名' : 'Please enter forward group domain'} 
              disabled={editingGroup && editingGroup.ID === 1}
            />
          </Form.Item>

          <Form.Item
            name="Description"
            label={t('forwardGroups.description', currentLanguage)}
            tooltip={editingGroup && editingGroup.ID === 1 ? t('forwardGroups.defaultGroupDescriptionLocked', currentLanguage) : ''}
          >
            <Input.TextArea 
              rows={3} 
              placeholder={currentLanguage === 'zh-CN' ? '请输入转发组描述' : 'Please enter forward group description'} 
              disabled={editingGroup && editingGroup.ID === 1}
            />
          </Form.Item>

          <Form.Item
            name="Enable"
            label={currentLanguage === 'zh-CN' ? '启用状态' : 'Enable Status'}
            valuePropName="checked"
          >
            <Switch defaultChecked />
          </Form.Item>

          {/* Server configuration can be added here if needed */}
        </Form>
      </Modal>

      {/* Server Management Modal */}
      <Modal
        title={editingServer ? (currentLanguage === 'zh-CN' ? '编辑服务器' : 'Edit Server') : (currentLanguage === 'zh-CN' ? '添加服务器' : 'Add Server')}
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
            label={currentLanguage === 'zh-CN' ? '服务器地址' : 'Server Address'}
            rules={[{ required: true, message: currentLanguage === 'zh-CN' ? '请输入服务器地址' : 'Please enter server address' }]}
          >
            <Input placeholder={currentLanguage === 'zh-CN' ? '请输入DNS服务器地址 (IPv4/IPv6)' : 'Please enter DNS server address (IPv4/IPv6)'} />
          </Form.Item>

          <Form.Item
            name="Port"
            label={currentLanguage === 'zh-CN' ? '端口' : 'Port'}
            rules={[{ required: true, message: currentLanguage === 'zh-CN' ? '请输入端口' : 'Please enter port' }]}
          >
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="Description"
            label={currentLanguage === 'zh-CN' ? '描述' : 'Description'}
          >
            <Input.TextArea rows={3} placeholder={currentLanguage === 'zh-CN' ? '请输入服务器描述' : 'Please enter server description'} />
          </Form.Item>

          <Form.Item
            name="Priority"
            label={currentLanguage === 'zh-CN' ? '优先级' : 'Priority'}
            rules={[{ required: true, message: currentLanguage === 'zh-CN' ? '请选择优先级' : 'Please select priority' }]}
          >
            <Select style={{ width: '100%' }}>
              <Option value={1}>{currentLanguage === 'zh-CN' ? '高 (1)' : 'High (1)'}</Option>
              <Option value={2}>{currentLanguage === 'zh-CN' ? '中 (2)' : 'Medium (2)'}</Option>
              <Option value={3}>{currentLanguage === 'zh-CN' ? '低 (3)' : 'Low (3)'}</Option>
            </Select>
          </Form.Item>


        </Form>
      </Modal>

      {/* Domain Match Test Section */}
      <Card title={<Space><BarChartOutlined />{t('forwardGroups.domainTest', currentLanguage)}</Space>} style={{ marginBottom: 24, marginTop: 24 }}>
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={16}>
            <Input 
              placeholder={currentLanguage === 'zh-CN' ? '请输入要测试的域名，例如: www.example.com' : 'Please enter a domain to test, e.g.: www.example.com'}
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
                {t('forwardGroups.testMatch', currentLanguage)}
              </Button>
              <Button 
                onClick={() => {
                  setTestDomain('')
                  setTestResult(null)
                }}
              >
                {currentLanguage === 'zh-CN' ? '重置' : 'Reset'}
              </Button>
            </Space>
          </Col>
          {testResult && (
            <Col xs={24}>
              <div style={{ marginTop: 16, padding: 12, backgroundColor: '#f6ffed', borderRadius: 4 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                  <strong>{t('forwardGroups.testResult', currentLanguage)}:</strong>
                  <span style={{ color: '#52c41a', fontWeight: 'bold' }}>
                    {testResult.matched_group ? testResult.matched_group : t('forwardGroups.noMatch', currentLanguage)}
                  </span>
                </div>
                <div style={{ fontSize: '14px', color: '#666' }}>
                  {t('forwardGroups.matchedGroup', currentLanguage)}: 
                  <strong>{testResult.matched_group || t('forwardGroups.noMatch', currentLanguage)}</strong>
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