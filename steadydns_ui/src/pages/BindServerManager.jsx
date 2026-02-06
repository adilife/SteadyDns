import { useState, useEffect, useCallback } from 'react'
import {
  Card,
  Row,
  Col,
  Button,
  Space,
  message,
  Spin,
  Statistic,
  Tag,
  Alert,
  Form,
  Input,
  Select,
  Modal
} from 'antd'
import {
  DatabaseOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  SyncOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  SettingOutlined,
  InfoCircleOutlined
} from '@ant-design/icons'
import { t } from '../i18n'
import { apiClient } from '../utils/apiClient'
import BindConfigEditor from '../components/bind-config-editor/BindConfigEditor'

const { Option } = Select

const BindServerManager = ({ currentLanguage }) => {
  const [bindStatus, setBindStatus] = useState(null)
  const [bindStats, setBindStats] = useState(null)
  const [bindConfig, setBindConfig] = useState(null)
  const [loading, setLoading] = useState(false)
  const [actionLoading, setActionLoading] = useState(false)
  
  // Confirm modal state
  const [confirmModalVisible, setConfirmModalVisible] = useState(false)
  const [confirmModalTitle, setConfirmModalTitle] = useState('')
  const [confirmModalContent, setConfirmModalContent] = useState('')
  const [currentAction, setCurrentAction] = useState('')
  const [currentActionParams, setCurrentActionParams] = useState(null)
  
  // Bind config editor state
  const [configEditorVisible, setConfigEditorVisible] = useState(false)

  // Load BIND server status
  const loadBindServerStatus = useCallback(async () => {
    setLoading(true)
    try {
      // Load status
      const statusResponse = await apiClient.getBindServerStatus()
      if (statusResponse.success) {
        setBindStatus(statusResponse.data)
      } else {
        message.error(statusResponse.message || t('bindServer.fetchError', currentLanguage))
      }

      // Load stats
      const statsResponse = await apiClient.getBindServerStats()
      if (statsResponse.success) {
        setBindStats(statsResponse.data)
      } else {
        message.error(statsResponse.message || t('bindServer.fetchError', currentLanguage))
      }

      // Load config
      const configResponse = await apiClient.getBindConfig()
      if (configResponse.success) {
        setBindConfig(configResponse.data)
      } else {
        message.error(configResponse.message || t('bindServer.fetchError', currentLanguage))
      }
    } catch (error) {
      console.error('Error loading BIND server data:', error)
      message.error(t('bindServer.fetchError', currentLanguage))
    } finally {
      setLoading(false)
    }
  }, [currentLanguage])

  // Load BIND server status on component mount
  useEffect(() => {
    loadBindServerStatus()
  }, [loadBindServerStatus])

  // Handle confirm action
  const handleConfirmAction = async () => {
    setConfirmModalVisible(false)
    setActionLoading(true)
    
    try {
      switch (currentAction) {
        case 'controlBindServer':
          await handleControlBindServer(currentActionParams)
          break
        case 'validateBindConfig':
          await handleValidateBindConfig()
          break
        default:
          break
      }
    } catch (error) {
      console.error('Error executing action:', error)
      message.error(t('bindServer.controlError', currentLanguage))
    } finally {
      setActionLoading(false)
    }
  }

  // Control BIND server - actual implementation
  const handleControlBindServer = async (action) => {
    try {
      const response = await apiClient.controlBindServer(action)
      if (response.success) {
        message.success(response.message || t('bindServer.controlSuccess', currentLanguage))
        // Reload status after action
        setTimeout(loadBindServerStatus, 1000)
      } else {
        message.error(response.message || t('bindServer.controlError', currentLanguage))
      }
    } catch (error) {
      console.error(`Error ${action}ing BIND server:`, error)
      message.error(t('bindServer.controlError', currentLanguage))
    }
  }

  // Validate BIND configuration - actual implementation
  const handleValidateBindConfig = async () => {
    try {
      const response = await apiClient.validateBindConfig()
      if (response.success) {
        message.success(response.message || t('bindServer.validateSuccess', currentLanguage))
      } else {
        message.error(response.message || t('bindServer.validateError', currentLanguage))
      }
    } catch (error) {
      console.error('Error validating BIND config:', error)
      message.error(t('bindServer.validateError', currentLanguage))
    }
  }



  // Control BIND server - with confirmation modal
  const controlBindServer = (action) => {
    // Map action to human-readable name and warning message
    const actionMap = {
      start: {
        title: t('bindServer.start', currentLanguage),
        content: t('bindServer.startWarning', currentLanguage)
      },
      stop: {
        title: t('bindServer.stop', currentLanguage),
        content: t('bindServer.stopWarning', currentLanguage)
      },
      restart: {
        title: t('bindServer.restart', currentLanguage),
        content: t('bindServer.restartWarning', currentLanguage)
      },
      reload: {
        title: t('bindServer.reload', currentLanguage),
        content: t('bindServer.reloadWarning', currentLanguage)
      }
    }

    const actionInfo = actionMap[action]
    if (actionInfo) {
      setConfirmModalTitle(`${actionInfo.title} BIND Server`)
      setConfirmModalContent(actionInfo.content)
      setCurrentAction('controlBindServer')
      setCurrentActionParams(action)
      setConfirmModalVisible(true)
    }
  }

  // Validate BIND configuration - with confirmation modal
  const validateBindConfig = () => {
    setConfirmModalTitle(t('bindServer.validateConfig', currentLanguage))
    setConfirmModalContent(t('bindServer.validateWarning', currentLanguage))
    setCurrentAction('validateBindConfig')
    setCurrentActionParams(null)
    setConfirmModalVisible(true)
  }

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2>
          <Space>
            <DatabaseOutlined />
            {t('bindServer.title', currentLanguage)}
          </Space>
        </h2>
        <Button
          icon={<ReloadOutlined />}
          onClick={loadBindServerStatus}
          loading={loading}
        >
          {t('bindServer.refresh', currentLanguage)}
        </Button>
      </div>

      <Spin spinning={loading}>
        {bindStatus ? (
          <>
            {/* BIND Server Health */}
            <Card title={t('bindServer.health', currentLanguage)} style={{ marginBottom: 24 }}>
              <Row gutter={[16, 16]}>
                <Col xs={24} sm={8}>
                  <Alert
                    title={t('bindServer.configValid', currentLanguage)}
                    description={t('bindServer.configValidDescription', currentLanguage)}
                    type="success"
                    showIcon
                  />
                </Col>
                <Col xs={24} sm={8}>
                  <Alert
                    title={t('bindServer.portAvailable', currentLanguage)}
                    description={t('bindServer.portAvailableDescription', currentLanguage)}
                    type="success"
                    showIcon
                  />
                </Col>
                <Col xs={24} sm={8}>
                  <Alert
                    title={t('bindServer.overallHealth', currentLanguage)}
                    description={t('bindServer.overallHealthDescription', currentLanguage)}
                    type="success"
                    showIcon
                  />
                </Col>
              </Row>
            </Card>

            {/* BIND Server Status Card */}
            <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
              <Col xs={24} sm={12} md={8}>
                <Card
                  title={t('bindServer.status', currentLanguage)}
                  style={{ border: '1px solid #f0f0f0' }}
                  hoverable
                >
                  <Statistic
                    title={t('bindServer.status', currentLanguage)}
                    value={bindStatus.status || 'unknown'}
                    valueProps={{
                      style: {
                        color: bindStatus.status === 'running' ? '#52c41a' : '#ff4d4f'
                      }
                    }}
                  />
                  <div style={{ marginTop: 16 }}>
                    <Tag color={bindStatus.status === 'running' ? 'green' : 'red'}>
                      {bindStatus.status === 'running' ? t('bindServer.running', currentLanguage) : t('bindServer.stopped', currentLanguage)}
                    </Tag>
                  </div>
                  <div style={{ marginTop: 16 }}>
                    <Space>
                      {bindStatus.status !== 'running' ? (
                        <Button
                          type="primary"
                          icon={<PlayCircleOutlined />}
                          onClick={() => controlBindServer('start')}
                          loading={actionLoading}
                        >
                          {t('bindServer.start', currentLanguage)}
                        </Button>
                      ) : (
                        <Button
                          danger
                          icon={<PauseCircleOutlined />}
                          onClick={() => controlBindServer('stop')}
                          loading={actionLoading}
                        >
                          {t('bindServer.stop', currentLanguage)}
                        </Button>
                      )}
                      <Button
                        icon={<SyncOutlined />}
                        onClick={() => controlBindServer('restart')}
                        loading={actionLoading}
                      >
                        {t('bindServer.restart', currentLanguage)}
                      </Button>
                      <Button
                        icon={<ReloadOutlined />}
                        onClick={() => controlBindServer('reload')}
                        loading={actionLoading}
                      >
                        {t('bindServer.reload', currentLanguage)}
                      </Button>
                    </Space>
                  </div>
                </Card>
              </Col>
              <Col xs={24} sm={12} md={8}>
                <Card
                  title={t('bindServer.stats', currentLanguage)}
                  style={{ border: '1px solid #f0f0f0' }}
                  hoverable
                >
                  <Statistic
                    title={t('bindServer.version', currentLanguage)}
                    value={bindStats?.version || 'unknown'}
                  />
                  <Statistic
                    title="CPUs"
                    value={bindStats?.["CPUs found"] || 'unknown'}
                  />
                  <Statistic
                    title="Zones"
                    value={bindStats?.["number of zones"] || 'unknown'}
                  />
                  <Statistic
                    title="Worker Threads"
                    value={bindStats?.["worker threads"] || 'unknown'}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} md={8}>
                <Card
                  title={t('bindServer.actions', currentLanguage)}
                  style={{ border: '1px solid #f0f0f0' }}
                  hoverable
                >
                  <Button
                    type="primary"
                    icon={<CheckCircleOutlined />}
                    onClick={validateBindConfig}
                    loading={actionLoading}
                    block
                    style={{ marginBottom: 16 }}
                  >
                    {t('bindServer.validateConfig', currentLanguage)}
                  </Button>
                  <Alert
                    title={t('bindServer.warning', currentLanguage)}
                    description={t('bindServer.actionWarning', currentLanguage)}
                    type="warning"
                    showIcon
                    style={{ marginBottom: 16 }}
                  />
                </Card>
              </Col>
            </Row>

            {/* BIND Server Configuration */}
            <Card title={t('bindServer.configuration', currentLanguage)} style={{ marginBottom: 24 }}>
              <Row gutter={[16, 16]}>
                <Col xs={24} sm={12}>
                  <div style={{ marginBottom: 16 }}>
                    <span style={{ fontWeight: 'bold', display: 'block', marginBottom: 4 }}>
                      {t('bindServer.bindAddress', currentLanguage)}:
                    </span>
                    <span>{bindConfig?.BIND_ADDRESS || 'unknown'}</span>
                  </div>
                </Col>
                <Col xs={24} sm={12}>
                  <div style={{ marginBottom: 16 }}>
                    <span style={{ fontWeight: 'bold', display: 'block', marginBottom: 4 }}>
                      {t('bindServer.zoneFilePath', currentLanguage)}:
                    </span>
                    <span>{bindConfig?.ZONE_FILE_PATH || 'unknown'}</span>
                  </div>
                </Col>
                <Col xs={24} sm={12}>
                  <div style={{ marginBottom: 16 }}>
                    <span style={{ fontWeight: 'bold', display: 'block', marginBottom: 4 }}>
                      {t('bindServer.namedConfPath', currentLanguage)}:
                    </span>
                    <span>{bindConfig?.NAMED_CONF_PATH || 'unknown'}</span>
                  </div>
                </Col>
                <Col xs={24} sm={12}>
                  <div style={{ marginBottom: 16, display: 'flex', alignItems: 'flex-end' }}>
                    <Button
                      type="primary"
                      icon={<SettingOutlined />}
                      style={{ marginBottom: 16 }}
                      onClick={() => setConfigEditorVisible(true)}
                    >
                      {t('bindServer.editConfig', currentLanguage)}
                    </Button>
                  </div>
                </Col>
              </Row>
            </Card>

            {/* BIND Server Detailed Statistics */}
            <Card title="Detailed Statistics" style={{ marginBottom: 24 }}>
              <Row gutter={[16, 16]}>
                <Col xs={24} sm={12}>
                  <div style={{ marginBottom: 16 }}>
                    <h4 style={{ marginBottom: 8 }}>Server Information</h4>
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
                      <div>
                        <strong>Boot Time:</strong> {bindStats?.["boot time"] || 'unknown'}
                      </div>
                      <div>
                        <strong>Last Configured:</strong> {bindStats?.["last configured"] || 'unknown'}
                      </div>
                      <div>
                        <strong>Configuration File:</strong> {bindStats?.["configuration file"] || 'unknown'}
                      </div>
                      <div>
                        <strong>Running On:</strong> {bindStats?.["running on localhost"] || 'unknown'}
                      </div>
                    </div>
                  </div>
                </Col>
                <Col xs={24} sm={12}>
                  <div style={{ marginBottom: 16 }}>
                    <h4 style={{ marginBottom: 8 }}>Performance Statistics</h4>
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
                      <div>
                        <strong>Debug Level:</strong> {bindStats?.["debug level"] || 'unknown'}
                      </div>
                      <div>
                        <strong>TCP High-water:</strong> {bindStats?.["TCP high-water"] || 'unknown'}
                      </div>
                      <div>
                        <strong>Recursive Clients:</strong> {bindStats?.["recursive clients"] || 'unknown'}
                      </div>
                      <div>
                        <strong>TCP Clients:</strong> {bindStats?.["tcp clients"] || 'unknown'}
                      </div>
                    </div>
                  </div>
                </Col>
                <Col xs={24}>
                  <div>
                    <h4 style={{ marginBottom: 8 }}>Transfer Statistics</h4>
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 8 }}>
                      <div>
                        <strong>Xfers Running:</strong> {bindStats?.["xfers running"] || 'unknown'}
                      </div>
                      <div>
                        <strong>Xfers Deferred:</strong> {bindStats?.["xfers deferred"] || 'unknown'}
                      </div>
                      <div>
                        <strong>Xfers First Refresh:</strong> {bindStats?.["xfers first refresh"] || 'unknown'}
                      </div>
                      <div>
                        <strong>Recursive High-water:</strong> {bindStats?.["recursive high-water"] || 'unknown'}
                      </div>
                      <div>
                        <strong>SOA Queries In Progress:</strong> {bindStats?.["soa queries in progress"] || 'unknown'}
                      </div>
                    </div>
                  </div>
                </Col>
              </Row>
            </Card>


          </>
        ) : (
          <Alert
            title={t('bindServer.loading', currentLanguage)}
            description={t('bindServer.loadingDescription', currentLanguage)}
            type="info"
            showIcon
          />
        )}
      </Spin>

      {/* Confirm Modal */}
      <Modal
        title={confirmModalTitle}
        open={confirmModalVisible}
        onOk={handleConfirmAction}
        onCancel={() => setConfirmModalVisible(false)}
        okText={t('bindServer.confirm', currentLanguage)}
        cancelText={t('bindServer.cancel', currentLanguage)}
        confirmLoading={actionLoading}
        okButtonProps={{ danger: true }}
      >
        <div>
          <Alert
            title={t('bindServer.warning', currentLanguage)}
            description={confirmModalContent}
            type="warning"
            showIcon
            style={{ marginBottom: 16 }}
          />
          <p>{t('bindServer.confirmMessage', currentLanguage)}</p>
        </div>
      </Modal>

      {/* Bind Config Editor */}
      <BindConfigEditor
        visible={configEditorVisible}
        onClose={() => setConfigEditorVisible(false)}
      />
    </div>
  )
}

export default BindServerManager