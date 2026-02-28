import { useState, useEffect, useCallback } from 'react'
import {
  Tabs,
  Card,
  Spin
} from 'antd'
import {
  SettingOutlined
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import ServerManager from './ServerManager'
import CacheManager from './CacheManager'
import BindServerManager from './BindServerManager'
import Settings from './Settings'
import { apiClient } from '../utils/apiClient'

const Configuration = ({ userInfo }) => {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState('server')
  const [settingsKey, setSettingsKey] = useState(0)
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

  // 组件挂载时检查插件状态
  useEffect(() => {
    checkPluginStatus()
  }, [checkPluginStatus])

  const handleTabChange = (key) => {
    setActiveTab(key)
    // 当切换到settings标签时，强制重新渲染Settings组件
    if (key === 'settings') {
      setSettingsKey(prev => prev + 1)
    }
  }

  // 动态生成标签页
  const items = [
    {
      key: 'server',
      label: t('configuration.server'),
      children: <ServerManager userInfo={userInfo} />
    },
    {
      key: 'cache',
      label: t('configuration.cache'),
      children: <CacheManager userInfo={userInfo} />
    },
    // 只有当BIND插件启用时才显示BIND Server标签页
    ...(pluginEnabled ? [{
      key: 'bindServer',
      label: t('configuration.bindServer'),
      children: <BindServerManager userInfo={userInfo} />
    }] : []),
    {
      key: 'settings',
      label: t('configuration.settings'),
      children: <Settings key={settingsKey} userInfo={userInfo} />
    }
  ]

  // 当检查插件状态时显示加载状态
  if (checkingPluginStatus) {
    return (
      <div style={{ textAlign: 'center', padding: '60px' }}>
        <Spin size="large" tip={t('configuration.checkingPluginStatus')}>
          <div style={{ padding: 50 }} />
        </Spin>
      </div>
    )
  }

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', alignItems: 'center' }}>
        <h2>
          <SettingOutlined style={{ marginRight: 8 }} />
          {t('configuration.title')}
        </h2>
      </div>

      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={handleTabChange}
          size="large"
          style={{ marginBottom: 24 }}
          items={items}
        />
      </Card>
    </div>
  )
}

export default Configuration