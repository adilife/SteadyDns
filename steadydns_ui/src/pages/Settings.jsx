import { useState, useEffect, useCallback, useMemo } from 'react'
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
  Modal
} from 'antd'
import {
  SaveOutlined,
  ReloadOutlined,
  DatabaseOutlined,
  SafetyOutlined,
  SettingOutlined,
  RollbackOutlined
} from '@ant-design/icons'
import { t } from '../i18n'
import { apiClient } from '../utils/apiClient'

const { Option } = Select

const Settings = ({ currentLanguage, userInfo }) => {
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)

  // Default settings structure
  const defaultSettings = useMemo(() => ({
    API: {
      LOG_ENABLED: true,
      LOG_LEVEL: 'debug',
      LOG_REQUEST_BODY: false,
      LOG_RESPONSE_BODY: false,
      RATE_LIMIT_ENABLED: true
    },
    APIServer: {
      API_SERVER_PORT: 8080,
      API_SERVER_IP_ADDR: '0.0.0.0',
      API_SERVER_IPV6_ADDR: '::',
      GIN_MODE: 'debug'
    },
    BIND: {
      BIND_ADDRESS: '127.0.0.1:5300',
      RNDC_KEY: '/etc/named/rndc.key',
      ZONE_FILE_PATH: '/usr/local/bind9/var/named',
      NAMED_CONF_PATH: '/etc/named',
      RNDC_PORT: 9530,
      BIND_USER: 'named',
      BIND_GROUP: 'named',
      BIND_EXEC_START: '/usr/local/bind9/sbin/named -u named',
      BIND_EXEC_RELOAD: '/usr/local/bind9/sbin/rndc -k $RNDC_KEY -s 127.0.0.1 -p $RNDC_PORT reload',
      BIND_EXEC_STOP: '/usr/local/bind9/sbin/rndc -k $RNDC_KEY -s 127.0.0.1 -p $RNDC_PORT stop',
      BIND_CHECKCONF_PATH: '/usr/local/bind9/bin/named-checkconf',
      BIND_CHECKZONE_PATH: '/usr/local/bind9/bin/named-checkzone'
    },
    Cache: {
      DNS_CACHE_SIZE_MB: 100,
      DNS_CACHE_CLEANUP_INTERVAL: 60,
      DNS_CACHE_ERROR_TTL: 3600
    },
    DNS: {
      DNS_CLIENT_WORKERS: 10000,
      DNS_QUEUE_MULTIPLIER: 2,
      DNS_PRIORITY_TIMEOUT_MS: 50
    },
    Database: {
      DB_PATH: 'steadydns.db'
    },
    JWT: {
      JWT_SECRET_KEY: 'your-default-jwt-secret-key-change-this-in-production',
      ACCESS_TOKEN_EXPIRATION: 30,
      REFRESH_TOKEN_EXPIRATION: 7,
      JWT_ALGORITHM: 'HS256'
    },
    Logging: {
      QUERY_LOG_PATH: 'log/',
      QUERY_LOG_MAX_SIZE: 10,
      QUERY_LOG_MAX_FILES: 10,
      DNS_LOG_LEVEL: 'debug'
    },
    Security: {
      DNS_RATE_LIMIT_PER_IP: 300,
      DNS_RATE_LIMIT_GLOBAL: 10000,
      DNS_BAN_DURATION: 5,
      DNS_MESSAGE_SIZE_LIMIT: 4096,
      DNS_VALIDATION_ENABLED: true
    }
  }), [])

  // Transform API config to form structure
  const transformConfigToForm = useCallback((apiConfig) => {
    /**
     * 将API返回的配置数据转换为表单结构
     * @param {Object} apiConfig - API返回的配置数据
     * @returns {Object} 转换后的表单结构数据
     */
    // 如果没有API配置数据，返回默认设置
    if (!apiConfig || !apiConfig.data) {
      return defaultSettings
    }

    const data = apiConfig.data
    
    // 构建表单结构，优先使用API返回的数据，否则使用默认值
    return {
      API: {
        LOG_ENABLED: data.API?.LOG_ENABLED === 'true' || defaultSettings.API.LOG_ENABLED,
        LOG_LEVEL: data.API?.LOG_LEVEL || defaultSettings.API.LOG_LEVEL,
        LOG_REQUEST_BODY: data.API?.LOG_REQUEST_BODY === 'true' || defaultSettings.API.LOG_REQUEST_BODY,
        LOG_RESPONSE_BODY: data.API?.LOG_RESPONSE_BODY === 'true' || defaultSettings.API.LOG_RESPONSE_BODY,
        RATE_LIMIT_ENABLED: data.API?.RATE_LIMIT_ENABLED === 'true' || defaultSettings.API.RATE_LIMIT_ENABLED
      },
      APIServer: {
        API_SERVER_PORT: data.APIServer?.API_SERVER_PORT ? parseInt(data.APIServer.API_SERVER_PORT) : defaultSettings.APIServer.API_SERVER_PORT,
        API_SERVER_IP_ADDR: data.APIServer?.API_SERVER_IP_ADDR || defaultSettings.APIServer.API_SERVER_IP_ADDR,
        API_SERVER_IPV6_ADDR: data.APIServer?.API_SERVER_IPV6_ADDR || defaultSettings.APIServer.API_SERVER_IPV6_ADDR,
        GIN_MODE: data.APIServer?.GIN_MODE || defaultSettings.APIServer.GIN_MODE
      },
      BIND: {
        BIND_ADDRESS: data.BIND?.BIND_ADDRESS || defaultSettings.BIND.BIND_ADDRESS,
        RNDC_KEY: data.BIND?.RNDC_KEY || defaultSettings.BIND.RNDC_KEY,
        ZONE_FILE_PATH: data.BIND?.ZONE_FILE_PATH || defaultSettings.BIND.ZONE_FILE_PATH,
        NAMED_CONF_PATH: data.BIND?.NAMED_CONF_PATH || defaultSettings.BIND.NAMED_CONF_PATH,
        RNDC_PORT: data.BIND?.RNDC_PORT ? parseInt(data.BIND.RNDC_PORT) : defaultSettings.BIND.RNDC_PORT,
        BIND_USER: data.BIND?.BIND_USER || defaultSettings.BIND.BIND_USER,
        BIND_GROUP: data.BIND?.BIND_GROUP || defaultSettings.BIND.BIND_GROUP,
        BIND_EXEC_START: data.BIND?.BIND_EXEC_START || defaultSettings.BIND.BIND_EXEC_START,
        BIND_EXEC_RELOAD: data.BIND?.BIND_EXEC_RELOAD || defaultSettings.BIND.BIND_EXEC_RELOAD,
        BIND_EXEC_STOP: data.BIND?.BIND_EXEC_STOP || defaultSettings.BIND.BIND_EXEC_STOP,
        BIND_CHECKCONF_PATH: data.BIND?.BIND_CHECKCONF_PATH || defaultSettings.BIND.BIND_CHECKCONF_PATH,
        BIND_CHECKZONE_PATH: data.BIND?.BIND_CHECKZONE_PATH || defaultSettings.BIND.BIND_CHECKZONE_PATH
      },
      Cache: {
        DNS_CACHE_SIZE_MB: data.Cache?.DNS_CACHE_SIZE_MB ? parseInt(data.Cache.DNS_CACHE_SIZE_MB) : defaultSettings.Cache.DNS_CACHE_SIZE_MB,
        DNS_CACHE_CLEANUP_INTERVAL: data.Cache?.DNS_CACHE_CLEANUP_INTERVAL ? parseInt(data.Cache.DNS_CACHE_CLEANUP_INTERVAL) : defaultSettings.Cache.DNS_CACHE_CLEANUP_INTERVAL,
        DNS_CACHE_ERROR_TTL: data.Cache?.DNS_CACHE_ERROR_TTL ? parseInt(data.Cache.DNS_CACHE_ERROR_TTL) : defaultSettings.Cache.DNS_CACHE_ERROR_TTL
      },
      DNS: {
        DNS_CLIENT_WORKERS: data.DNS?.DNS_CLIENT_WORKERS ? parseInt(data.DNS.DNS_CLIENT_WORKERS) : defaultSettings.DNS.DNS_CLIENT_WORKERS,
        DNS_QUEUE_MULTIPLIER: data.DNS?.DNS_QUEUE_MULTIPLIER ? parseInt(data.DNS.DNS_QUEUE_MULTIPLIER) : defaultSettings.DNS.DNS_QUEUE_MULTIPLIER,
        DNS_PRIORITY_TIMEOUT_MS: data.DNS?.DNS_PRIORITY_TIMEOUT_MS ? parseInt(data.DNS.DNS_PRIORITY_TIMEOUT_MS) : defaultSettings.DNS.DNS_PRIORITY_TIMEOUT_MS
      },
      Database: {
        DB_PATH: data.Database?.DB_PATH || defaultSettings.Database.DB_PATH
      },
      JWT: {
        JWT_SECRET_KEY: data.JWT?.JWT_SECRET_KEY || defaultSettings.JWT.JWT_SECRET_KEY,
        ACCESS_TOKEN_EXPIRATION: data.JWT?.ACCESS_TOKEN_EXPIRATION ? parseInt(data.JWT.ACCESS_TOKEN_EXPIRATION) : defaultSettings.JWT.ACCESS_TOKEN_EXPIRATION,
        REFRESH_TOKEN_EXPIRATION: data.JWT?.REFRESH_TOKEN_EXPIRATION ? parseInt(data.JWT.REFRESH_TOKEN_EXPIRATION) : defaultSettings.JWT.REFRESH_TOKEN_EXPIRATION,
        JWT_ALGORITHM: data.JWT?.JWT_ALGORITHM || defaultSettings.JWT.JWT_ALGORITHM
      },
      Logging: {
        QUERY_LOG_PATH: data.Logging?.QUERY_LOG_PATH || defaultSettings.Logging.QUERY_LOG_PATH,
        QUERY_LOG_MAX_SIZE: data.Logging?.QUERY_LOG_MAX_SIZE ? parseInt(data.Logging.QUERY_LOG_MAX_SIZE) : defaultSettings.Logging.QUERY_LOG_MAX_SIZE,
        QUERY_LOG_MAX_FILES: data.Logging?.QUERY_LOG_MAX_FILES ? parseInt(data.Logging.QUERY_LOG_MAX_FILES) : defaultSettings.Logging.QUERY_LOG_MAX_FILES,
        DNS_LOG_LEVEL: data.Logging?.DNS_LOG_LEVEL || defaultSettings.Logging.DNS_LOG_LEVEL
      },
      Security: {
        DNS_RATE_LIMIT_PER_IP: data.Security?.DNS_RATE_LIMIT_PER_IP ? parseInt(data.Security.DNS_RATE_LIMIT_PER_IP) : defaultSettings.Security.DNS_RATE_LIMIT_PER_IP,
        DNS_RATE_LIMIT_GLOBAL: data.Security?.DNS_RATE_LIMIT_GLOBAL ? parseInt(data.Security.DNS_RATE_LIMIT_GLOBAL) : defaultSettings.Security.DNS_RATE_LIMIT_GLOBAL,
        DNS_BAN_DURATION: data.Security?.DNS_BAN_DURATION ? parseInt(data.Security.DNS_BAN_DURATION) : defaultSettings.Security.DNS_BAN_DURATION,
        DNS_MESSAGE_SIZE_LIMIT: data.Security?.DNS_MESSAGE_SIZE_LIMIT ? parseInt(data.Security.DNS_MESSAGE_SIZE_LIMIT) : defaultSettings.Security.DNS_MESSAGE_SIZE_LIMIT,
        DNS_VALIDATION_ENABLED: data.Security?.DNS_VALIDATION_ENABLED === 'true' || defaultSettings.Security.DNS_VALIDATION_ENABLED
      }
    }
  }, [defaultSettings])

  // Load configuration
  const loadConfig = useCallback(async () => {
    /**
     * 加载配置数据
     * 从API获取配置，转换为表单结构并设置到表单中
     */
    try {
      setLoading(true)
      // Fetch configuration
      const configResponse = await apiClient.getConfig()
      if (configResponse.success) {
        // Transform API response to form structure
        const transformedConfig = transformConfigToForm(configResponse)
        form.setFieldsValue(transformedConfig)
      } else {
        // Use default settings if API fails
        form.setFieldsValue(defaultSettings)
      }
    } catch (error) {
      console.error('Error loading config:', error)
      // Use default settings on error
      form.setFieldsValue(defaultSettings)
    } finally {
      setLoading(false)
    }
  }, [form, transformConfigToForm, defaultSettings])

  // Load configuration from API
  useEffect(() => {
    loadConfig()
  }, [loadConfig])

  // Transform form values to API config structure
  const transformFormToConfig = (formValues) => {
    /**
     * 将表单值转换为API配置结构
     * @param {Object} formValues - 表单值
     * @returns {Object} 转换后的API配置结构
     */
    // 构建API所需的配置结构
    return {
      API: {
        LOG_ENABLED: formValues.API.LOG_ENABLED.toString(),
        LOG_LEVEL: formValues.API.LOG_LEVEL || defaultSettings.API.LOG_LEVEL,
        LOG_REQUEST_BODY: formValues.API.LOG_REQUEST_BODY.toString(),
        LOG_RESPONSE_BODY: formValues.API.LOG_RESPONSE_BODY.toString(),
        RATE_LIMIT_ENABLED: formValues.API.RATE_LIMIT_ENABLED.toString()
      },
      APIServer: {
        API_SERVER_PORT: formValues.APIServer.API_SERVER_PORT.toString(),
        API_SERVER_IP_ADDR: formValues.APIServer.API_SERVER_IP_ADDR,
        API_SERVER_IPV6_ADDR: formValues.APIServer.API_SERVER_IPV6_ADDR,
        GIN_MODE: formValues.APIServer.GIN_MODE
      },
      BIND: {
        BIND_ADDRESS: formValues.BIND.BIND_ADDRESS,
        RNDC_KEY: formValues.BIND.RNDC_KEY,
        ZONE_FILE_PATH: formValues.BIND.ZONE_FILE_PATH,
        NAMED_CONF_PATH: formValues.BIND.NAMED_CONF_PATH,
        RNDC_PORT: formValues.BIND.RNDC_PORT.toString(),
        BIND_USER: formValues.BIND.BIND_USER,
        BIND_GROUP: formValues.BIND.BIND_GROUP,
        BIND_EXEC_START: formValues.BIND.BIND_EXEC_START,
        BIND_EXEC_RELOAD: formValues.BIND.BIND_EXEC_RELOAD,
        BIND_EXEC_STOP: formValues.BIND.BIND_EXEC_STOP,
        BIND_CHECKCONF_PATH: formValues.BIND.BIND_CHECKCONF_PATH,
        BIND_CHECKZONE_PATH: formValues.BIND.BIND_CHECKZONE_PATH
      },
      Cache: {
        DNS_CACHE_SIZE_MB: formValues.Cache.DNS_CACHE_SIZE_MB.toString(),
        DNS_CACHE_CLEANUP_INTERVAL: formValues.Cache.DNS_CACHE_CLEANUP_INTERVAL.toString(),
        DNS_CACHE_ERROR_TTL: formValues.Cache.DNS_CACHE_ERROR_TTL.toString()
      },
      DNS: {
        DNS_CLIENT_WORKERS: formValues.DNS.DNS_CLIENT_WORKERS.toString(),
        DNS_QUEUE_MULTIPLIER: formValues.DNS.DNS_QUEUE_MULTIPLIER.toString(),
        DNS_PRIORITY_TIMEOUT_MS: formValues.DNS.DNS_PRIORITY_TIMEOUT_MS.toString()
      },
      Database: {
        DB_PATH: formValues.Database.DB_PATH
      },
      JWT: {
        JWT_SECRET_KEY: formValues.JWT.JWT_SECRET_KEY,
        ACCESS_TOKEN_EXPIRATION: formValues.JWT.ACCESS_TOKEN_EXPIRATION.toString(),
        REFRESH_TOKEN_EXPIRATION: formValues.JWT.REFRESH_TOKEN_EXPIRATION.toString(),
        JWT_ALGORITHM: formValues.JWT.JWT_ALGORITHM
      },
      Logging: {
        QUERY_LOG_PATH: formValues.Logging.QUERY_LOG_PATH,
        QUERY_LOG_MAX_SIZE: formValues.Logging.QUERY_LOG_MAX_SIZE.toString(),
        QUERY_LOG_MAX_FILES: formValues.Logging.QUERY_LOG_MAX_FILES.toString(),
        DNS_LOG_LEVEL: formValues.Logging.DNS_LOG_LEVEL || defaultSettings.Logging.DNS_LOG_LEVEL
      },
      Security: {
        DNS_RATE_LIMIT_PER_IP: formValues.Security.DNS_RATE_LIMIT_PER_IP.toString(),
        DNS_RATE_LIMIT_GLOBAL: formValues.Security.DNS_RATE_LIMIT_GLOBAL.toString(),
        DNS_BAN_DURATION: formValues.Security.DNS_BAN_DURATION.toString(),
        DNS_MESSAGE_SIZE_LIMIT: formValues.Security.DNS_MESSAGE_SIZE_LIMIT.toString(),
        DNS_VALIDATION_ENABLED: formValues.Security.DNS_VALIDATION_ENABLED.toString()
      }
    }
  }

  // Save configuration
  const handleSave = () => {
    form.validateFields().then(async values => {
        setLoading(true)
        try {
          // Transform form values to API structure
          const configData = transformFormToConfig(values)
          
          // Save each section
          for (const section in configData) {
            for (const key in configData[section]) {
              await apiClient.updateConfig(section, key, configData[section][key], userInfo.username)
            }
          }
          
          // Reload configuration
          await apiClient.reloadConfig()
          
          message.success(t('settings.settingsSaved', currentLanguage))
        } catch (error) {
          console.error('Error saving settings:', error)
          message.error(t('settings.saveError', currentLanguage))
        } finally {
          setLoading(false)
        }
      }).catch(() => {
        message.error(t('settings.pleaseCheckFormFields', currentLanguage))
      })
  }

  // Reset configuration
  const handleReset = async () => {
    try {
      setLoading(true)
      await apiClient.resetConfig(userInfo.username)
      await apiClient.reloadConfig()
      await loadConfig()
      message.success(t('settings.settingsReset', currentLanguage))
    } catch (error) {
      console.error('Error resetting settings:', error)
      message.error(t('settings.resetError', currentLanguage))
    } finally {
      setLoading(false)
    }
  }

  // Reload configuration
  const handleReload = async () => {
    try {
      setLoading(true)
      await apiClient.reloadConfig()
      await loadConfig()
      message.success(t('settings.settingsReloaded', currentLanguage))
    } catch (error) {
      console.error('Error reloading settings:', error)
      message.error(t('settings.reloadError', currentLanguage))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <h2>{t('settings.title', currentLanguage)}</h2>
      </div>

      <Space style={{ width: '100%', justifyContent: 'flex-end', marginBottom: 16 }}>
        <Button
          icon={<RollbackOutlined />}
          onClick={handleReset}
        >
          {t('settings.reset', currentLanguage)}
        </Button>
        <Button
          icon={<ReloadOutlined />}
          onClick={handleReload}
        >
          {t('settings.reload', currentLanguage)}
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

      <Form
        form={form}
        layout="vertical"
        onFinish={handleSave}
      >
        {/* API配置卡片 */}
        <Card
          title={
            <Space>
              <SettingOutlined />
              {t('settings.apiConfig', currentLanguage)}
            </Space>
          }
          style={{ marginBottom: 16 }}
          collapsible
        >
          <Space style={{ width: '100%', justifyContent: 'space-between' }}>
            <Form.Item
              name={['API', 'LOG_ENABLED']}
              label={t('settings.logEnabled', currentLanguage)}
              valuePropName="checked"
              tooltip={t('settings.logEnabledTooltip', currentLanguage)}
            >
              <Switch />
            </Form.Item>

            <Form.Item
              name={['API', 'RATE_LIMIT_ENABLED']}
              label={t('settings.rateLimitEnabled', currentLanguage)}
              valuePropName="checked"
              tooltip={t('settings.rateLimitEnabledTooltip', currentLanguage)}
            >
              <Switch />
            </Form.Item>

            <Form.Item
              name={['API', 'LOG_REQUEST_BODY']}
              label={t('settings.logRequestBody', currentLanguage)}
              valuePropName="checked"
              tooltip={t('settings.logRequestBodyTooltip', currentLanguage)}
            >
              <Switch />
            </Form.Item>

            <Form.Item
              name={['API', 'LOG_RESPONSE_BODY']}
              label={t('settings.logResponseBody', currentLanguage)}
              valuePropName="checked"
              tooltip={t('settings.logResponseBodyTooltip', currentLanguage)}
            >
              <Switch />
            </Form.Item>
          </Space>
        </Card>

        {/* API服务器配置卡片 */}
        <Card
          title={
            <Space>
              <SettingOutlined />
              {t('settings.apiServerConfig', currentLanguage)}
            </Space>
          }
          style={{ marginBottom: 16 }}
          collapsible
        >
          <Form.Item
            name={['APIServer', 'API_SERVER_PORT']}
            label={t('settings.apiServerPort', currentLanguage)}
            rules={[{ required: true, message: t('settings.pleaseInputPort', currentLanguage) }]}
            tooltip={t('settings.apiServerPortTooltip', currentLanguage)}
          >
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name={['APIServer', 'API_SERVER_IP_ADDR']}
            label={t('settings.apiServerIpAddr', currentLanguage)}
            rules={[{ required: true, message: t('settings.pleaseInputIpv4Address', currentLanguage) }]}
            tooltip={t('settings.apiServerIpAddrTooltip', currentLanguage)}
          >
            <Input placeholder="例如：0.0.0.0 表示所有接口" />
          </Form.Item>

          <Form.Item
            name={['APIServer', 'API_SERVER_IPV6_ADDR']}
            label={t('settings.apiServerIpv6Addr', currentLanguage)}
            tooltip={t('settings.apiServerIpv6AddrTooltip', currentLanguage)}
          >
            <Input placeholder={currentLanguage === 'zh-CN' ? '例如：:: 表示所有接口' : 'e.g.: :: for all interfaces'} />
          </Form.Item>

          <Form.Item
            name={['APIServer', 'GIN_MODE']}
            label={t('settings.ginMode', currentLanguage)}
            tooltip={t('settings.ginModeTooltip', currentLanguage)}
          >
            <Select style={{ width: '100%' }}>
              <Option value="debug">debug</Option>
              <Option value="release">release</Option>
            </Select>
          </Form.Item>
        </Card>

        {/* BIND服务器配置卡片 */}
        <Card
          title={
            <Space>
              <SettingOutlined />
              {t('settings.bindServerConfig', currentLanguage)}
            </Space>
          }
          style={{ marginBottom: 16 }}
          collapsible
        >
          <Form.Item
            name={['BIND', 'BIND_ADDRESS']}
            label={t('settings.bindAddress', currentLanguage)}
            tooltip={t('settings.bindAddressTooltip', currentLanguage)}
          >
            <Input disabled={true} />
          </Form.Item>

          <Form.Item
            name={['BIND', 'RNDC_KEY']}
            label={t('settings.rndcKey', currentLanguage)}
            tooltip={t('settings.rndcKeyTooltip', currentLanguage)}
          >
            <Input disabled={true} />
          </Form.Item>

          <Form.Item
            name={['BIND', 'ZONE_FILE_PATH']}
            label={t('settings.zoneFilePath', currentLanguage)}
            tooltip={t('settings.zoneFilePathTooltip', currentLanguage)}
          >
            <Input disabled={true} />
          </Form.Item>

          <Form.Item
            name={['BIND', 'NAMED_CONF_PATH']}
            label={t('settings.namedConfPath', currentLanguage)}
            tooltip={t('settings.namedConfPathTooltip', currentLanguage)}
          >
            <Input disabled={true} />
          </Form.Item>

          <Form.Item
            name={['BIND', 'RNDC_PORT']}
            label={t('settings.rndcPort', currentLanguage)}
            tooltip={t('settings.rndcPortTooltip', currentLanguage)}
          >
            <InputNumber min={1} max={65535} style={{ width: '100%' }} disabled={true} />
          </Form.Item>

          <Form.Item
            name={['BIND', 'BIND_USER']}
            label={t('settings.bindUser', currentLanguage)}
            tooltip={t('settings.bindUserTooltip', currentLanguage)}
          >
            <Input disabled={true} />
          </Form.Item>

          <Form.Item
            name={['BIND', 'BIND_GROUP']}
            label={t('settings.bindGroup', currentLanguage)}
            tooltip={t('settings.bindGroupTooltip', currentLanguage)}
          >
            <Input disabled={true} />
          </Form.Item>

          <Form.Item
            name={['BIND', 'BIND_EXEC_START']}
            label={t('settings.bindExecStart', currentLanguage)}
            tooltip={t('settings.bindExecStartTooltip', currentLanguage)}
          >
            <Input disabled={true} />
          </Form.Item>

          <Form.Item
            name={['BIND', 'BIND_EXEC_RELOAD']}
            label={t('settings.bindExecReload', currentLanguage)}
            tooltip={t('settings.bindExecReloadTooltip', currentLanguage)}
          >
            <Input disabled={true} />
          </Form.Item>

          <Form.Item
            name={['BIND', 'BIND_EXEC_STOP']}
            label={t('settings.bindExecStop', currentLanguage)}
            tooltip={t('settings.bindExecStopTooltip', currentLanguage)}
          >
            <Input disabled={true} />
          </Form.Item>

          <Form.Item
            name={['BIND', 'BIND_CHECKCONF_PATH']}
            label={t('settings.bindCheckconfPath', currentLanguage)}
            tooltip={t('settings.bindCheckconfPathTooltip', currentLanguage)}
          >
            <Input disabled={true} />
          </Form.Item>

          <Form.Item
            name={['BIND', 'BIND_CHECKZONE_PATH']}
            label={t('settings.bindCheckzonePath', currentLanguage)}
            tooltip={t('settings.bindCheckzonePathTooltip', currentLanguage)}
          >
            <Input disabled={true} />
          </Form.Item>
        </Card>

        {/* 缓存配置卡片 */}
        <Card
          title={
            <Space>
              <DatabaseOutlined />
              {t('settings.cacheConfig', currentLanguage)}
            </Space>
          }
          style={{ marginBottom: 16 }}
          collapsible
        >
          <Form.Item
            name={['Cache', 'DNS_CACHE_SIZE_MB']}
            label={t('settings.dnsCacheSizeMb', currentLanguage)}
            tooltip={t('settings.dnsCacheSizeMbTooltip', currentLanguage)}
          >
            <InputNumber min={1} max={10000} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name={['Cache', 'DNS_CACHE_CLEANUP_INTERVAL']}
            label={t('settings.dnsCacheCleanupInterval', currentLanguage)}
            tooltip={t('settings.dnsCacheCleanupIntervalTooltip', currentLanguage)}
          >
            <InputNumber min={1} max={3600} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name={['Cache', 'DNS_CACHE_ERROR_TTL']}
            label={t('settings.dnsCacheErrorTtl', currentLanguage)}
            tooltip={t('settings.dnsCacheErrorTtlTooltip', currentLanguage)}
          >
            <InputNumber min={1} max={86400} style={{ width: '100%' }} />
          </Form.Item>
        </Card>

        {/* DNS配置卡片 */}
        <Card
          title={
            <Space>
              <SettingOutlined />
              {t('settings.dnsConfig', currentLanguage)}
            </Space>
          }
          style={{ marginBottom: 16 }}
          collapsible
        >
          <Form.Item
            name={['DNS', 'DNS_CLIENT_WORKERS']}
            label={t('settings.dnsClientWorkers', currentLanguage)}
            tooltip={t('settings.dnsClientWorkersTooltip', currentLanguage)}
          >
            <InputNumber min={100} max={100000} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name={['DNS', 'DNS_QUEUE_MULTIPLIER']}
            label={t('settings.dnsQueueMultiplier', currentLanguage)}
            tooltip={t('settings.dnsQueueMultiplierTooltip', currentLanguage)}
          >
            <InputNumber min={1} max={10} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name={['DNS', 'DNS_PRIORITY_TIMEOUT_MS']}
            label={t('settings.dnsPriorityTimeoutMs', currentLanguage)}
            tooltip={t('settings.dnsPriorityTimeoutMsTooltip', currentLanguage)}
          >
            <InputNumber min={1} max={1000} style={{ width: '100%' }} />
          </Form.Item>
        </Card>

        {/* 数据库配置卡片 */}
        <Card
          title={
            <Space>
              <DatabaseOutlined />
              {t('settings.databaseConfig', currentLanguage)}
            </Space>
          }
          style={{ marginBottom: 16 }}
          collapsible
        >
          <Form.Item
            name={['Database', 'DB_PATH']}
            label={t('settings.dbPath', currentLanguage)}
            tooltip={t('settings.dbPathTooltip', currentLanguage)}
          >
            <Input disabled={true} />
          </Form.Item>
        </Card>

        {/* JWT配置卡片 */}
        <Card
          title={
            <Space>
              <SafetyOutlined />
              {t('settings.jwtConfig', currentLanguage)}
            </Space>
          }
          style={{ marginBottom: 16 }}
          collapsible
        >
          <Form.Item
            name={['JWT', 'JWT_SECRET_KEY']}
            label={t('settings.jwtSecretKey', currentLanguage)}
            tooltip={t('settings.jwtSecretKeyTooltip', currentLanguage)}
          >
            <Input />
          </Form.Item>

          <Form.Item
            name={['JWT', 'ACCESS_TOKEN_EXPIRATION']}
            label={t('settings.accessTokenExpiration', currentLanguage)}
            tooltip={t('settings.accessTokenExpirationTooltip', currentLanguage)}
          >
            <InputNumber min={1} max={1440} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name={['JWT', 'REFRESH_TOKEN_EXPIRATION']}
            label={t('settings.refreshTokenExpiration', currentLanguage)}
            tooltip={t('settings.refreshTokenExpirationTooltip', currentLanguage)}
          >
            <InputNumber min={1} max={365} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name={['JWT', 'JWT_ALGORITHM']}
            label={t('settings.jwtAlgorithm', currentLanguage)}
            tooltip={t('settings.jwtAlgorithmTooltip', currentLanguage)}
          >
            <Select style={{ width: '100%' }} disabled={true}>
              <Option value="HS256">HS256</Option>
              <Option value="HS384">HS384</Option>
              <Option value="HS512">HS512</Option>
            </Select>
          </Form.Item>
        </Card>

        {/* 日志配置卡片 */}
        <Card
          title={
            <Space>
              <SettingOutlined />
              {t('settings.loggingConfig', currentLanguage)}
            </Space>
          }
          style={{ marginBottom: 16 }}
          collapsible
        >
          <Form.Item
            name={['Logging', 'QUERY_LOG_PATH']}
            label={t('settings.queryLogPath', currentLanguage)}
            tooltip={t('settings.queryLogPathTooltip', currentLanguage)}
          >
            <Input />
          </Form.Item>

          <Space style={{ width: '100%', justifyContent: 'space-between' }}>
            <Form.Item
              name={['Logging', 'QUERY_LOG_MAX_SIZE']}
              label={t('settings.queryLogMaxSize', currentLanguage)}
              tooltip={t('settings.queryLogMaxSizeTooltip', currentLanguage)}
            >
              <InputNumber min={1} max={1000} style={{ width: 150 }} />
            </Form.Item>

            <Form.Item
              name={['Logging', 'QUERY_LOG_MAX_FILES']}
              label={t('settings.queryLogMaxFiles', currentLanguage)}
              tooltip={t('settings.queryLogMaxFilesTooltip', currentLanguage)}
            >
              <InputNumber min={1} max={100} style={{ width: 150 }} />
            </Form.Item>
          </Space>
        </Card>

        {/* 安全配置卡片 */}
        <Card
          title={
            <Space>
              <SafetyOutlined />
              {t('settings.securityConfig', currentLanguage)}
            </Space>
          }
          style={{ marginBottom: 16 }}
          collapsible
        >
          <Form.Item
            name={['Security', 'DNS_VALIDATION_ENABLED']}
            label={t('settings.dnsValidationEnabled', currentLanguage)}
            valuePropName="checked"
            tooltip={t('settings.dnsValidationEnabledTooltip', currentLanguage)}
          >
            <Switch />
          </Form.Item>

          <Form.Item
            name={['Security', 'DNS_RATE_LIMIT_PER_IP']}
            label={t('settings.dnsRateLimitPerIp', currentLanguage)}
            tooltip={t('settings.dnsRateLimitPerIpTooltip', currentLanguage)}
          >
            <InputNumber min={1} max={10000} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name={['Security', 'DNS_RATE_LIMIT_GLOBAL']}
            label={t('settings.dnsRateLimitGlobal', currentLanguage)}
            tooltip={t('settings.dnsRateLimitGlobalTooltip', currentLanguage)}
          >
            <InputNumber min={1} max={100000} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name={['Security', 'DNS_BAN_DURATION']}
            label={t('settings.dnsBanDuration', currentLanguage)}
            tooltip={t('settings.dnsBanDurationTooltip', currentLanguage)}
          >
            <InputNumber min={1} max={60} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name={['Security', 'DNS_MESSAGE_SIZE_LIMIT']}
            label={t('settings.dnsMessageSizeLimit', currentLanguage)}
            tooltip={t('settings.dnsMessageSizeLimitTooltip', currentLanguage)}
          >
            <InputNumber min={512} max={65535} style={{ width: '100%' }} />
          </Form.Item>
        </Card>
      </Form>
    </div>
  )
}

export default Settings