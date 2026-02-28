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
import { useTranslation } from 'react-i18next'
import { apiClient } from '../utils/apiClient'
import BindConfigEditor from '../components/bind-config-editor/BindConfigEditor'

const { Option } = Select

const BindServerManager = () => {
  const { t } = useTranslation()
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
  
  // Backup management state
  const [backupModalVisible, setBackupModalVisible] = useState(false)
  const [backups, setBackups] = useState([])
  const [backupLoading, setBackupLoading] = useState(false)
  
  // Plugin status state
  const [pluginEnabled, setPluginEnabled] = useState(true)
  const [checkingPluginStatus, setCheckingPluginStatus] = useState(true)

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

  // Load BIND server status
  const loadBindServerStatus = useCallback(async () => {
    if (!pluginEnabled) return
    
    setLoading(true)
    try {
      // Load status
      const statusResponse = await apiClient.getBindServerStatus()
      if (statusResponse.success) {
        setBindStatus(statusResponse.data)
      } else {
        message.error(statusResponse.message || t('bindServer.fetchError'))
      }

      // Load stats
      const statsResponse = await apiClient.getBindServerStats()
      if (statsResponse.success) {
        setBindStats(statsResponse.data)
      } else {
        message.error(statsResponse.message || t('bindServer.fetchError'))
      }

      // Load config
      const configResponse = await apiClient.getBindConfig()
      if (configResponse.success) {
        setBindConfig(configResponse.data)
      } else {
        message.error(configResponse.message || t('bindServer.fetchError'))
      }
    } catch (error) {
      console.error('Error loading BIND server data:', error)
      // 检查是否是404错误（插件禁用）
      if (error.message.includes('404')) {
        setPluginEnabled(false)
        message.error(t('bindServer.pluginNotEnabled'))
      } else {
        message.error(t('bindServer.fetchError'))
      }
    } finally {
      setLoading(false)
    }
  }, [pluginEnabled, t])

  // 组件挂载时检查插件状态
  useEffect(() => {
    checkPluginStatus()
  }, [checkPluginStatus])

  // 插件状态变化时加载数据
  useEffect(() => {
    if (pluginEnabled) {
      loadBindServerStatus()
    }
  }, [pluginEnabled, loadBindServerStatus])

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
        case 'deleteBackup':
          await handleDeleteBackup(currentActionParams)
          break
        default:
          break
      }
    } catch (error) {
      console.error('Error executing action:', error)
      message.error(t('bindServer.controlError'))
    } finally {
      setActionLoading(false)
    }
  }

  // Control BIND server - actual implementation
  const handleControlBindServer = async (action) => {
    if (!pluginEnabled) {
      message.error(t('bindServer.pluginNotEnabled'))
      return
    }
    
    try {
      const response = await apiClient.controlBindServer(action)
      if (response.success) {
        message.success(response.message || t('bindServer.controlSuccess'))
        // Reload status after action
        setTimeout(loadBindServerStatus, 1000)
      } else {
        message.error(response.message || t('bindServer.controlError'))
      }
    } catch (error) {
      console.error(`Error ${action}ing BIND server:`, error)
      if (error.message.includes('404')) {
        setPluginEnabled(false)
        message.error(t('bindServer.pluginNotEnabled'))
      } else {
        message.error(t('bindServer.controlError'))
      }
    }
  }

  // Validate BIND configuration - actual implementation
  const handleValidateBindConfig = async () => {
    if (!pluginEnabled) {
      message.error(t('bindServer.pluginNotEnabled'))
      return
    }
    
    try {
      const response = await apiClient.validateBindConfig()
      if (response.success) {
        message.success(response.message || t('bindServer.validateSuccess'))
      } else {
        message.error(response.message || t('bindServer.validateError'))
      }
    } catch (error) {
      console.error('Error validating BIND config:', error)
      if (error.message.includes('404')) {
        setPluginEnabled(false)
        message.error(t('bindServer.pluginNotEnabled'))
      } else {
        message.error(t('bindServer.validateError'))
      }
    }
  }



  // Control BIND server - with confirmation modal
  const controlBindServer = (action) => {
    // Map action to human-readable name and warning message
    const actionMap = {
      start: {
        title: t('bindServer.start'),
        content: t('bindServer.startWarning')
      },
      stop: {
        title: t('bindServer.stop'),
        content: t('bindServer.stopWarning')
      },
      restart: {
        title: t('bindServer.restart'),
        content: t('bindServer.restartWarning')
      },
      reload: {
        title: t('bindServer.reload'),
        content: t('bindServer.reloadWarning')
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
    setConfirmModalTitle(t('bindServer.validateConfig'))
    setConfirmModalContent(t('bindServer.validateWarning'))
    setCurrentAction('validateBindConfig')
    setCurrentActionParams(null)
    setConfirmModalVisible(true)
  }

  // Load backups
  const loadBackups = async () => {
    if (!pluginEnabled) {
      message.error(t('bindServer.pluginNotEnabled'))
      setBackupLoading(false)
      return
    }
    
    setBackupLoading(true)
    try {
      const response = await apiClient.getBindNamedConfBackups()
      if (response.success) {
        setBackups(response.data || [])
      } else {
        message.error(response.message || t('bindServer.loadBackupListFailed'))
      }
    } catch (error) {
      console.error('Error loading backups:', error)
      if (error.message.includes('404')) {
        setPluginEnabled(false)
        message.error(t('bindServer.pluginNotEnabled'))
      } else {
        message.error(t('bindServer.loadBackupListFailed'))
      }
    } finally {
      setBackupLoading(false)
    }
  }

  // Handle restore backup
  const handleRestoreBackup = async (backupPath) => {
    if (!pluginEnabled) {
      message.error(t('bindServer.pluginNotEnabled'))
      setBackupLoading(false)
      return
    }
    
    try {
      setBackupLoading(true)
      const response = await apiClient.restoreBindNamedConfBackup(backupPath)
      if (response.success) {
        message.success(response.message || t('bindServer.restoreBackupSuccess'))
        // Reload BIND server status after restore
        setTimeout(loadBindServerStatus, 1000)
        setBackupModalVisible(false)
      } else {
        message.error(response.message || t('bindServer.restoreBackupFailed'))
      }
    } catch (error) {
      console.error('Error restoring backup:', error)
      if (error.message.includes('404')) {
        setPluginEnabled(false)
        message.error(t('bindServer.pluginNotEnabled'))
      } else {
        message.error(t('bindServer.restoreBackupFailed'))
      }
    } finally {
      setBackupLoading(false)
    }
  }

  // Handle delete backup
  const handleDeleteBackup = async (backupId) => {
    if (!pluginEnabled) {
      message.error(t('bindServer.pluginNotEnabled'))
      setBackupLoading(false)
      return
    }
    
    try {
      setBackupLoading(true)
      const response = await apiClient.deleteBindNamedConfBackup(backupId)
      if (response.success) {
        message.success(response.message || t('bindServer.deleteBackupSuccess'))
        // Reload backups after delete
        loadBackups()
      } else {
        message.error(response.message || t('bindServer.deleteBackupFailed'))
      }
    } catch (error) {
      console.error('Error deleting backup:', error)
      if (error.message.includes('404')) {
        setPluginEnabled(false)
        message.error(t('bindServer.pluginNotEnabled'))
      } else {
        message.error(t('bindServer.deleteBackupFailed'))
      }
    } finally {
      setBackupLoading(false)
    }
  }

  // Handle delete backup with confirmation
  const confirmDeleteBackup = (backupId) => {
    setConfirmModalTitle(t('bindServer.deleteBackup'))
    setConfirmModalContent(t('bindServer.confirmDeleteBackup'))
    setCurrentAction('deleteBackup')
    setCurrentActionParams(backupId)
    setConfirmModalVisible(true)
  }

  // 当插件未启用时显示的提示信息
  if (!pluginEnabled) {
    return (
      <div style={{ textAlign: 'center', padding: '60px 20px' }}>
        <div style={{ maxWidth: '600px', margin: '0 auto' }}>
          <Alert
            title={t('bindServer.pluginNotEnabled')}
            description={
              <div>
                <p style={{ marginBottom: '16px' }}>{t('bindServer.pluginNotEnabledDescription')}</p>
                <p style={{ marginBottom: '8px' }}><strong>{t('bindServer.enableMethod')}：</strong></p>
                <p>1. {t('bindServer.editConfigFile')}：<code>/src/cmd/config/steadydns.conf</code></p>
                <p>2. {t('bindServer.setBindEnabled')} <code>true</code></p>
                <p>3. {t('bindServer.restartService')}</p>
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

  // 当检查插件状态时显示加载状态
  if (checkingPluginStatus) {
    return (
      <div style={{ textAlign: 'center', padding: '60px' }}>
        <Spin size="large" tip={t('bindServer.checkingPluginStatus')}>
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
            {t('bindServer.title')}
          </Space>
        </h2>
        <Space>
          <Button
            icon={<SettingOutlined />}
            onClick={() => {
              setBackupModalVisible(true)
              loadBackups()
            }}
          >
            {t('bindServer.backupManagement')}
          </Button>
          <Button
            icon={<ReloadOutlined />}
            onClick={loadBindServerStatus}
            loading={loading}
          >
            {t('bindServer.refresh')}
          </Button>
        </Space>
      </div>

      <Spin spinning={loading}>
        {bindStatus ? (
          <>
            {/* BIND Server Health */}
            <Card title={t('bindServer.health')} style={{ marginBottom: 24 }}>
              <Row gutter={[16, 16]}>
                <Col xs={24} sm={8}>
                  <Alert
                    title={t('bindServer.configValid')}
                    description={t('bindServer.configValidDescription')}
                    type="success"
                    showIcon
                  />
                </Col>
                <Col xs={24} sm={8}>
                  <Alert
                    title={t('bindServer.portAvailable')}
                    description={t('bindServer.portAvailableDescription')}
                    type="success"
                    showIcon
                  />
                </Col>
                <Col xs={24} sm={8}>
                  <Alert
                    title={t('bindServer.overallHealth')}
                    description={t('bindServer.overallHealthDescription')}
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
                  title={t('bindServer.status')}
                  style={{ border: '1px solid #f0f0f0' }}
                  hoverable
                >
                  <Statistic
                    title={t('bindServer.status')}
                    value={bindStatus.status || 'unknown'}
                    valueProps={{
                      style: {
                        color: bindStatus.status === 'running' ? '#52c41a' : '#ff4d4f'
                      }
                    }}
                  />
                  <div style={{ marginTop: 16 }}>
                    <Tag color={bindStatus.status === 'running' ? 'green' : 'red'}>
                      {bindStatus.status === 'running' ? t('bindServer.running') : t('bindServer.stopped')}
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
                          {t('bindServer.start')}
                        </Button>
                      ) : (
                        <Button
                          danger
                          icon={<PauseCircleOutlined />}
                          onClick={() => controlBindServer('stop')}
                          loading={actionLoading}
                        >
                          {t('bindServer.stop')}
                        </Button>
                      )}
                      <Button
                        icon={<SyncOutlined />}
                        onClick={() => controlBindServer('restart')}
                        loading={actionLoading}
                      >
                        {t('bindServer.restart')}
                      </Button>
                      <Button
                        icon={<ReloadOutlined />}
                        onClick={() => controlBindServer('reload')}
                        loading={actionLoading}
                      >
                        {t('bindServer.reload')}
                      </Button>
                    </Space>
                  </div>
                </Card>
              </Col>
              <Col xs={24} sm={12} md={8}>
                <Card
                  title={t('bindServer.stats')}
                  style={{ border: '1px solid #f0f0f0' }}
                  hoverable
                >
                  <Statistic
                    title={t('bindServer.version')}
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
                  title={t('bindServer.actions')}
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
                    {t('bindServer.validateConfig')}
                  </Button>
                  <Alert
                    title={t('bindServer.warning')}
                    description={t('bindServer.actionWarning')}
                    type="warning"
                    showIcon
                    style={{ marginBottom: 16 }}
                  />
                </Card>
              </Col>
            </Row>

            {/* BIND Server Configuration */}
            <Card title={t('bindServer.configuration')} style={{ marginBottom: 24 }}>
              <Row gutter={[16, 16]}>
                <Col xs={24} sm={12}>
                  <div style={{ marginBottom: 16 }}>
                    <span style={{ fontWeight: 'bold', display: 'block', marginBottom: 4 }}>
                      {t('bindServer.bindAddress')}:
                    </span>
                    <span>{bindConfig?.BIND_ADDRESS || 'unknown'}</span>
                  </div>
                </Col>
                <Col xs={24} sm={12}>
                  <div style={{ marginBottom: 16 }}>
                    <span style={{ fontWeight: 'bold', display: 'block', marginBottom: 4 }}>
                      {t('bindServer.zoneFilePath')}:
                    </span>
                    <span>{bindConfig?.ZONE_FILE_PATH || 'unknown'}</span>
                  </div>
                </Col>
                <Col xs={24} sm={12}>
                  <div style={{ marginBottom: 16 }}>
                    <span style={{ fontWeight: 'bold', display: 'block', marginBottom: 4 }}>
                      {t('bindServer.namedConfPath')}:
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
                      {t('bindServer.editConfig')}
                    </Button>
                  </div>
                </Col>
              </Row>
            </Card>

            {/* BIND Server Detailed Statistics */}
            <Card title={t('bindServer.detailedStatistics')} style={{ marginBottom: 24 }}>
              <Row gutter={[16, 16]}>
                <Col xs={24} sm={12}>
                  <div style={{ marginBottom: 16 }}>
                    <h4 style={{ marginBottom: 8 }}>{t('bindServer.serverInformation')}</h4>
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
                      <div>
                        <strong>{t('bindServer.bootTime')}</strong> {bindStats?.["boot time"] || 'unknown'}
                      </div>
                      <div>
                        <strong>{t('bindServer.lastConfigured')}</strong> {bindStats?.["last configured"] || 'unknown'}
                      </div>
                      <div>
                        <strong>{t('bindServer.configurationFile')}</strong> {bindStats?.["configuration file"] || 'unknown'}
                      </div>
                      <div>
                        <strong>{t('bindServer.runningOn')}</strong> {bindStats?.["running on localhost"] || 'unknown'}
                      </div>
                    </div>
                  </div>
                </Col>
                <Col xs={24} sm={12}>
                  <div style={{ marginBottom: 16 }}>
                    <h4 style={{ marginBottom: 8 }}>{t('bindServer.performanceStatistics')}</h4>
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
                      <div>
                        <strong>{t('bindServer.debugLevel')}</strong> {bindStats?.["debug level"] || 'unknown'}
                      </div>
                      <div>
                        <strong>{t('bindServer.tcpHighWater')}</strong> {bindStats?.["TCP high-water"] || 'unknown'}
                      </div>
                      <div>
                        <strong>{t('bindServer.recursiveClients')}</strong> {bindStats?.["recursive clients"] || 'unknown'}
                      </div>
                      <div>
                        <strong>{t('bindServer.tcpClients')}</strong> {bindStats?.["tcp clients"] || 'unknown'}
                      </div>
                    </div>
                  </div>
                </Col>
                <Col xs={24}>
                  <div>
                    <h4 style={{ marginBottom: 8 }}>{t('bindServer.transferStatistics')}</h4>
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 8 }}>
                      <div>
                        <strong>{t('bindServer.xfersRunning')}</strong> {bindStats?.["xfers running"] || 'unknown'}
                      </div>
                      <div>
                        <strong>{t('bindServer.xfersDeferred')}</strong> {bindStats?.["xfers deferred"] || 'unknown'}
                      </div>
                      <div>
                        <strong>{t('bindServer.xfersFirstRefresh')}</strong> {bindStats?.["xfers first refresh"] || 'unknown'}
                      </div>
                      <div>
                        <strong>{t('bindServer.recursiveHighWater')}</strong> {bindStats?.["recursive high-water"] || 'unknown'}
                      </div>
                      <div>
                        <strong>{t('bindServer.soaQueriesInProgress')}</strong> {bindStats?.["soa queries in progress"] || 'unknown'}
                      </div>
                    </div>
                  </div>
                </Col>
              </Row>
            </Card>


          </>
        ) : (
          <Alert
            title={t('bindServer.loading')}
            description={t('bindServer.loadingDescription')}
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
        okText={t('bindServer.confirm')}
        cancelText={t('bindServer.cancel')}
        confirmLoading={actionLoading}
        okButtonProps={{ danger: true }}
      >
        <div>
          <Alert
            title={t('bindServer.warning')}
            description={confirmModalContent}
            type="warning"
            showIcon
            style={{ marginBottom: 16 }}
          />
          <p>{t('bindServer.confirmMessage')}</p>
        </div>
      </Modal>

      {/* Bind Config Editor */}
      <BindConfigEditor
        visible={configEditorVisible}
        onClose={() => setConfigEditorVisible(false)}
      />
      
      {/* Backup Management Modal */}
      <Modal
        title={t('bindServer.backupManagementTitle')}
        open={backupModalVisible}
        onOk={() => setBackupModalVisible(false)}
        onCancel={() => setBackupModalVisible(false)}
        okText={t('bindServer.close')}
        cancelText={t('bindServer.cancel')}
        width={800}
        styles={{ body: { maxHeight: 600, overflow: 'auto' } }}
      >
        <div>
          <h3 style={{ marginBottom: 16 }}>{t('bindServer.backupList')}</h3>
          <Spin spinning={backupLoading}>
            {backups.length > 0 ? (
              <div className="backup-list">
                {backups.map((backup, index) => {
                  // Extract filename from filePath
                  const filename = backup.filePath.split('/').pop()
                  return (
                    <div key={index} style={{ 
                      display: 'flex', 
                      justifyContent: 'space-between', 
                      alignItems: 'center',
                      padding: '12px',
                      border: '1px solid #f0f0f0',
                      borderRadius: '4px',
                      marginBottom: '8px',
                      backgroundColor: '#f9f9f9'
                    }}>
                      <div>
                        <div style={{ fontWeight: 'bold', marginBottom: '4px' }}>{filename}</div>
                        <div style={{ fontSize: '12px', color: '#666' }}>
                          <span>{t('bindServer.time')}: {new Date(backup.timestamp).toLocaleString()}</span>
                          <span style={{ marginLeft: '16px' }}>{t('bindServer.size')}: {backup.size} bytes</span>
                        </div>
                      </div>
                      <Space>
                        <Button 
                          type="primary" 
                          size="small"
                          onClick={() => handleRestoreBackup(backup.filePath)}
                        >
                          {t('bindServer.restore')}
                        </Button>
                        <Button 
                          danger 
                          size="small"
                          onClick={() => confirmDeleteBackup(filename)}
                        >
                          {t('bindServer.delete')}
                        </Button>
                      </Space>
                    </div>
                  )
                })}
              </div>
            ) : (
              <Alert
                title={t('bindServer.noBackups')}
                description={t('bindServer.noBackupsDescription')}
                type="info"
                showIcon
              />
            )}
          </Spin>
        </div>
      </Modal>
    </div>
  )
}

export default BindServerManager