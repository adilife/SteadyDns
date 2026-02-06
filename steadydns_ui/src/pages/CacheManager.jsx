import { useState, useEffect, useCallback } from 'react'
import {
  Card,
  Row,
  Col,
  Button,
  Space,
  message,
  Input,
  Spin,
  Statistic,
  Progress,
  Alert,
  Table,
  Tag
} from 'antd'
import {
  DatabaseOutlined,
  ReloadOutlined,
  DeleteOutlined,
  SearchOutlined,
  InfoCircleOutlined
} from '@ant-design/icons'
import { t } from '../i18n'
import { apiClient } from '../utils/apiClient'

const CacheManager = ({ currentLanguage }) => {
  const [cacheStats, setCacheStats] = useState(null)
  const [loading, setLoading] = useState(false)
  const [clearLoading, setClearLoading] = useState(false)
  const [testDomain, setTestDomain] = useState('')

  // Load cache statistics from API
  const loadCacheStats = useCallback(async () => {
    setLoading(true)
    try {
      const response = await apiClient.getCacheStats()
      if (response.success) {
        setCacheStats(response.data)
      } else {
        message.error(response.message || t('cachemanager.fetchError', currentLanguage))
      }
    } catch (error) {
      console.error('Error loading cache stats:', error)
      message.error(t('cachemanager.fetchError', currentLanguage))
    } finally {
      setLoading(false)
    }
  }, [currentLanguage])

  // Load cache stats on component mount
  useEffect(() => {
    loadCacheStats()
  }, [loadCacheStats])

  // Auto refresh cache stats every 30 seconds
  useEffect(() => {
    const refreshInterval = setInterval(loadCacheStats, 30000)
    return () => clearInterval(refreshInterval)
  }, [loadCacheStats])



  // Clear entire cache
  const clearEntireCache = async () => {
    setClearLoading(true)
    try {
      const response = await apiClient.clearCache()
      if (response.success) {
        message.success(response.message || t('cachemanager.clearSuccess', currentLanguage))
        loadCacheStats()
      } else {
        message.error(response.message || t('cachemanager.clearError', currentLanguage))
      }
    } catch (error) {
      console.error('Error clearing cache:', error)
      message.error(t('cachemanager.clearError', currentLanguage))
    } finally {
      setClearLoading(false)
    }
  }

  // Clear cache for specific domain
  const clearDomainCache = async () => {
    if (!testDomain.trim()) {
      message.warning(t('cachemanager.domainRequired', currentLanguage))
      return
    }

    setClearLoading(true)
    try {
      const response = await apiClient.clearCache(testDomain)
      if (response.success) {
        message.success(response.message || t('cachemanager.clearDomainSuccess', currentLanguage))
        loadCacheStats()
        setTestDomain('')
      } else {
        message.error(response.message || t('cachemanager.clearDomainError', currentLanguage))
      }
    } catch (error) {
      console.error('Error clearing domain cache:', error)
      message.error(t('cachemanager.clearDomainError', currentLanguage))
    } finally {
      setClearLoading(false)
    }
  }

  // Cache size information
  const formatCacheSize = (bytes) => {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
  }

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2>
          <Space>
            <DatabaseOutlined />
            {t('cachemanager.title', currentLanguage)}
          </Space>
        </h2>
        <Button
          icon={<ReloadOutlined />}
          onClick={loadCacheStats}
          loading={loading}
        >
          {t('cachemanager.refresh', currentLanguage)}
        </Button>
      </div>

      <Spin spinning={loading}>
        {cacheStats ? (
          <>
            {/* Cache Statistics Cards */}
            <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
              <Col xs={24} sm={12} md={8}>
                <Card
                  title={t('cachemanager.totalItems', currentLanguage)}
                  variant="outlined"
                  hoverable
                >
                  <Statistic
                    value={cacheStats.count || 0}
                    precision={0}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} md={8}>
                <Card
                  title={t('cachemanager.hitCount', currentLanguage)}
                  variant="outlined"
                  hoverable
                >
                  <Statistic
                    value={cacheStats.hitCount || 0}
                    precision={0}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} md={8}>
                <Card
                  title={t('cachemanager.missCount', currentLanguage)}
                  variant="outlined"
                  hoverable
                >
                  <Statistic
                    value={cacheStats.missCount || 0}
                    precision={0}
                  />
                </Card>
              </Col>
            </Row>

            {/* Cache Hit Rate */}
            <Card title={t('cachemanager.hitRate', currentLanguage)} style={{ marginBottom: 24 }}>
              <div style={{ marginBottom: 16 }}>
                <Progress
                  percent={cacheStats.hitRate.toFixed(1)}
                  status="active"
                  strokeColor={{ from: '#52c41a', to: '#73d13d' }}
                  size="large"
                />
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-around' }}>
                <Statistic
                  title={t('cachemanager.hitRate', currentLanguage)}
                  value={cacheStats.hitRate.toFixed(1)}
                  suffix="%"
                  styles={{ content: { color: '#52c41a' } }}
                />
                <Statistic
                  title={t('cachemanager.missRate', currentLanguage)}
                  value={(100 - cacheStats.hitRate).toFixed(1)}
                  suffix="%"
                  styles={{ content: { color: '#ff4d4f' } }}
                />
              </div>
            </Card>

            {/* Cache Management */}
            <Card title={t('cachemanager.management', currentLanguage)} style={{ marginBottom: 24 }}>
              <Alert
                title={t('cachemanager.warning', currentLanguage)}
                description={t('cachemanager.clearWarning', currentLanguage)}
                type="warning"
                showIcon
                style={{ marginBottom: 16 }}
              />
              <Row gutter={[16, 16]}>
                <Col xs={24} sm={16}>
                  <Input
                    placeholder={t('cachemanager.domainPlaceholder', currentLanguage)}
                    value={testDomain}
                    onChange={(e) => setTestDomain(e.target.value)}
                    onPressEnter={clearDomainCache}
                    prefix={<SearchOutlined />}
                  />
                </Col>
                <Col xs={24} sm={8}>
                  <Space style={{ width: '100%' }}>
                    <Button
                      type="primary"
                      icon={<DeleteOutlined />}
                      onClick={clearDomainCache}
                      loading={clearLoading}
                      style={{ flex: 1 }}
                    >
                      {t('cachemanager.clearDomain', currentLanguage)}
                    </Button>
                    <Button
                      danger
                      icon={<DeleteOutlined />}
                      onClick={clearEntireCache}
                      loading={clearLoading}
                      style={{ flex: 1 }}
                    >
                      {t('cachemanager.clearAll', currentLanguage)}
                    </Button>
                  </Space>
                </Col>
              </Row>
            </Card>

            {/* Cache Details */}
            <Card title={t('cachemanager.details', currentLanguage)} style={{ marginBottom: 24 }}>
              <Row gutter={[16, 16]}>
                <Col xs={24} sm={12} md={8}>
                  <Statistic
                    title={t('cachemanager.currentSize', currentLanguage)}
                    value={formatCacheSize(cacheStats.currentSize || 0)}
                  />
                </Col>
                <Col xs={24} sm={12} md={8}>
                  <Statistic
                    title={t('cachemanager.maxSize', currentLanguage)}
                    value={formatCacheSize(cacheStats.maxSize || 0)}
                  />
                </Col>
                <Col xs={24} sm={12} md={8}>
                  <Statistic
                    title={t('cachemanager.usagePercent', currentLanguage)}
                    value={(cacheStats.usagePercent || 0).toFixed(2)}
                    suffix="%"
                  />
                </Col>
                <Col xs={24} sm={12} md={8}>
                  <Statistic
                    title={t('cachemanager.cleanupCount', currentLanguage)}
                    value={cacheStats.cleanupCount || 0}
                  />
                </Col>
                <Col xs={24} sm={12} md={8}>
                  <Statistic
                    title={t('cachemanager.evictionCount', currentLanguage)}
                    value={cacheStats.evictionCount || 0}
                  />
                </Col>
                <Col xs={24} sm={12} md={8}>
                  <Statistic
                    title={t('cachemanager.totalRequests', currentLanguage)}
                    value={cacheStats.totalRequests || 0}
                  />
                </Col>
              </Row>
            </Card>

            {/* Cache Information */}
            <Card title={t('cachemanager.information', currentLanguage)}>
              <Alert
                title={t('cachemanager.infoMessage', currentLanguage)}
                description={t('cachemanager.infoDescription', currentLanguage)}
                type="info"
                showIcon
              />
            </Card>
          </>
        ) : (
          <Alert
            title={t('cachemanager.loading', currentLanguage)}
            description={t('cachemanager.loadingDescription', currentLanguage)}
            type="info"
            showIcon
          />
        )}
      </Spin>
    </div>
  )
}

export default CacheManager