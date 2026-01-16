import { useState, useEffect } from 'react'
import {
  Table,
  Input,
  Select,
  Button,
  Space,
  DatePicker,
  message,
  Tooltip
} from 'antd'
import {
  ReloadOutlined,
  DownloadOutlined,
  SearchOutlined
} from '@ant-design/icons'
import { t } from '../i18n'

const { Option } = Select
const { RangePicker } = DatePicker

// Mock data for DNS logs
const generateMockLogs = () => {
  const logs = []
  const domains = ['example.com', 'test.com', 'api.example.com', 'google.com', 'baidu.com']
  const types = ['A', 'CNAME', 'MX', 'TXT', 'AAAA']
  const results = ['SUCCESS', 'FAILED', 'TIMEOUT']
  const clients = ['192.168.1.100', '192.168.1.101', '192.168.1.102', '10.0.0.1', '10.0.0.2']

  for (let i = 1; i <= 100; i++) {
    const timestamp = new Date()
    timestamp.setMinutes(timestamp.getMinutes() - Math.floor(Math.random() * 1440)) // Random time in last 24 hours

    logs.push({
      id: i,
      timestamp: timestamp.toISOString(),
      client: clients[Math.floor(Math.random() * clients.length)],
      domain: domains[Math.floor(Math.random() * domains.length)],
      type: types[Math.floor(Math.random() * types.length)],
      result: results[Math.floor(Math.random() * results.length)],
      response: results[Math.floor(Math.random() * results.length)] === 'SUCCESS' 
        ? `192.168.1.${Math.floor(Math.random() * 255)}` 
        : 'N/A',
      latency: `${Math.floor(Math.random() * 100)}ms`
    })
  }

  return logs.sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp))
}

const Logs = ({ currentLanguage, userInfo }) => {
  const [logs, setLogs] = useState([])
  const [filteredLogs, setFilteredLogs] = useState([])
  const [loading, setLoading] = useState(false)
  const [searchText, setSearchText] = useState('')
  const [resultFilter, setResultFilter] = useState('')
  const [dateRange, setDateRange] = useState(null)

  useEffect(() => {
    loadLogs()
  }, [])

  const loadLogs = () => {
    setLoading(true)
    // Simulate API call
    setTimeout(() => {
      const mockLogs = generateMockLogs()
      setLogs(mockLogs)
      setFilteredLogs(mockLogs)
      setLoading(false)
    }, 500)
  }

  const handleSearch = () => {
    let filtered = logs

    // Filter by search text
    if (searchText) {
      filtered = filtered.filter(log => 
        log.domain.includes(searchText) || 
        log.client.includes(searchText) ||
        log.response.includes(searchText)
      )
    }

    // Filter by result
    if (resultFilter) {
      filtered = filtered.filter(log => log.result === resultFilter)
    }

    // Filter by date range
    if (dateRange && dateRange.length === 2) {
      const start = dateRange[0].startOf('day').toISOString()
      const end = dateRange[1].endOf('day').toISOString()
      filtered = filtered.filter(log => log.timestamp >= start && log.timestamp <= end)
    }

    setFilteredLogs(filtered)
  }

  const handleReset = () => {
    setSearchText('')
    setResultFilter('')
    setDateRange(null)
    setFilteredLogs(logs)
  }

  const handleDownload = () => {
    // Simulate download
    message.success(t('logs.logsDownloaded', currentLanguage))
  }

  const formatTimestamp = (timestamp) => {
    return new Date(timestamp).toLocaleString()
  }

  const columns = [
    {
      title: t('logs.id', currentLanguage),
      dataIndex: 'id',
      key: 'id',
      width: 60
    },
    {
      title: t('logs.timestamp', currentLanguage),
      dataIndex: 'timestamp',
      key: 'timestamp',
      render: (text) => formatTimestamp(text),
      sorter: (a, b) => new Date(b.timestamp) - new Date(a.timestamp),
      defaultSortOrder: 'descend'
    },
    {
      title: t('logs.clientIP', currentLanguage),
      dataIndex: 'client',
      key: 'client',
      width: 120
    },
    {
      title: t('logs.domain', currentLanguage),
      dataIndex: 'domain',
      key: 'domain',
      ellipsis: true
    },
    {
      title: t('logs.type', currentLanguage),
      dataIndex: 'type',
      key: 'type',
      width: 80
    },
    {
      title: t('logs.result', currentLanguage),
      dataIndex: 'result',
      key: 'result',
      width: 100,
      render: (text) => {
        let color = ''
        switch (text) {
          case 'SUCCESS':
            color = 'green'
            break
          case 'FAILED':
            color = 'red'
            break
          case 'TIMEOUT':
            color = 'orange'
            break
          default:
            color = 'default'
        }
        return <span style={{ color }}>{text}</span>
      }
    },
    {
      title: t('logs.response', currentLanguage),
      dataIndex: 'response',
      key: 'response',
      ellipsis: true
    },
    {
      title: t('logs.latency', currentLanguage),
      dataIndex: 'latency',
      key: 'latency',
      width: 100
    }
  ]

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <h2>{t('logs.title', currentLanguage)}</h2>
      </div>

      <div style={{ marginBottom: 16, padding: 16, background: '#f5f5f5', borderRadius: 8 }}>
        <Space orientation="vertical" style={{ width: '100%' }}>
          <Space wrap>
            <Input
              placeholder={t('logs.searchPlaceholder', currentLanguage)}
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              style={{ width: 300 }}
              prefix={<SearchOutlined />}
            />
            <Select
              placeholder={t('logs.filterByResult', currentLanguage)}
              value={resultFilter}
              onChange={setResultFilter}
              style={{ width: 150 }}
              allowClear
            >
              <Option value="SUCCESS">{t('logs.success', currentLanguage)}</Option>
              <Option value="FAILED">{t('logs.failed', currentLanguage)}</Option>
              <Option value="TIMEOUT">{t('logs.timeout', currentLanguage)}</Option>
            </Select>
            <RangePicker
              value={dateRange}
              onChange={setDateRange}
              style={{ width: 300 }}
            />
            <Button
              type="primary"
              onClick={handleSearch}
              icon={<SearchOutlined />}
            >
              {t('logs.search', currentLanguage)}
            </Button>
            <Button onClick={handleReset}>{t('logs.reset', currentLanguage)}</Button>
            <Button
              onClick={loadLogs}
              icon={<ReloadOutlined />}
              loading={loading}
            >
              {t('logs.refresh', currentLanguage)}
            </Button>
            <Button
              icon={<DownloadOutlined />}
              onClick={handleDownload}
            >
              {t('logs.download', currentLanguage)}
            </Button>
          </Space>
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={filteredLogs}
        rowKey="id"
        pagination={{
          showSizeChanger: true,
          pageSizeOptions: ['10', '20', '50', '100'],
          defaultPageSize: 20
        }}
        scroll={{ x: 'max-content' }}
        loading={loading}
      />
    </div>
  )
}

export default Logs