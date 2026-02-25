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
      currentQps: 0,
      avgQps: 0,
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
    resourceUsage: [],
    networkUsage: []
  })

  // Get summary data from API
  const fetchSummaryData = useCallback(async () => {
    try {
      setError(null)
      
      // Fetch dashboard summary data
      const response = await apiClient.getDashboardSummary()
      if (response.success) {
        setDashboardData(prev => ({
          ...prev,
          systemStats: response.data.systemStats,
          forwardServers: response.data.forwardServers,
          cacheStats: response.data.cacheStats,
          systemResources: response.data.systemResources
        }))
      }
      
      // Still fetch server status for additional info
      const serverStatusResponse = await apiClient.getServerStatus()
      if (serverStatusResponse.success) {
        setDashboardData(prev => ({
          ...prev,
          serverStatus: serverStatusResponse.data
        }))
      }
      
    } catch (error) {
      console.error('Error fetching summary data:', error)
      setError(t('dashboard.fetchError', currentLanguage) || 'Failed to fetch data')
    }
  }, [currentLanguage])

  // 网络流量单位转换函数
  const formatNetworkSpeed = (bytesPerSecond) => {
    if (bytesPerSecond < 1024) {
      return `${bytesPerSecond} B/s`
    } else if (bytesPerSecond < 1024 * 1024) {
      return `${(bytesPerSecond / 1024).toFixed(1)} KB/s`
    } else {
      return `${(bytesPerSecond / (1024 * 1024)).toFixed(1)} MB/s`
    }
  }

  // Get trends data from API
  const fetchTrendsData = useCallback(async () => {
    try {
      // Fetch dashboard trends data
      const response = await apiClient.getDashboardTrends('all', timeRange, 12)
      if (response.success) {
        // 处理数据格式转换
        let qpsTrend = []
        let resourceUsage = []
        let networkUsage = []
        
        // 检查返回的数据格式
        if (response.data.qpsTrend && response.data.qpsTrend.timeLabels) {
          // 处理带 statistics 的格式
          qpsTrend = response.data.qpsTrend.timeLabels.map((time, index) => ({
            time,
            qps: response.data.qpsTrend.qpsValues[index]
          }))
          
          if (response.data.resourceUsage && response.data.resourceUsage.timeLabels) {
            resourceUsage = response.data.resourceUsage.timeLabels.map((time, index) => ({
              time,
              cpu: response.data.resourceUsage.cpuValues[index],
              memory: response.data.resourceUsage.memValues[index],
              disk: response.data.resourceUsage.diskValues[index]
            }))
          }
          
          if (response.data.networkUsage && response.data.networkUsage.timeLabels) {
            networkUsage = response.data.networkUsage.timeLabels.map((time, index) => ({
              time,
              inbound: response.data.networkUsage.inboundValues[index],
              outbound: response.data.networkUsage.outboundValues[index]
            }))
          }
        } else {
          // 处理原始数据格式
          qpsTrend = response.data.qpsTrend || []
          resourceUsage = response.data.resourceUsage || []
          networkUsage = response.data.networkUsage || []
        }
        
        // 固定延迟区间顺序
        const fixedLatencyRanges = ['<10ms', '10-20ms', '20-50ms', '50-100ms', '>100ms']
        const latencyDataMap = {}
        
        // 将返回的数据转换为映射
        const latencyDataArray = Array.isArray(response.data.latencyData) ? response.data.latencyData : []
        latencyDataArray.forEach(item => {
          // 处理可能的HTML实体编码
          const range = item.range.replace(/&lt;/g, '<').replace(/&gt;/g, '>')
          latencyDataMap[range] = item.count
        })
        
        // 根据固定顺序重组数据
        const orderedLatencyData = fixedLatencyRanges.map(range => ({
          range,
          count: latencyDataMap[range] || 0
        }))
        
        setDashboardData(prev => ({
          ...prev,
          qpsTrend,
          latencyData: orderedLatencyData,
          resourceUsage,
          networkUsage
        }))
      }
    } catch (error) {
      console.error('Error fetching trends data:', error)
    }
  }, [timeRange])

  // Get top data from API
  const fetchTopData = useCallback(async () => {
    try {
      // Fetch dashboard top data
      const response = await apiClient.getDashboardTop(10)
      if (response.success) {
        setDashboardData(prev => ({
          ...prev,
          topDomains: response.data.topDomains || [],
          topClients: response.data.topClients || []
        }))
      }
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
          variant="outlined"
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
            style={{ height: '100%' }}
          >
            <Statistic 
              title={t('dashboard.totalQueries', currentLanguage)} 
              value={dashboardData.systemStats.totalQueries}
              precision={0}
              styles={{ content: { color: '#3f8600' } }}
              suffix={t('dashboard.queries', currentLanguage)}
            />
            <Text type="secondary" style={{ marginTop: 8, display: 'block' }}>
              {t('dashboard.successfulQueries', currentLanguage)}: {dashboardData.serverStatus?.dns_server?.stats?.successfulRequests || 0}
            </Text>
            <Text type="secondary" style={{ marginTop: 4, display: 'block' }}>
              {t('dashboard.failedQueries', currentLanguage)}: {dashboardData.serverStatus?.dns_server?.stats?.failedRequests || 0}
            </Text>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6}>
          <Card 
            variant="outlined"
            icon={<DatabaseOutlined />}
            hoverable
            style={{ height: '100%' }}
          >
            <Statistic 
              title={t('dashboard.cacheHitRate', currentLanguage)} 
              value={dashboardData.cacheStats.hitRate}
              precision={0}
              styles={{ content: { color: '#1890ff' } }}
              suffix="%"
            />
            <Text type="secondary" style={{ marginTop: 4, display: 'block' }}>
              {t('dashboard.cacheSize', currentLanguage)}: {dashboardData.cacheStats.size}
            </Text>
            <Text type="secondary" style={{ marginTop: 4, display: 'block' }}>
              {t('dashboard.cacheItems', currentLanguage)}: {dashboardData.cacheStats.items}
            </Text>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6}>
          <Card 
            variant="outlined"
            icon={<AppstoreOutlined />}
            hoverable
            style={{ height: '100%' }}
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
            <div style={{ height: 20 }}></div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6}>
          <Card 
            variant="outlined"
            icon={<BarChartOutlined />}
            hoverable
            style={{ height: '100%' }}
          >
            <Statistic 
              title={t('dashboard.currentQps', currentLanguage)} 
              value={dashboardData.systemStats.currentQps}
              precision={2}
              styles={{ content: { color: '#fa8c16' } }}
              suffix={t('dashboard.queriesPerSecond', currentLanguage)}
            />
            <Text type="secondary" style={{ marginTop: 4, display: 'block' }}>
              {t('dashboard.peakQps', currentLanguage)}: {dashboardData.systemStats.qps.toFixed(2)}
            </Text>
            <Text type="secondary" style={{ marginTop: 4, display: 'block' }}>
              {t('dashboard.avgQps', currentLanguage)}: {dashboardData.systemStats.avgQps.toFixed(2)}
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
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={dashboardData.latencyData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="range" />
                  <YAxis tickFormatter={(value) => `${value}%`} />
                  <Tooltip 
                    formatter={(value) => [`${value}%`, t('dashboard.percentage', currentLanguage)]}
                    labelFormatter={(label) => `${t('dashboard.latencyRange', currentLanguage)}: ${label}`}
                  />
                  <Bar dataKey="count" fill="#82ca9d" />
                </BarChart>
              </ResponsiveContainer>
              <Text style={{ textAlign: 'center', display: 'block', marginTop: 8 }} type="secondary">
                {t('dashboard.latencyDistribution', currentLanguage)}
              </Text>
            </div>
          </Col>
        </Row>
        
        {/* Forward Servers Cards */}
        <Row gutter={[16, 16]} justify="start" style={{ marginTop: 24 }}>
          {dashboardData.forwardServers.map((server) => (
            <Col key={server.id} xs={24} sm={12} md={4} lg={4} xl={4}>
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
                <Tooltip formatter={(value, name) => [`${value}%`, name]} />
                <Legend />
                <Line type="monotone" dataKey="cpu" stroke="#ff7300" name="CPU" />
                <Line type="monotone" dataKey="memory" stroke="#3f8600" name="Memory" />
                <Line type="monotone" dataKey="disk" stroke="#13c2c2" name="Disk" />
              </LineChart>
            </ResponsiveContainer>
          </Col>
          <Col xs={24} lg={12}>
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={dashboardData.networkUsage}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" />
                <YAxis tickFormatter={(value) => `${Math.round(value / 1024)} KB/s`} />
                <Tooltip formatter={(value, name) => [formatNetworkSpeed(value), name === 'inbound' ? t('dashboard.inbound', currentLanguage) : t('dashboard.outbound', currentLanguage)]} />
                <Legend />
                <Line type="monotone" dataKey="inbound" stroke="#1890ff" name={t('dashboard.inbound', currentLanguage)} />
                <Line type="monotone" dataKey="outbound" stroke="#52c41a" name={t('dashboard.outbound', currentLanguage)} />
              </LineChart>
            </ResponsiveContainer>
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