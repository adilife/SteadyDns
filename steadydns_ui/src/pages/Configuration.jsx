import { useState } from 'react'
import {
  Tabs,
  Card
} from 'antd'
import {
  SettingOutlined
} from '@ant-design/icons'
import { t } from '../i18n'
import ServerManager from './ServerManager'
import CacheManager from './CacheManager'
import BindServerManager from './BindServerManager'
import Settings from './Settings'

const Configuration = ({ currentLanguage, userInfo }) => {
  const [activeTab, setActiveTab] = useState('server')
  const [settingsKey, setSettingsKey] = useState(0)

  const handleTabChange = (key) => {
    setActiveTab(key)
    // 当切换到settings标签时，强制重新渲染Settings组件
    if (key === 'settings') {
      setSettingsKey(prev => prev + 1)
    }
  }

  const items = [
    {
      key: 'server',
      label: t('configuration.server', currentLanguage),
      children: <ServerManager currentLanguage={currentLanguage} userInfo={userInfo} />
    },
    {
      key: 'cache',
      label: t('configuration.cache', currentLanguage),
      children: <CacheManager currentLanguage={currentLanguage} userInfo={userInfo} />
    },
    {
      key: 'bindServer',
      label: t('configuration.bindServer', currentLanguage),
      children: <BindServerManager currentLanguage={currentLanguage} userInfo={userInfo} />
    },
    {
      key: 'settings',
      label: t('configuration.settings', currentLanguage),
      children: <Settings key={settingsKey} currentLanguage={currentLanguage} userInfo={userInfo} />
    }
  ]

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', alignItems: 'center' }}>
        <h2>
          <SettingOutlined style={{ marginRight: 8 }} />
          {t('configuration.title', currentLanguage)}
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