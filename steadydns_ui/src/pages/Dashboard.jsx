import { useState, useEffect, useCallback } from 'react'
import { Card, Row, Col, Statistic, Table, Progress, Select, Space, Typography, Spin, Button } from 'antd'
import { 
  AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend,
  BarChart, Bar, PieChart, Pie, Cell, LineChart, Line
} from 'recharts'
import { 
  DashboardOutlined, 
  GlobalOutlined, 
  DatabaseOutlined, 
  AppstoreOutlined, 
  BarChartOutlined, 
  UserOutlined 
} from '@ant-design/icons'
import { t } from '../i18n'
import { apiClient } from '../utils/apiClient'

const { Title, Text } = Typography
const { Option } = Select

const Dashboard = ({ currentLanguage, userInfo }) => {
  const [loading, setLoading] = useState(false)
  const [timeRange, setTimeRange] = useState('1h')
  const [error, setError] = useState(null)
  const [dashboardData, setDashboardData] = useState({
    systemStats: {
      totalQueries: 0,
      qps: 0,
      cacheHitRate: 0,
      systemHealth: 0,
      activeServers: 0
    },
    serverStatus: null,
    forwardServers: [],
    cacheStats: {
      size: '0 GB',
      maxSize: '0 GB',
      hitRate: 0,
      missRate: 0,
      items: 0
    },
    systemResources: {
      cpu: 0,
      memory: 0,
      disk: 0,
      network: {
        inbound: '0 MB/s',
        outbound: '0 MB/s'
      }
    },
    topDomains: [],
    topClients: [],
    qpsTrend: [],
    latencyData: [],
    resourceUsage: []
  })

  // Get summary data from API
  const fetchSummaryData = useCallback(async () => {
    try {
      setError(null)
      
      // Fetch server status
      const serverStatusResponse = await apiClient.getServerStatus()
      if (serverStatusResponse.success) {
        setDashboardData(prev => ({
          ...prev,
          serverStatus: serverStatusResponse.data
        }))
      }
      
      // Fetch health status
      const healthStatusResponse = await apiClient.getHealthStatus()
      if (healthStatusResponse.success) {
        setDashboardData(prev => ({
          ...prev,
          systemStats: {
            totalQueries: 0,
            qps: 0,
            cacheHitRate: 0,
            systemHealth: healthStatusResponse.data.status === 'healthy' ? 100 : 0,
            activeServers: healthStatusResponse.data.dns.is_running ? 1 : 0
          },
          systemResources: {
            cpu: healthStatusResponse.data.system.cpu || 0,
            memory: 0,
            disk: 0,
            network: {
              inbound: '0 MB/s',
              outbound: '0 MB/s'
            }
          }
        }))
      }
      
      // Fetch cache stats
      const cacheStatsResponse = await apiClient.getCacheStats()
      if (cacheStatsResponse.success) {
        const cacheData = cacheStatsResponse.data
        // Calculate miss rate
        const missRate = 100 - cacheData.hitRate
        // Convert bytes to MB
        const currentSizeMB = (cacheData.currentSize / (1024 * 1024)).toFixed(2)
        const maxSizeMB = (cacheData.maxSize / (1024 * 1024)).toFixed(2)
        
        setDashboardData(prev => ({
          ...prev,
          cacheStats: {
            size: `${currentSizeMB} MB`,
            maxSize: `${maxSizeMB} MB`,
            hitRate: cacheData.hitRate,
            missRate: missRate,
            items: cacheData.count
          },
          systemStats: {
            ...prev.systemStats,
            cacheHitRate: cacheData.hitRate
          }
        }))
      }
      
    } catch (error) {
      console.error('Error fetching summary data:', error)
      setError(t('dashboard.fetchError', currentLanguage) || 'Failed to fetch data')
    }
  }, [currentLanguage])

  // Get trends data from API
  const fetchTrendsData = useCallback(async () => {
    try {
      // Mock trends data for now
      const mockQpsTrend = Array.from({ length: 12 }, (_, i) => ({
        time: `${i}:00`,
        qps: Math.floor(Math.random() * 100) + 10
      }))
      
      const mockLatencyData = [
        { range: '< 10ms', count: Math.floor(Math.random() * 500) + 100 },
        { range: '10-50ms', count: Math.floor(Math.random() * 300) + 50 },
        { range: '50-100ms', count: Math.floor(Math.random() * 200) + 20 },
        { range: '100-200ms', count: Math.floor(Math.random() * 100) + 10 },
        { range: '> 200ms', count: Math.floor(Math.random() * 50) + 5 }
      ]
      
      const mockResourceUsage = Array.from({ length: 12 }, (_, i) => ({
        time: `${i}:00`,
        cpu: Math.floor(Math.random() * 50) + 10,
        memory: Math.floor(Math.random() * 40) + 20,
        disk: Math.floor(Math.random() * 30) + 10
      }))
      
      setDashboardData(prev => ({
        ...prev,
        qpsTrend: mockQpsTrend,
        latencyData: mockLatencyData,
        resourceUsage: mockResourceUsage
      }))
    } catch (error) {
      console.error('Error fetching trends data:', error)
    }
  }, [])

  // Get top data from API
  const fetchTopData = useCallback(async () => {
    try {
      // Mock top data for now
      const mockTopDomains = Array.from({ length: 10 }, (_, i) => ({
        rank: i + 1,
        domain: `example${i + 1}.com`,
        queries: Math.floor(Math.random() * 1000) + 100,
        percentage: (Math.random() * 20).toFixed(1)
      }))
      
      const mockTopClients = Array.from({ length: 10 }, (_, i) => ({
        rank: i + 1,
        ip: `192.168.1.${i + 1}`,
        queries: Math.floor(Math.random() * 500) + 50,
        percentage: (Math.random() * 15).toFixed(1)
      }))
      
      setDashboardData(prev => ({
        ...prev,
        topDomains: mockTopDomains,
        topClients: mockTopClients
      }))
    } catch (error) {
      console.error('Error fetching top data:', error)
    }
  }, [])

  // Refresh all data
  const refreshAllData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      await Promise.all([
        fetchSummaryData(),
        fetchTrendsData(),
        fetchTopData()
      ])
    } catch (err) {
      console.error('Error refreshing data:', err)
      setError(t('dashboard.fetchError', currentLanguage) || 'Failed to fetch data')
    } finally {
      setLoading(false)
    }
  }, [currentLanguage, fetchSummaryData, fetchTrendsData, fetchTopData])

  // Initial load
  useEffect(() => {
    refreshAllData()
  }, [refreshAllData])

  // Auto refresh summary data every 5 seconds
  useEffect(() => {
    const summaryInterval = setInterval(fetchSummaryData, 5000)
    return () => clearInterval(summaryInterval)
  }, [fetchSummaryData])

  // Auto refresh trends data every 60 seconds
  useEffect(() => {
    const trendsInterval = setInterval(fetchTrendsData, 60000)
    return () => clearInterval(trendsInterval)
  }, [timeRange, fetchTrendsData])

  // Auto refresh top data every 30 seconds
  useEffect(() => {
    const topInterval = setInterval(fetchTopData, 30000)
    return () => clearInterval(topInterval)
  }, [fetchTopData])

  const handleTimeRangeChange = (value) => {
    setTimeRange(value)
    // Fetch new data for the selected time range
    fetchTrendsData()
  }

  const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042', '#8884d8']

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <Title level={2} style={{ marginBottom: 8 }}>
            <Space>
              <DashboardOutlined />
              {t('dashboard.title', currentLanguage)}
            </Space>
          </Title>
          <Text type="secondary">
            {t('dashboard.welcome', currentLanguage, { username: userInfo.username || 'Admin' })}
          </Text>
        </div>
        <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
          <Select 
            value={timeRange} 
            onChange={handleTimeRangeChange}
            style={{ width: 120 }}
          >
            <Option value="1h">1h</Option>
            <Option value="6h">6h</Option>
            <Option value="24h">24h</Option>
            <Option value="7d">7d</Option>
          </Select>
          <Button 
            onClick={refreshAllData}
            loading={loading}
          >
            {t('dashboard.refresh', currentLanguage) || 'Refresh'}
          </Button>
        </div>
      </div>

      {/* Error Message */}
      {error && (
        <Card 
          style={{ marginBottom: 24, borderColor: '#ff4d4f' }}
          bordered
        >
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', color: '#ff4d4f' }}>
            <Text strong>{error}</Text>
            <Button 
              type="primary" 
              onClick={refreshAllData}
              size="small"
            >
              {t('dashboard.retry', currentLanguage) || 'Retry'}
            </Button>
          </div>
        </Card>
      )}

      {/* Summary Cards */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} md={8} lg={6}>
          <Card 
            variant="outlined"
            icon={<GlobalOutlined />}
            hoverable
          >
            <Statistic 
              title={t('dashboard.totalQueries', currentLanguage)} 
              value={dashboardData.systemStats.totalQueries}
              precision={0}
              styles={{ content: { color: '#3f8600' } }}
              suffix={t('dashboard.queries', currentLanguage)}
            />
            <Text type="secondary" style={{ marginTop: 8, display: 'block' }}>
              {t('dashboard.qps', currentLanguage)}: {dashboardData.systemStats.qps}
            </Text>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6}>
          <Card 
            variant="outlined"
            icon={<DatabaseOutlined />}
            hoverable
          >
            <Statistic 
              title={t('dashboard.cacheHitRate', currentLanguage)} 
              value={dashboardData.systemStats.cacheHitRate}
              precision={1}
              styles={{ content: { color: '#1890ff' } }}
              suffix="%"
            />
            <Text type="secondary" style={{ marginTop: 8, display: 'block' }}>
              {t('dashboard.cacheSize', currentLanguage)}: {dashboardData.cacheStats.size}
            </Text>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6}>
          <Card 
            variant="outlined"
            icon={<AppstoreOutlined />}
            hoverable
          >
            <Statistic 
              title={t('dashboard.systemHealth', currentLanguage)} 
              value={dashboardData.systemStats.systemHealth}
              precision={0}
              styles={{ content: { color: '#52c41a' } }}
              suffix="%"
            />
            <Text type="secondary" style={{ marginTop: 8, display: 'block' }}>
              {t('dashboard.activeServers', currentLanguage)}: {dashboardData.systemStats.activeServers}
            </Text>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6}>
          <Card 
            variant="outlined"
            icon={<BarChartOutlined />}
            hoverable
          >
            <Statistic 
              title={t('dashboard.qps', currentLanguage)} 
              value={dashboardData.systemStats.qps}
              precision={1}
              styles={{ content: { color: '#fa8c16' } }}
              suffix={t('dashboard.queriesPerSecond', currentLanguage)}
            />
            <Text type="secondary" style={{ marginTop: 8, display: 'block' }}>
              {t('dashboard.timeRange', currentLanguage)}: {timeRange}
            </Text>
          </Card>
        </Col>
      </Row>

      {/* Forward Servers Status */}
      <Card title={<Space><GlobalOutlined />{t('dashboard.forwardServers', currentLanguage)}</Space>} style={{ marginBottom: 24 }}>
        <Row gutter={[16, 16]}>
          <Col xs={24} lg={12}>
            <ResponsiveContainer width="100%" height={300}>
              <AreaChart data={dashboardData.qpsTrend}>
                <defs>
                  <linearGradient id="colorQps" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#8884d8" stopOpacity={0.8}/>
                    <stop offset="95%" stopColor="#8884d8" stopOpacity={0.1}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" />
                <YAxis />
                <Tooltip />
                <Area type="monotone" dataKey="qps" stroke="#8884d8" fillOpacity={1} fill="url(#colorQps)" />
              </AreaChart>
            </ResponsiveContainer>
            <Text style={{ textAlign: 'center', display: 'block', marginTop: 8 }} type="secondary">
              {t('dashboard.qpsTrend', currentLanguage)}
            </Text>
          </Col>
          <Col xs={24} lg={12}>
            <div style={{ marginBottom: 24 }}>
              <ResponsiveContainer width="100%" height={200}>
                <BarChart data={dashboardData.latencyData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="range" />
                  <YAxis />
                  <Tooltip />
                  <Bar dataKey="count" fill="#82ca9d" />
                </BarChart>
              </ResponsiveContainer>
              <Text style={{ textAlign: 'center', display: 'block', marginTop: 8 }} type="secondary">
                {t('dashboard.latencyDistribution', currentLanguage)}
              </Text>
            </div>
            <Row gutter={[16, 16]}>
              {dashboardData.forwardServers.map((server) => (
                <Col key={server.id} xs={24} sm={12} md={8}>
                  <Card size="small" title={server.address}>
                    <Statistic 
                      title={t('dashboard.qps', currentLanguage)} 
                      value={server.qps} 
                      precision={1} 
                    />
                    <Statistic 
                      title={t('dashboard.latency', currentLanguage)} 
                      value={server.latency} 
                      precision={1} 
                      suffix="ms" 
                    />
                    <Text 
                      style={{ 
                        marginTop: 8, 
                        display: 'block',
                        color: server.status === 'healthy' ? '#52c41a' : '#ff4d4f'
                      }}
                    >
                      {server.status === 'healthy' ? t('dashboard.healthy', currentLanguage) : t('dashboard.unhealthy', currentLanguage)}
                    </Text>
                  </Card>
                </Col>
              ))}
            </Row>
          </Col>
        </Row>
      </Card>

      {/* System Resources */}
      <Card title={<Space><AppstoreOutlined />{t('dashboard.systemResources', currentLanguage)}</Space>} style={{ marginBottom: 24 }}>
        <Row gutter={[16, 16]}>
          <Col xs={24} lg={12}>
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={dashboardData.resourceUsage}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" />
                <YAxis />
                <Tooltip />
                <Legend />
                <Line type="monotone" dataKey="cpu" stroke="#ff7300" name="CPU" />
                <Line type="monotone" dataKey="memory" stroke="#3f8600" name="Memory" />
                <Line type="monotone" dataKey="disk" stroke="#13c2c2" name="Disk" />
              </LineChart>
            </ResponsiveContainer>
          </Col>
          <Col xs={24} lg={12}>
            <Row gutter={[16, 16]}>
              <Col span={24}>
                <Card size="small" title={t('dashboard.cpuUsage', currentLanguage)}>
                  <Progress 
                    percent={dashboardData.systemResources.cpu} 
                    status="active" 
                    strokeColor={{
                      from: '#108ee9',
                      to: '#87d068',
                    }}
                  />
                  <Text style={{ marginTop: 8, display: 'block' }}>
                    {dashboardData.systemResources.cpu}% {t('dashboard.used', currentLanguage)}
                  </Text>
                </Card>
              </Col>
              <Col span={24}>
                <Card size="small" title={t('dashboard.memoryUsage', currentLanguage)}>
                  <Progress 
                    percent={dashboardData.systemResources.memory} 
                    status="active" 
                    strokeColor={{
                      from: '#108ee9',
                      to: '#87d068',
                    }}
                  />
                  <Text style={{ marginTop: 8, display: 'block' }}>
                    {dashboardData.systemResources.memory}% {t('dashboard.used', currentLanguage)}
                  </Text>
                </Card>
              </Col>
              <Col span={24}>
                <Card size="small" title={t('dashboard.diskUsage', currentLanguage)}>
                  <Progress 
                    percent={dashboardData.systemResources.disk} 
                    status="active" 
                    strokeColor={{
                      from: '#108ee9',
                      to: '#87d068',
                    }}
                  />
                  <Text style={{ marginTop: 8, display: 'block' }}>
                    {dashboardData.systemResources.disk}% {t('dashboard.used', currentLanguage)}
                  </Text>
                </Card>
              </Col>
              <Col span={24}>
                <Card size="small" title={t('dashboard.networkUsage', currentLanguage)}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                    <Text>{t('dashboard.inbound', currentLanguage)}:</Text>
                    <Text strong>{dashboardData.systemResources.network.inbound}</Text>
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                    <Text>{t('dashboard.outbound', currentLanguage)}:</Text>
                    <Text strong>{dashboardData.systemResources.network.outbound}</Text>
                  </div>
                </Card>
              </Col>
            </Row>
          </Col>
        </Row>
      </Card>

      {/* Cache Status */}
      <Card title={<Space><DatabaseOutlined />{t('dashboard.cacheStatus', currentLanguage)}</Space>} style={{ marginBottom: 24 }}>
        <Row gutter={[16, 16]}>
          <Col xs={24} lg={8}>
            <Card size="small">
              <Statistic 
                title={t('dashboard.cacheSize', currentLanguage)} 
                value={dashboardData.cacheStats.size} 
              />
              <Statistic 
                title={t('dashboard.maxSize', currentLanguage)} 
                value={dashboardData.cacheStats.maxSize} 
              />
              <Statistic 
                title={t('dashboard.items', currentLanguage)} 
                value={dashboardData.cacheStats.items} 
              />
            </Card>
          </Col>
          <Col xs={24} lg={16}>
            <div style={{ marginBottom: 16 }}>
              <Text strong>{t('dashboard.cacheHitRate', currentLanguage)}: {dashboardData.cacheStats.hitRate}%</Text>
              <Progress 
                percent={dashboardData.cacheStats.hitRate} 
                status="success" 
                strokeColor="#52c41a"
                style={{ marginVertical: 8 }}
              />
            </div>
            <div>
              <Text strong>{t('dashboard.cacheMissRate', currentLanguage)}: {dashboardData.cacheStats.missRate}%</Text>
              <Progress 
                percent={dashboardData.cacheStats.missRate} 
                status="warning" 
                strokeColor="#faad14"
                style={{ marginVertical: 8 }}
              />
            </div>
            <div style={{ marginTop: 16 }}>
              <ResponsiveContainer width="100%" height={200}>
                <PieChart>
                  <Pie
                    data={[
                      { name: t('dashboard.hits', currentLanguage), value: dashboardData.cacheStats.hitRate },
                      { name: t('dashboard.misses', currentLanguage), value: dashboardData.cacheStats.missRate }
                    ]}
                    cx="50%"
                    cy="50%"
                    labelLine={false}
                    outerRadius={80}
                    fill="#8884d8"
                    dataKey="value"
                    label={({ name, percent }) => `${name}: ${(percent * 100).toFixed(0)}%`}
                  >
                    {COLORS.map((color, index) => (
                      <Cell key={`cell-${index}`} fill={color} />
                    ))}
                  </Pie>
                  <Tooltip />
                </PieChart>
              </ResponsiveContainer>
            </div>
          </Col>
        </Row>
      </Card>

      {/* Top Lists */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title={<Space><BarChartOutlined />{t('dashboard.topDomains', currentLanguage)}</Space>}>
            <Table 
              dataSource={dashboardData.topDomains}
              pagination={false}
              columns={[
                { title: 'Rank', dataIndex: 'rank', key: 'rank', width: 60 },
                { title: 'Domain', dataIndex: 'domain', key: 'domain' },
                { title: t('dashboard.queriesColumn', currentLanguage), dataIndex: 'queries', key: 'queries', width: 100 },
                { 
                  title: 'Percentage', 
                  dataIndex: 'percentage', 
                  key: 'percentage', 
                  render: (value) => `${value}%`
                }
              ]}
              rowKey="rank"
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title={<Space><UserOutlined />{t('dashboard.topClients', currentLanguage)}</Space>}>
            <Table 
              dataSource={dashboardData.topClients}
              pagination={false}
              columns={[
                { title: 'Rank', dataIndex: 'rank', key: 'rank', width: 60 },
                { title: 'IP Address', dataIndex: 'ip', key: 'ip' },
                { title: t('dashboard.queriesColumn', currentLanguage), dataIndex: 'queries', key: 'queries', width: 100 },
                { 
                  title: 'Percentage', 
                  dataIndex: 'percentage', 
                  key: 'percentage', 
                  width: 120,
                  render: (value) => `${value}%`
                }
              ]}
              rowKey="rank"
            />
          </Card>
        </Col>
      </Row>
    </div>
  )
}

export default Dashboard