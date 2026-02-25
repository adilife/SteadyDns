/**
 * SteadyDNS UI
 * Copyright (C) 2026 SteadyDNS Team
 * 
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published
 * by the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 * 
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 * 
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

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
import { logout, hasValidToken, startTokenRefreshInterval, resetSessionTimeoutTimer, startSessionTimeoutTimer } from './utils/tokenManager'
import { apiClient } from './utils/apiClient'

const { Header, Sider, Content } = Layout
const { Option } = Select

function App() {
  const [selectedKey, setSelectedKey] = useState('1')
  const [collapsed, setCollapsed] = useState(false)
  const [isLoggedIn, setIsLoggedIn] = useState(false)
  const [currentLanguage, setCurrentLanguage] = useState(getSavedLanguage())
  const [userInfo, setUserInfo] = useState({ username: '' })
  const [pluginsStatus, setPluginsStatus] = useState({})
  // Check if user is logged in from sessionStorage
  useEffect(() => {
    const checkLoginStatus = () => {
      const savedUser = sessionStorage.getItem('steadyDNS_user')
      const hasToken = hasValidToken()
      
      if (savedUser && hasToken) {
        try {
          if (savedUser !== 'undefined') {
            const parsedUser = JSON.parse(savedUser)
            if (parsedUser && Object.keys(parsedUser).length > 0) {
              setIsLoggedIn(true)
              setUserInfo(parsedUser)
              // Start token refresh interval only once
              const tokenRefreshStarted = sessionStorage.getItem('steadyDNS_token_refresh_started')
              if (tokenRefreshStarted !== 'true') {
                startTokenRefreshInterval()
                sessionStorage.setItem('steadyDNS_token_refresh_started', 'true')
              }
              
              // Start session timeout timer
              // Note: startSessionTimeoutTimer will check if already running internally
              const expiresInStr = sessionStorage.getItem('steadyDNS_token_expires_in')
              const expiresIn = expiresInStr ? parseInt(expiresInStr) : 1800
              startSessionTimeoutTimer(expiresIn)
              
              // Get plugins status when logged in
              fetchPluginsStatus()
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
    
    // Fetch plugins status
    const fetchPluginsStatus = async () => {
      try {
        const response = await apiClient.getPluginsStatus()
        if (response.success) {
          // Convert plugins array to object for easier access
          const pluginsMap = {}
          response.data.plugins.forEach(plugin => {
            pluginsMap[plugin.name] = plugin
          })
          setPluginsStatus(pluginsMap)
        } else {
          console.error('Failed to get plugins status:', response.error)
        }
      } catch (error) {
        console.error('Error fetching plugins status:', error)
      }
    }
    
    checkLoginStatus()
    // Check login status periodically
    const interval = setInterval(checkLoginStatus, 30000)
    
    // User activity event listeners to reset session timeout
    const handleUserActivity = () => {
      resetSessionTimeoutTimer()
    }
    
    // Session storage change listener to detect token changes
    const handleStorageChange = (event) => {
      // Check if any token-related keys or user info have changed
      const tokenRelatedKeys = [
        'steadyDNS_access_token',
        'steadyDNS_refresh_token', 
        'steadyDNS_token_expires_at',
        'steadyDNS_user'
      ]
      
      if (tokenRelatedKeys.includes(event.key)) {
        checkLoginStatus()
      }
    }
    
    window.addEventListener('mousedown', handleUserActivity)
    window.addEventListener('keydown', handleUserActivity)
    window.addEventListener('storage', handleStorageChange)
    
    return () => {
      clearInterval(interval)
      window.removeEventListener('mousedown', handleUserActivity)
      window.removeEventListener('keydown', handleUserActivity)
      window.removeEventListener('storage', handleStorageChange)
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
    setIsLoggedIn(true)
    // Handle both cases - capitalized (Go default) and lowercase (common in JSON)
    const user = loginData.User || loginData.user
    setUserInfo(user || {})
    if (user) {
      try {
        sessionStorage.setItem('steadyDNS_user', JSON.stringify(user))
      } catch (error) {
        console.error('Error saving user data to sessionStorage:', error)
      }
    } else {
      console.error('Invalid login data:', { user })
    }
    
    // Start session timeout timer after login
    // Use expires_in from login data, default to 1800 seconds (30 minutes)
    const expiresIn = loginData.expires_in || loginData.ExpiresIn || 1800
    startSessionTimeoutTimer(expiresIn)
  }

  const handleLogout = async () => {
    try {
      await logout()
      setIsLoggedIn(false)
      setUserInfo({ username: '' })
      sessionStorage.removeItem('steadyDNS_user')
      // Clear the token refresh started flag so it can be started again on next login
      sessionStorage.removeItem('steadyDNS_token_refresh_started')
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
        // Check if BIND plugin is enabled before rendering AuthZones
        if (!pluginsStatus.bind?.enabled) {
          return (
            <div style={{ textAlign: 'center', padding: '40px' }}>
              <h2 style={{ color: '#ff4d4f', marginBottom: '16px' }}>BIND插件未启用</h2>
              <p style={{ marginBottom: '24px' }}>请启用BIND插件后再访问权威域管理功能</p>
              <p>插件启用/禁用通过配置文件控制，修改后需重启服务生效</p>
              <p>配置文件位置: /src/cmd/config/steadydns.conf</p>
            </div>
          )
        }
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
            // Only show auth zones menu if BIND plugin is enabled
            ...(pluginsStatus.bind?.enabled ? [{
              key: '2',
              icon: <DatabaseOutlined />,
              label: t('nav.authZones', currentLanguage),
            }] : []),
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
