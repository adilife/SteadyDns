import { useState, useEffect } from 'react'
import { Layout, Menu, theme, Select, Space, Avatar, Dropdown, message } from 'antd'
import {
  SettingOutlined,
  LogoutOutlined,
  AppstoreOutlined,
  UserOutlined,
  DownOutlined,
  DashboardOutlined,
  DatabaseOutlined
} from '@ant-design/icons'
import './App.css'
import DnsRules from './pages/DnsRules'
import Logs from './pages/Logs'
import ForwardGroups from './pages/ForwardGroups'
import Dashboard from './pages/Dashboard'
import AuthZones from './pages/AuthZones'
import Configuration from './pages/Configuration'
import Login from './pages/Login'
import { t, getSavedLanguage, switchLanguage } from './i18n'
import { logout, hasValidToken, startTokenRefreshInterval, resetSessionTimeoutTimer, clearTokenRefreshInterval } from './utils/tokenManager'

const { Header, Sider, Content } = Layout
const { Option } = Select

function App() {
  const [selectedKey, setSelectedKey] = useState('1')
  const [collapsed, setCollapsed] = useState(false)
  const [isLoggedIn, setIsLoggedIn] = useState(false)
  const [currentLanguage, setCurrentLanguage] = useState(getSavedLanguage())
  const [userInfo, setUserInfo] = useState({ username: '' })
  // Check if user is logged in from sessionStorage
  useEffect(() => {
    const checkLoginStatus = () => {
      const savedUser = sessionStorage.getItem('steadyDNS_user')
      if (savedUser && hasValidToken()) {
        try {
          if (savedUser !== 'undefined') {
            const parsedUser = JSON.parse(savedUser)
            if (parsedUser && Object.keys(parsedUser).length > 0) {
              setIsLoggedIn(true)
              setUserInfo(parsedUser)
              // Start token refresh interval
              startTokenRefreshInterval()
            } else {
              console.error('Invalid user data in sessionStorage:', parsedUser)
              sessionStorage.removeItem('steadyDNS_user')
              setIsLoggedIn(false)
              setUserInfo({ username: '' })
            }
          } else {
            console.error('Invalid data in sessionStorage')
            sessionStorage.removeItem('steadyDNS_user')
            setIsLoggedIn(false)
            setUserInfo({ username: '' })
          }
        } catch (error) {
          console.error('Error parsing user data from sessionStorage:', error)
          sessionStorage.removeItem('steadyDNS_user')
          setIsLoggedIn(false)
          setUserInfo({ username: '' })
        }
      } else {
        setIsLoggedIn(false)
        setUserInfo({ username: '' })
      }
    }
    
    checkLoginStatus()
    // Check login status periodically
    const interval = setInterval(checkLoginStatus, 30000)
    
    // User activity event listeners to reset session timeout
    const handleUserActivity = () => {
      resetSessionTimeoutTimer()
    }
    
    window.addEventListener('mousedown', handleUserActivity)
    window.addEventListener('keydown', handleUserActivity)
    window.addEventListener('scroll', handleUserActivity)
    
    return () => {
      clearInterval(interval)
      window.removeEventListener('mousedown', handleUserActivity)
      window.removeEventListener('keydown', handleUserActivity)
      window.removeEventListener('scroll', handleUserActivity)
    }
  }, [])
  // Update language when changed
  useEffect(() => {
    switchLanguage(currentLanguage)
  }, [currentLanguage])

  const {
    token: { colorBgContainer, borderRadiusLG },
  } = theme.useToken()
  const handleLogin = (loginData) => {
    console.log('Login data received:', loginData)
    setIsLoggedIn(true)
    // Handle both cases - capitalized (Go default) and lowercase (common in JSON)
    const user = loginData.User || loginData.user
    const message = loginData.Message || loginData.message
    console.log('Extracted user:', user)
    console.log('Extracted message:', message)
    setUserInfo(user || {})
    if (user) {
      try {
        sessionStorage.setItem('steadyDNS_user', JSON.stringify(user))
        console.log('User data saved to sessionStorage:', user)
      } catch (error) {
        console.error('Error saving user data to sessionStorage:', error)
      }
    } else {
      console.error('Invalid login data:', { user })
    }
  }

  const handleLogout = async () => {
    try {
      await logout()
      setIsLoggedIn(false)
      setUserInfo({ username: '' })
      sessionStorage.removeItem('steadyDNS_user')
      message.success(t('login.logoutSuccess'))
    } catch (error) {
      console.error('Logout error:', error)
      // Even if logout API fails, clear local data
      setIsLoggedIn(false)
      setUserInfo({ username: '' })
      sessionStorage.removeItem('steadyDNS_user')
      message.success(t('login.logoutSuccess'))
    }
  }
  const handleLanguageChange = (lang) => {
    setCurrentLanguage(lang)
  }

  const renderContent = () => {
    switch (selectedKey) {
      case '1':
        return <Dashboard currentLanguage={currentLanguage} userInfo={userInfo} />
      case '2':
        return <AuthZones currentLanguage={currentLanguage} userInfo={userInfo} />
      case '3':
        return <DnsRules currentLanguage={currentLanguage} userInfo={userInfo} />
      case '4':
        return <Logs currentLanguage={currentLanguage} userInfo={userInfo} />
      case '5':
        return <ForwardGroups currentLanguage={currentLanguage} userInfo={userInfo} />
      case '6':
        return <Configuration currentLanguage={currentLanguage} userInfo={userInfo} />
      default:
        return <Dashboard currentLanguage={currentLanguage} userInfo={userInfo} />
    }
  }

  // User dropdown menu
  const userMenu = [
    {
      key: 'logout',
      label: (
        <a onClick={handleLogout}>
          {t('header.logout', currentLanguage)}
        </a>
      ),
    },
  ]

  if (!isLoggedIn) {
    return <Login 
      onLogin={handleLogin} 
      currentLanguage={currentLanguage} 
      onLanguageChange={handleLanguageChange} 
    />
  }

  return (
    <Layout style={{ minHeight: '100vh', height: '100vh' }}>
      <Sider
        collapsible
        collapsed={collapsed}
        onCollapse={(value) => setCollapsed(value)}
        style={{
          overflow: 'auto',
        }}
      >
        <div className="logo" />
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selectedKey]}
          onSelect={({ key }) => setSelectedKey(key)}
          items={[
            {
              key: '1',
              icon: <DashboardOutlined />,
              label: t('nav.dashboard', currentLanguage),
            },
            {
              key: '2',
              icon: <DatabaseOutlined />,
              label: t('nav.authZones', currentLanguage),
            },
            {
              key: '3',
              icon: <AppstoreOutlined />,
              label: t('nav.dnsRules', currentLanguage),
            },
            {
              key: '4',
              icon: <LogoutOutlined />,
              label: t('nav.logs', currentLanguage),
            },
            {
              key: '5',
              icon: <SettingOutlined />,
              label: t('nav.forwardGroups', currentLanguage),
            },
            {
              key: '6',
              icon: <SettingOutlined />,
              label: t('configuration.title', currentLanguage),
            },
          ]}
        />
      </Sider>
      <Layout style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
        <Header style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '0 24px', background: colorBgContainer }}>
          <h1 style={{ margin: 0, fontSize: '16px', fontWeight: 'bold' }}>
            {t('header.title', currentLanguage)}
          </h1>
          <Space size="large">
            {/* Language selector */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <span style={{ fontSize: '14px', color: '#666' }}>{t('header.language', currentLanguage)}:</span>
              <Select
                value={currentLanguage}
                style={{ width: 120 }}
                onChange={handleLanguageChange}
              >
                <Option value="zh-CN">{t('header.chinese', currentLanguage)}</Option>
                <Option value="en-US">{t('header.english', currentLanguage)}</Option>
              </Select>
            </div>
            
            {/* User info and logout */}
            <Dropdown menu={{ items: userMenu }}>
              <Space style={{ cursor: 'pointer' }}>
                <Avatar icon={<UserOutlined />} />
                <span>{t('header.welcome', currentLanguage, { username: userInfo.username || '' })}</span>
                <DownOutlined />
              </Space>
            </Dropdown>
          </Space>
        </Header>
        <Content
          style={{
            margin: '24px 16px',
            padding: 24,
            flex: 1,
            minHeight: 0,
            background: colorBgContainer,
            borderRadius: borderRadiusLG,
            overflow: 'auto',
          }}
        >
          {renderContent()}
        </Content>
      </Layout>
    </Layout>
  )
}

export default App
