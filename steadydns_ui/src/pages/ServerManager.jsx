import { useState, useEffect, useCallback } from 'react'
import {
  Card,
  Row,
  Col,
  Button,
  Space,
  message,
  Select,
  Spin,
  Statistic,
  Tag,
  Alert,
  Modal
} from 'antd'
import {
  AppstoreOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  SyncOutlined,
  ReloadOutlined,
  SettingOutlined,
  InfoCircleOutlined
} from '@ant-design/icons'
import { t } from '../i18n'
import { apiClient } from '../utils/apiClient'

const { Option } = Select

const ServerManager = ({ currentLanguage }) => {
  const [serverStatus, setServerStatus] = useState(null)
  const [loading, setLoading] = useState(false)
  const [apiLogLevel, setApiLogLevel] = useState('INFO')
  const [dnsLogLevel, setDnsLogLevel] = useState('DEBUG')
  const [actionLoading, setActionLoading] = useState(false)
  // Confirmation modal states
  const [confirmModalVisible, setConfirmModalVisible] = useState(false)
  const [confirmModalTitle, setConfirmModalTitle] = useState('')
  const [confirmModalContent, setConfirmModalContent] = useState('')
  const [pendingAction, setPendingAction] = useState(null)

  // Load server status from API
  const loadServerStatus = useCallback(async () => {
    setLoading(true)
    try {
      const response = await apiClient.getServerStatus()
      if (response.success) {
        setServerStatus(response.data)
        // Update log levels from new logging field
        if (response.data.logging) {
          setApiLogLevel(response.data.logging.api_log_level || 'INFO')
          setDnsLogLevel(response.data.logging.dns_log_level || 'DEBUG')
        }
      } else {
        message.error(response.message || t('servermanager.fetchError', currentLanguage))
      }
    } catch (error) {
      console.error('Error loading server status:', error)
      message.error(t('servermanager.fetchError', currentLanguage))
    } finally {
      setLoading(false)
    }
  }, [currentLanguage])

  // Load server status on component mount
  useEffect(() => {
    loadServerStatus()
  }, [loadServerStatus])

  // Auto refresh server status every 30 seconds
  useEffect(() => {
    const refreshInterval = setInterval(loadServerStatus, 30000)
    return () => clearInterval(refreshInterval)
  }, [loadServerStatus])

  // Show confirmation modal
  const showConfirmModal = (action, title, content, params) => {
    setConfirmModalTitle(title)
    setConfirmModalContent(content)
    setPendingAction({ action, params })
    setConfirmModalVisible(true)
  }

  // Handle confirm action
  const handleConfirmAction = async () => {
    if (!pendingAction) return

    const { action, params } = pendingAction
    setConfirmModalVisible(false)

    try {
      switch (action) {
        case 'controlServer':
          if (params.serverType === 'httpd' && params.action === 'restart') {
            // For HTTP server restart, keep actionLoading true during delay
            setActionLoading(true)
            try {
              await controlServer(params.action, params.serverType)
              // Add 6 seconds delay for HTTP server restart
              await new Promise(resolve => setTimeout(resolve, 6000))
            } finally {
              setActionLoading(false)
            }
          } else {
            // For other actions, use normal flow
            await controlServer(params.action, params.serverType)
          }
          break
        case 'setLogLevels':
          await handleLogLevelsChange(params.apiLevel, params.dnsLevel)
          break
        default:
          break
      }
    } catch (error) {
      console.error('Error executing action:', error)
      setActionLoading(false)
    } finally {
      setPendingAction(null)
    }
  }

  // Handle cancel action
  const handleCancelAction = () => {
    setConfirmModalVisible(false)
    setPendingAction(null)
  }

  // Control server (start/stop/restart)
  const controlServer = async (action, serverType = 'sdnsd') => {
    // For non-HTTP restart actions, set actionLoading here
    if (!(serverType === 'httpd' && action === 'restart')) {
      setActionLoading(true)
    }
    
    try {
      const response = await apiClient.controlServer(action, serverType)
      if (response.success) {
        message.success(response.message || t('servermanager.controlSuccess', currentLanguage))
        // Reload server status after action
        // Add longer delay for HTTP server restart to ensure it fully restarts
        const reloadDelay = (serverType === 'httpd' && action === 'restart') ? 6000 : 1000
        setTimeout(loadServerStatus, reloadDelay)
      } else {
        message.error(response.message || t('servermanager.controlError', currentLanguage))
      }
    } catch (error) {
      console.error(`Error ${action}ing ${serverType}:`, error)
      message.error(t('servermanager.controlError', currentLanguage))
    } finally {
      // For non-HTTP restart actions, reset actionLoading here
      if (!(serverType === 'httpd' && action === 'restart')) {
        setActionLoading(false)
      }
    }
  }

  // Reload forward groups
  const reloadForwardGroups = async () => {
    setActionLoading(true)
    try {
      const response = await apiClient.reloadForwardGroups()
      if (response.success) {
        message.success(response.message || t('servermanager.reloadSuccess', currentLanguage))
      } else {
        message.error(response.message || t('servermanager.reloadError', currentLanguage))
      }
    } catch (error) {
      console.error('Error reloading forward groups:', error)
      message.error(t('servermanager.reloadError', currentLanguage))
    } finally {
      setActionLoading(false)
    }
  }

  // Set log levels for API and DNS
  const handleLogLevelsChange = async (apiLevel, dnsLevel) => {
    setActionLoading(true)
    try {
      const response = await apiClient.setLogLevels({
        api_log_level: apiLevel,
        dns_log_level: dnsLevel
      })
      if (response.success) {
        // Update local state
        setApiLogLevel(apiLevel)
        setDnsLogLevel(dnsLevel)
        message.success(response.message || 'Log levels set successfully')
      } else {
        message.error(response.error || 'Failed to set log levels')
      }
    } catch (error) {
      console.error('Error setting log levels:', error)
      message.error('Failed to set log levels')
    } finally {
      setActionLoading(false)
    }
  }

  // Get DNS server status based on tcp_server and udp_server
  const getDnsServerStatus = () => {
    const tcpServer = serverStatus?.dns_server?.tcp_server
    const udpServer = serverStatus?.dns_server?.udp_server
    
    if (tcpServer && udpServer) {
      return 'running'
    } else if (!tcpServer && !udpServer) {
      return 'stopped'
    } else {
      return 'partial'
    }
  }

  // Get status text based on status code
  const getStatusText = (status) => {
    switch (status) {
      case 'running':
        return t('servermanager.running', currentLanguage)
      case 'stopped':
        return t('servermanager.stopped', currentLanguage)
      case 'partial':
        return t('servermanager.partialRunning', currentLanguage)
      default:
        return t('servermanager.unknown', currentLanguage)
    }
  }

  // Get status color based on status code
  const getStatusColor = (status) => {
    switch (status) {
      case 'running':
        return '#52c41a'
      case 'stopped':
        return '#ff4d4f'
      case 'partial':
        return '#faad14'
      default:
        return '#666'
    }
  }

  return (
    <div>
      {/* Global loading spinner for actionLoading */}
      <Spin spinning={actionLoading} fullscreen />
      
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2>
          <Space>
            <AppstoreOutlined />
            {t('servermanager.title', currentLanguage)}
          </Space>
        </h2>
        <Button
          icon={<ReloadOutlined />}
          onClick={loadServerStatus}
          loading={loading}
        >
          {t('servermanager.refresh', currentLanguage)}
        </Button>
      </div>

      <Spin spinning={loading}>
        {serverStatus ? (
          <>
            {/* Server Status Cards */}
            <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
              <Col xs={24} sm={12} md={8}>
                <Card
                  title={t('servermanager.dnsServer', currentLanguage)}
                  style={{ border: '1px solid #f0f0f0' }}
                  hoverable
                >
                  <Statistic
                    title={t('servermanager.status', currentLanguage)}
                    value={getStatusText(getDnsServerStatus())}
                    valueProps={{
                      style: {
                        color: getStatusColor(getDnsServerStatus())
                      }
                    }}
                  />
                  <div style={{ marginTop: 16 }}>
                    <Space>
                      {getDnsServerStatus() !== 'running' ? (
                        <Button
                          type="primary"
                          icon={<PlayCircleOutlined />}
                          onClick={() => showConfirmModal(
                            'controlServer',
                            t('servermanager.confirmTitle', currentLanguage),
                            t('servermanager.confirmStartDns', currentLanguage),
                            { action: 'start', serverType: 'sdnsd' }
                          )}
                          loading={actionLoading}
                        >
                          {t('servermanager.start', currentLanguage)}
                        </Button>
                      ) : (
                        <Button
                          danger
                          icon={<PauseCircleOutlined />}
                          onClick={() => showConfirmModal(
                            'controlServer',
                            t('servermanager.confirmTitle', currentLanguage),
                            t('servermanager.confirmStopDns', currentLanguage),
                            { action: 'stop', serverType: 'sdnsd' }
                          )}
                          loading={actionLoading}
                        >
                          {t('servermanager.stop', currentLanguage)}
                        </Button>
                      )}
                      <Button
                        icon={<SyncOutlined />}
                        onClick={() => showConfirmModal(
                          'controlServer',
                          t('servermanager.confirmTitle', currentLanguage),
                          t('servermanager.confirmRestartDns', currentLanguage),
                          { action: 'restart', serverType: 'sdnsd' }
                        )}
                        loading={actionLoading}
                      >
                        {t('servermanager.restart', currentLanguage)}
                      </Button>
                    </Space>
                  </div>
                </Card>
              </Col>
              <Col xs={24} sm={12} md={8}>
                <Card
                  title={t('servermanager.httpServer', currentLanguage)}
                  style={{ border: '1px solid #f0f0f0' }}
                  hoverable
                >
                  <Statistic
                    title={t('servermanager.status', currentLanguage)}
                    value={serverStatus.http_server?.running ? t('servermanager.running', currentLanguage) : t('servermanager.stopped', currentLanguage)}
                    valueProps={{
                      style: {
                        color: serverStatus.http_server?.running ? '#52c41a' : '#ff4d4f'
                      }
                    }}
                  />
                  <div style={{ marginTop: 16 }}>
                    <Space>
                      <Button
                        icon={<SyncOutlined />}
                        onClick={() => showConfirmModal(
                          'controlServer',
                          t('servermanager.confirmTitle', currentLanguage),
                          t('servermanager.confirmRestartHttp', currentLanguage),
                          { action: 'restart', serverType: 'httpd' }
                        )}
                        loading={actionLoading}
                      >
                        {t('servermanager.restart', currentLanguage)}
                      </Button>
                    </Space>
                  </div>
                </Card>
              </Col>
              <Col xs={24} sm={12} md={8}>
                <Card
                  title={t('servermanager.systemInfo', currentLanguage)}
                  style={{ border: '1px solid #f0f0f0' }}
                  hoverable
                >
                  <Statistic
                    title="Cache Initialized"
                    value={serverStatus.cache?.initialized ? t('servermanager.running', currentLanguage) : t('servermanager.stopped', currentLanguage)}
                    valueProps={{
                      style: {
                        color: serverStatus.cache?.initialized ? '#52c41a' : '#ff4d4f'
                      }
                    }}
                  />
                  <Statistic
                    title="Forwarder Initialized"
                    value={serverStatus.forwarder?.initialized ? t('servermanager.running', currentLanguage) : t('servermanager.stopped', currentLanguage)}
                    valueProps={{
                      style: {
                        color: serverStatus.forwarder?.initialized ? '#52c41a' : '#ff4d4f'
                      }
                    }}
                  />
                  <div style={{ marginTop: 16 }}>
                    <Button
                      icon={<ReloadOutlined />}
                      onClick={reloadForwardGroups}
                      loading={actionLoading}
                    >
                      {t('servermanager.reloadForwardGroups', currentLanguage)}
                    </Button>
                  </div>
                </Card>
              </Col>
            </Row>

            {/* Server Configuration */}
            <Card title={t('servermanager.configuration', currentLanguage)} style={{ marginBottom: 24 }}>
              <Row gutter={[16, 16]}>
                <Col xs={24} sm={12}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
                    <span style={{ fontWeight: 'bold', minWidth: 120 }}>
                      DNS {t('servermanager.logLevel', currentLanguage)}:
                    </span>
                    <Select
                      value={dnsLogLevel}
                      onChange={(level) => showConfirmModal(
                        'setLogLevels',
                        t('servermanager.confirmTitle', currentLanguage),
                        t('servermanager.confirmDnsLogLevel', currentLanguage).replace('{{level}}', level),
                        { apiLevel: apiLogLevel, dnsLevel: level }
                      )}
                      style={{ width: 150 }}
                      disabled={actionLoading}
                    >
                      <Select.Option value="debug">{t('servermanager.logDebug', currentLanguage)}</Select.Option>
                      <Select.Option value="info">{t('servermanager.logInfo', currentLanguage)}</Select.Option>
                      <Select.Option value="warn">{t('servermanager.logWarn', currentLanguage)}</Select.Option>
                      <Select.Option value="error">{t('servermanager.logError', currentLanguage)}</Select.Option>
                    </Select>
                  </div>
                </Col>
                <Col xs={24} sm={12}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 16 }}>
                    <span style={{ fontWeight: 'bold', minWidth: 120 }}>
                      API {t('servermanager.logLevel', currentLanguage)}:
                    </span>
                    <Select
                      value={apiLogLevel}
                      onChange={(level) => showConfirmModal(
                        'setLogLevels',
                        t('servermanager.confirmTitle', currentLanguage),
                        t('servermanager.confirmApiLogLevel', currentLanguage).replace('{{level}}', level),
                        { apiLevel: level, dnsLevel: dnsLogLevel }
                      )}
                      style={{ width: 150 }}
                      disabled={actionLoading}
                    >
                      <Select.Option value="debug">{t('servermanager.logDebug', currentLanguage)}</Select.Option>
                      <Select.Option value="info">{t('servermanager.logInfo', currentLanguage)}</Select.Option>
                      <Select.Option value="warn">{t('servermanager.logWarn', currentLanguage)}</Select.Option>
                      <Select.Option value="error">{t('servermanager.logError', currentLanguage)}</Select.Option>
                    </Select>
                  </div>
                </Col>
                
              </Row>
            </Card>
          </>
        ) : (
          <Alert
            title={t('servermanager.loading', currentLanguage)}
            description={t('servermanager.loadingDescription', currentLanguage)}
            type="info"
            showIcon
          />
        )}
      </Spin>

      {/* Confirmation Modal */}
      <Modal
        title={confirmModalTitle}
        open={confirmModalVisible}
        onOk={handleConfirmAction}
        onCancel={handleCancelAction}
        okText={t('servermanager.confirmOk', currentLanguage)}
        cancelText={t('servermanager.confirmCancel', currentLanguage)}
        okButtonProps={{ danger: true }}
        confirmLoading={actionLoading}
      >
        <p>{confirmModalContent}</p>
      </Modal>
    </div>
  )
}

export default ServerManager
