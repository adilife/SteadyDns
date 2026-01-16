import { useState, useEffect } from 'react'
import {
  Form,
  Input,
  Select,
  Switch,
  Button,
  Card,
  Space,
  message,
  InputNumber,
  Divider
} from 'antd'
import {
  SaveOutlined,
  ReloadOutlined,
  DatabaseOutlined,
  SafetyOutlined,
  SettingOutlined
} from '@ant-design/icons'
import { t } from '../i18n'

const { Option } = Select

// Mock server settings
const mockSettings = {
  basic: {
    port: 53,
    interface: '0.0.0.0',
    enableIPv6: true,
    enableTCP: true,
    enableUDP: true
  },
  upstream: {
    primaryDNS: '8.8.8.8',
    secondaryDNS: '8.8.4.4',
    timeout: 5000,
    retryTimes: 3
  },
  cache: {
    enabled: true,
    size: 10000,
    ttl: 3600
  },
  security: {
    enableRateLimit: true,
    maxQueriesPerSecond: 100,
    enableBlacklist: false,
    blacklistFile: '/etc/steadydns/blacklist.txt'
  }
}

const Settings = ({ currentLanguage, userInfo }) => {
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    // Initialize form with mock settings
    form.setFieldsValue(mockSettings)
  }, [form])

  const handleSave = () => {
    form.validateFields().then(values => {
        setLoading(true)
        // Simulate API call
        setTimeout(() => {
          console.log('Saved settings:', values)
          message.success(t('settings.settingsSaved', currentLanguage))
          setLoading(false)
        }, 500)
      }).catch(() => {
        message.error(currentLanguage === 'zh-CN' ? '请检查表单字段' : 'Please check form fields')
      })
  }

  const handleReset = () => {
    form.setFieldsValue(mockSettings)
    message.info(t('settings.settingsReset', currentLanguage))
  }

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <h2>{t('settings.title', currentLanguage)}</h2>
      </div>

      <Form
        form={form}
        layout="vertical"
        onFinish={handleSave}
      >
        <Card
          title={
            <Space>
              <SettingOutlined />
              {t('settings.basicSettings', currentLanguage)}
            </Space>
          }
          style={{ marginBottom: 16 }}
        >
          <Form.Item
            name={['basic', 'port']}
            label={t('settings.listeningPort', currentLanguage)}
            rules={[{ required: true, message: currentLanguage === 'zh-CN' ? '请输入端口' : 'Please input port' }]}
          >
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name={['basic', 'interface']}
            label={t('settings.listeningInterface', currentLanguage)}
            rules={[{ required: true, message: currentLanguage === 'zh-CN' ? '请输入接口' : 'Please input interface' }]}
          >
            <Input placeholder={currentLanguage === 'zh-CN' ? '例如：0.0.0.0 表示所有接口' : 'e.g., 0.0.0.0 for all interfaces'} />
          </Form.Item>

          <Space style={{ width: '100%', justifyContent: 'space-between' }}>
            <Form.Item
              name={['basic', 'enableIPv6']}
              label={t('settings.enableIPv6', currentLanguage)}
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>

            <Form.Item
              name={['basic', 'enableTCP']}
              label={t('settings.enableTCP', currentLanguage)}
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>

            <Form.Item
              name={['basic', 'enableUDP']}
              label={t('settings.enableUDP', currentLanguage)}
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>
          </Space>
        </Card>

        <Card
          title={
            <Space>
              <SettingOutlined />
              {t('settings.upstreamDns', currentLanguage)}
            </Space>
          }
          style={{ marginBottom: 16 }}
        >
          <Form.Item
            name={['upstream', 'primaryDNS']}
            label={t('settings.primaryDNS', currentLanguage)}
            rules={[{ required: true, message: currentLanguage === 'zh-CN' ? '请输入主DNS' : 'Please input primary DNS' }]}
          >
            <Input placeholder={currentLanguage === 'zh-CN' ? '例如：8.8.8.8' : 'e.g., 8.8.8.8'} />
          </Form.Item>

          <Form.Item
            name={['upstream', 'secondaryDNS']}
            label={t('settings.secondaryDNS', currentLanguage)}
            rules={[{ required: true, message: currentLanguage === 'zh-CN' ? '请输入备用DNS' : 'Please input secondary DNS' }]}
          >
            <Input placeholder={currentLanguage === 'zh-CN' ? '例如：8.8.4.4' : 'e.g., 8.8.4.4'} />
          </Form.Item>

          <Space style={{ width: '100%', justifyContent: 'space-between' }}>
            <Form.Item
              name={['upstream', 'timeout']}
              label={t('settings.timeout', currentLanguage)}
              rules={[{ required: true, message: currentLanguage === 'zh-CN' ? '请输入超时时间' : 'Please input timeout' }]}
            >
              <InputNumber min={1000} max={30000} style={{ width: 150 }} />
            </Form.Item>

            <Form.Item
              name={['upstream', 'retryTimes']}
              label={t('settings.retryTimes', currentLanguage)}
              rules={[{ required: true, message: currentLanguage === 'zh-CN' ? '请输入重试次数' : 'Please input retry times' }]}
            >
              <InputNumber min={1} max={10} style={{ width: 150 }} />
            </Form.Item>
          </Space>
        </Card>

        <Card
          title={
            <Space>
              <DatabaseOutlined />
              {t('settings.cacheSettings', currentLanguage)}
            </Space>
          }
          style={{ marginBottom: 16 }}
        >
          <Form.Item
            name={['cache', 'enabled']}
            label={t('settings.enableCache', currentLanguage)}
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>

          <Space style={{ width: '100%', justifyContent: 'space-between' }}>
            <Form.Item
              name={['cache', 'size']}
              label={t('settings.cacheSize', currentLanguage)}
              rules={[{ required: true, message: currentLanguage === 'zh-CN' ? '请输入缓存大小' : 'Please input cache size' }]}
            >
              <InputNumber min={100} max={1000000} style={{ width: 150 }} />
            </Form.Item>

            <Form.Item
              name={['cache', 'ttl']}
              label={t('settings.defaultTTL', currentLanguage)}
              rules={[{ required: true, message: currentLanguage === 'zh-CN' ? '请输入默认TTL' : 'Please input default TTL' }]}
            >
              <InputNumber min={60} max={86400} style={{ width: 150 }} />
            </Form.Item>
          </Space>
        </Card>

        <Card
          title={
            <Space>
              <SafetyOutlined />
              {t('settings.securitySettings', currentLanguage)}
            </Space>
          }
          style={{ marginBottom: 16 }}
        >
          <Form.Item
            name={['security', 'enableRateLimit']}
            label={t('settings.enableRateLimit', currentLanguage)}
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>

          <Form.Item
            name={['security', 'maxQueriesPerSecond']}
            label={t('settings.maxQueries', currentLanguage)}
            rules={[{ required: true, message: currentLanguage === 'zh-CN' ? '请输入最大查询速率' : 'Please input max queries per second' }]}
          >
            <InputNumber min={10} max={10000} style={{ width: 150 }} />
          </Form.Item>

          <Form.Item
            name={['security', 'enableBlacklist']}
            label={t('settings.enableBlacklist', currentLanguage)}
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>

          <Form.Item
            name={['security', 'blacklistFile']}
            label={t('settings.blacklistFile', currentLanguage)}
            rules={[{ required: true, message: currentLanguage === 'zh-CN' ? '请输入黑名单文件路径' : 'Please input blacklist file path' }]}
          >
            <Input placeholder={currentLanguage === 'zh-CN' ? '例如：/etc/steadydns/blacklist.txt' : 'e.g., /etc/steadydns/blacklist.txt'} />
          </Form.Item>
        </Card>

        <Divider />

        <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
          <Button
            icon={<ReloadOutlined />}
            onClick={handleReset}
          >
            {t('settings.reset', currentLanguage)}
          </Button>
          <Button
            type="primary"
            icon={<SaveOutlined />}
            onClick={handleSave}
            loading={loading}
          >
            {t('settings.saveSettings', currentLanguage)}
          </Button>
        </Space>
      </Form>
    </div>
  )
}

export default Settings