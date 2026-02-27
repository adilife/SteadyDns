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

import { useState, useEffect, useRef, useMemo, useCallback } from 'react'
import { Layout, Menu, theme, Select, Space, Avatar, Dropdown, message, Tooltip } from 'antd'
import {
  SettingOutlined,
  LogoutOutlined,
  AppstoreOutlined,
  UserOutlined,
  DownOutlined,
  DashboardOutlined,
  DatabaseOutlined,
  TeamOutlined,
  InfoCircleOutlined,
  ForwardOutlined
} from '@ant-design/icons'
import './App.css'
import DnsRules from './pages/DnsRules'
import Logs from './pages/Logs'
import ForwardGroups from './pages/ForwardGroups'
import Dashboard from './pages/Dashboard'
import AuthZones from './pages/AuthZones'
import Configuration from './pages/Configuration'
import UserManagement from './pages/UserManagement'
import Login from './pages/Login'
import AboutModal from './components/AboutModal'
import { t, getSavedLanguage, switchLanguage } from './i18n'
import { logout, hasValidToken, startTokenRefreshInterval, resetSessionTimeoutTimer, startSessionTimeoutTimer } from './utils/tokenManager'
import { apiClient } from './utils/apiClient'
import { VERSION } from './config/version'

const { Header, Sider, Content } = Layout
const { Option } = Select

// MenuLabel 组件，支持文本溢出检测和 tooltip 提示
const MenuLabel = ({ text }) => {
  const [isOverflowing, setIsOverflowing] = useState(false)
  const labelRef = useRef(null)

  useEffect(() => {
    if (labelRef.current) {
      setIsOverflowing(labelRef.current.scrollWidth > labelRef.current.clientWidth)
    }
  }, [text])

  return (
    <Tooltip title={isOverflowing ? text : ''}>
      <div 
        ref={labelRef}
        style={{
          whiteSpace: 'nowrap',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          maxWidth: '100%'
        }}
      >
        {text}
      </div>
    </Tooltip>
  )
}

function App() {
  const [selectedKey, setSelectedKey] = useState('1')
  const [collapsed, setCollapsed] = useState(false)
  const [isLoggedIn, setIsLoggedIn] = useState(false)
  const [currentLanguage, setCurrentLanguage] = useState(getSavedLanguage())
  const [userInfo, setUserInfo] = useState({ username: '' })
  const [pluginsStatus, setPluginsStatus] = useState({})
  const [isMobile, setIsMobile] = useState(false)
  const [aboutModalOpen, setAboutModalOpen] = useState(false)

  // 检测是否为 RTL 语言
  const isRTL = currentLanguage === 'ar-SA' // 阿拉伯语等 RTL 语言

  // 检测屏幕尺寸，判断是否为移动端
  useEffect(() => {
    const handleResize = () => {
      setIsMobile(window.innerWidth < 768)
    }
    
    handleResize() // 初始检测
    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [])

  // 计算文本宽度的辅助函数
  const calculateTextWidth = (text) => {
    const canvas = document.createElement('canvas')
    const ctx = canvas.getContext('2d')
    ctx.font = '14px -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif'
    return ctx.measureText(text).width
  }

  // 计算侧边栏宽度
  const siderWidth = useMemo(() => {
    // 获取所有菜单文本
    const menuTexts = [
      t('nav.dashboard', currentLanguage),
      pluginsStatus.bind?.enabled ? t('nav.authZones', currentLanguage) : '',
      pluginsStatus['dns-rules']?.enabled ? t('nav.dnsRules', currentLanguage) : '',
      pluginsStatus['log-analysis']?.enabled ? t('nav.logs', currentLanguage) : '',
      t('nav.forwardGroups', currentLanguage),
      t('configuration.title', currentLanguage),
      t('userManagement.title', currentLanguage)
    ]
    
    // 计算最长文本宽度，加上图标和padding
    const maxTextWidth = Math.max(...menuTexts.map(calculateTextWidth))
    return Math.min(Math.max(180, maxTextWidth + 80), 320) // 限制在180-320px之间
  }, [currentLanguage, pluginsStatus])

  // Fetch plugins status with cache support (defined at component level)
  const fetchPluginsStatus = useCallback(async (useCache = true) => {
    // Try to load from cache first
    if (useCache) {
      const cachedPlugins = localStorage.getItem('steadyDNS_plugins_status')
      if (cachedPlugins) {
        try {
          const parsedPlugins = JSON.parse(cachedPlugins)
          setPluginsStatus(parsedPlugins)
        } catch (e) {
          console.error('Error parsing cached plugins status:', e)
        }
      }
    }
    
    // Fetch latest from API
    try {
      const response = await apiClient.getPluginsStatus()
      if (response.success) {
        // Convert plugins array to object for easier access
        const pluginsMap = {}
        response.data.plugins.forEach(plugin => {
          pluginsMap[plugin.name] = plugin
        })
        setPluginsStatus(pluginsMap)
        // Cache the plugins status
        localStorage.setItem('steadyDNS_plugins_status', JSON.stringify(pluginsMap))
      } else {
        console.error('Failed to get plugins status:', response.error)
      }
    } catch (error) {
      console.error('Error fetching plugins status:', error)
    }
  }, [])

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
  }, [fetchPluginsStatus])
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
    
    // Fetch plugins status after login (use cache if available)
    fetchPluginsStatus(true)
  }

  const handleLogout = async () => {
    try {
      await logout()
      setIsLoggedIn(false)
      setUserInfo({ username: '' })
      setPluginsStatus({})
      sessionStorage.removeItem('steadyDNS_user')
      localStorage.removeItem('steadyDNS_plugins_status')
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
        // Check if DNS Rules plugin is enabled before rendering DnsRules
        if (!pluginsStatus['dns-rules']?.enabled) {
          return (
            <div style={{ textAlign: 'center', padding: '40px' }}>
              <h2 style={{ color: '#ff4d4f', marginBottom: '16px' }}>DNS Rules 插件未启用</h2>
              <p style={{ marginBottom: '24px' }}>请启用 DNS Rules 插件后再访问 DNS 规则管理功能</p>
              <p>插件启用/禁用通过配置文件控制，修改后需重启服务生效</p>
              <p>配置文件位置: /src/cmd/config/steadydns.conf</p>
            </div>
          )
        }
        return <DnsRules currentLanguage={currentLanguage} userInfo={userInfo} />
      case '4':
        // Check if Log Analysis plugin is enabled before rendering Logs
        if (!pluginsStatus['log-analysis']?.enabled) {
          return (
            <div style={{ textAlign: 'center', padding: '40px' }}>
              <h2 style={{ color: '#ff4d4f', marginBottom: '16px' }}>Log Analysis 插件未启用</h2>
              <p style={{ marginBottom: '24px' }}>请启用 Log Analysis 插件后再访问日志分析功能</p>
              <p>插件启用/禁用通过配置文件控制，修改后需重启服务生效</p>
              <p>配置文件位置: /src/cmd/config/steadydns.conf</p>
            </div>
          )
        }
        return <Logs currentLanguage={currentLanguage} userInfo={userInfo} />
      case '5':
        return <ForwardGroups currentLanguage={currentLanguage} userInfo={userInfo} />
      case '6':
        return <Configuration currentLanguage={currentLanguage} userInfo={userInfo} />
      case '7':
        return <UserManagement currentLanguage={currentLanguage} />
      default:
        return <Dashboard currentLanguage={currentLanguage} userInfo={userInfo} />
    }
  }

  // User dropdown menu
  const userMenu = [
    {
      key: 'about',
      label: (
        <a onClick={() => setAboutModalOpen(true)}>
          <InfoCircleOutlined style={{ marginRight: '8px' }} />
          {t('about.title', currentLanguage)}
        </a>
      ),
    },
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

  // 移动端布局
  if (isMobile) {
    return (
      <Layout style={{ minHeight: '100vh', direction: isRTL ? 'rtl' : 'ltr' }}>
        <Header style={{ display: 'flex', alignItems: 'center', justifyContent: isRTL ? 'space-between' : 'space-between', padding: '0 16px', background: colorBgContainer, direction: isRTL ? 'rtl' : 'ltr' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px', direction: isRTL ? 'rtl' : 'ltr' }}>
            <h1 style={{ margin: 0, fontSize: '14px', fontWeight: 'bold', textAlign: isRTL ? 'right' : 'left' }}>
              {t('header.title', currentLanguage)}
            </h1>
          </div>
          <Space size="middle">
            {/* Language selector */}
            <Select
              value={currentLanguage}
              style={{ width: 100 }}
              onChange={handleLanguageChange}
            >
              <Option value="zh-CN">中文</Option>
              <Option value="en-US">English</Option>
              <Option value="ar-SA">العربية</Option>
            </Select>
            
            {/* User info and logout */}
            <Dropdown menu={{ items: userMenu }}>
              <Space style={{ cursor: 'pointer' }}>
                <Avatar icon={<UserOutlined />} />
                <DownOutlined />
              </Space>
            </Dropdown>
          </Space>
        </Header>
        <Header style={{ background: '#f0f0f0', padding: 0, direction: isRTL ? 'rtl' : 'ltr' }}>
          <Menu
            theme="light"
            mode="horizontal"
            selectedKeys={[selectedKey]}
            onSelect={({ key }) => setSelectedKey(key)}
            style={{ lineHeight: '64px', direction: isRTL ? 'rtl' : 'ltr' }}
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
              // Only show DNS Rules menu if DNS Rules plugin is enabled
              ...(pluginsStatus['dns-rules']?.enabled ? [{
                key: '3',
                icon: <AppstoreOutlined />,
                label: t('nav.dnsRules', currentLanguage),
              }] : []),
              // Only show Logs menu if Log Analysis plugin is enabled
              ...(pluginsStatus['log-analysis']?.enabled ? [{
                key: '4',
                icon: <LogoutOutlined />,
                label: t('nav.logs', currentLanguage),
              }] : []),
              {
                key: '5',
                icon: <ForwardOutlined />,
                label: t('nav.forwardGroups', currentLanguage),
              },
              {
                key: '6',
                icon: <SettingOutlined />,
                label: t('configuration.title', currentLanguage),
              },
              {
                key: '7',
                icon: <TeamOutlined />,
                label: t('userManagement.title', currentLanguage),
              },
            ]}
          />
        </Header>
        <Content
          style={{
            margin: '16px',
            padding: 16,
            background: colorBgContainer,
            borderRadius: borderRadiusLG,
            overflow: 'auto',
            direction: isRTL ? 'rtl' : 'ltr',
            textAlign: isRTL ? 'right' : 'left'
          }}
        >
          {renderContent()}
        </Content>
        
        {/* 关于弹窗 */}
        <AboutModal
          open={aboutModalOpen}
          onCancel={() => setAboutModalOpen(false)}
          currentLanguage={currentLanguage}
        />
      </Layout>
    )
  }

  // 桌面端布局
  return (
    <Layout style={{ minHeight: '100vh', height: '100vh', direction: isRTL ? 'rtl' : 'ltr' }}>
      <Sider
        collapsible
        collapsed={collapsed}
        onCollapse={(value) => setCollapsed(value)}
        style={{
          position: 'relative',
          overflow: 'auto',
          width: siderWidth,
          minWidth: 180,
          maxWidth: 320,
          direction: isRTL ? 'rtl' : 'ltr',
          textAlign: isRTL ? 'right' : 'left'
        }}
      >
        <div className="logo" />
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selectedKey]}
          onSelect={({ key }) => setSelectedKey(key)}
          style={{ direction: isRTL ? 'rtl' : 'ltr' }}
          items={[
            {
              key: '1',
              icon: <DashboardOutlined />,
              label: <MenuLabel text={t('nav.dashboard', currentLanguage)} />,
            },
            // Only show auth zones menu if BIND plugin is enabled
            ...(pluginsStatus.bind?.enabled ? [{
              key: '2',
              icon: <DatabaseOutlined />,
              label: <MenuLabel text={t('nav.authZones', currentLanguage)} />,
            }] : []),
            // Only show DNS Rules menu if DNS Rules plugin is enabled
            ...(pluginsStatus['dns-rules']?.enabled ? [{
              key: '3',
              icon: <AppstoreOutlined />,
              label: <MenuLabel text={t('nav.dnsRules', currentLanguage)} />,
            }] : []),
            // Only show Logs menu if Log Analysis plugin is enabled
            ...(pluginsStatus['log-analysis']?.enabled ? [{
              key: '4',
              icon: <LogoutOutlined />,
              label: <MenuLabel text={t('nav.logs', currentLanguage)} />,
            }] : []),
            {
              key: '5',
              icon: <ForwardOutlined />,
              label: <MenuLabel text={t('nav.forwardGroups', currentLanguage)} />,
            },
            {
              key: '6',
              icon: <SettingOutlined />,
              label: <MenuLabel text={t('configuration.title', currentLanguage)} />,
            },
            {
              key: '7',
              icon: <TeamOutlined />,
              label: <MenuLabel text={t('userManagement.title', currentLanguage)} />,
            },
          ]}
        />
        {/* 侧边栏底部版本号 */}
        <div style={{
          position: 'absolute',
          bottom: '48px',
          left: 0,
          right: 0,
          padding: collapsed ? '8px' : '8px 24px',
          textAlign: 'center',
          color: 'rgba(255, 255, 255, 0.45)',
          fontSize: '12px',
          borderTop: '1px solid rgba(255, 255, 255, 0.1)',
          direction: isRTL ? 'rtl' : 'ltr',
          backgroundColor: 'rgba(0, 21, 41, 1)'
        }}>
          {collapsed ? VERSION.substring(0, 4) : VERSION}
        </div>
      </Sider>
      <Layout style={{ display: 'flex', flexDirection: 'column', height: '100%', direction: isRTL ? 'rtl' : 'ltr' }}>
        <Header style={{ display: 'flex', alignItems: 'center', justifyContent: isRTL ? 'space-between' : 'space-between', padding: '0 24px', background: colorBgContainer, direction: isRTL ? 'rtl' : 'ltr' }}>
          <h1 style={{ margin: 0, fontSize: '16px', fontWeight: 'bold', textAlign: isRTL ? 'right' : 'left' }}>
            {t('header.title', currentLanguage)}
          </h1>
          <Space size="large">
            {/* Language selector */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', direction: isRTL ? 'rtl' : 'ltr' }}>
              <span style={{ fontSize: '14px', color: '#666', textAlign: isRTL ? 'right' : 'left' }}>{t('header.language', currentLanguage)}:</span>
              <Select
                value={currentLanguage}
                style={{ width: 120 }}
                onChange={handleLanguageChange}
              >
                <Option value="zh-CN">中文</Option>
                <Option value="en-US">English</Option>
                <Option value="ar-SA">العربية</Option>
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
            direction: isRTL ? 'rtl' : 'ltr',
            textAlign: isRTL ? 'right' : 'left'
          }}
        >
          {renderContent()}
        </Content>
      </Layout>
      
      {/* 关于弹窗 */}
      <AboutModal
        open={aboutModalOpen}
        onCancel={() => setAboutModalOpen(false)}
        currentLanguage={currentLanguage}
      />
    </Layout>
  )
}

export default App
