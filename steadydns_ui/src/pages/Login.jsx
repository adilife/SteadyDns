/*
 * SteadyDNS UI
 * Copyright (C) 2024 SteadyDNS Team
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
import { Card, Form, Input, Button, message, Typography, Select, Space } from 'antd'
import { UserOutlined, LockOutlined, GlobalOutlined, CloudServerOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'

import { storeTokens } from '../utils/tokenManager'
import { apiClient } from '../utils/apiClient'

const { Title, Text } = Typography
const { Option } = Select

/**
 * 登录页面组件
 * @param {Object} props - 组件属性
 * @param {Function} props.onLogin - 登录成功回调函数
 * @returns {JSX.Element} 登录页面组件
 */
const Login = ({ onLogin }) => {
  const { t, i18n } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [isMobile, setIsMobile] = useState(false)

  /**
   * 检测屏幕尺寸变化，判断是否为移动端
   */
  useEffect(() => {
    const handleResize = () => {
      setIsMobile(window.innerWidth < 480)
    }
    
    handleResize()
    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [])

  /**
   * 处理登录表单提交
   * @param {Object} values - 表单值
   */
  const handleSubmit = async (values) => {
    setLoading(true)
    try {
      const response = await apiClient.login(values.username, values.password)
      
      if (response.success) {
        storeTokens(response.data.access_token, response.data.refresh_token, response.data.expires_in)
        
        message.success(response.message || t('login.success'))
        onLogin({
          user: response.data.user,
          message: response.message
        })
      } else {
        console.error('Login failed:', response.message)
        message.error(response.message || t('login.error'))
      }
    } catch (error) {
      console.error('Login error:', error)
    } finally {
      setLoading(false)
    }
  }

  /**
   * 处理语言切换
   * @param {string} lang - 目标语言
   */
  const handleLanguageChange = (lang) => {
    i18n.changeLanguage(lang)
  }

  const isRTL = i18n.language === 'ar-SA'

  // 更新HTML根元素的dir和lang属性，用于全局CSS选择器
  useEffect(() => {
    document.documentElement.dir = isRTL ? 'rtl' : 'ltr'
    document.documentElement.lang = i18n.language
  }, [isRTL, i18n.language])

  const containerStyle = {
    minHeight: '100vh',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    background: 'linear-gradient(135deg, #e8f4fc 0%, #f0f2f5 50%, #f5f7fa 100%)',
    padding: isMobile ? '16px' : '24px',
    direction: isRTL ? 'rtl' : 'ltr'
  }

  const cardStyle = {
    width: '100%',
    maxWidth: isMobile ? '100%' : '420px',
    borderRadius: isMobile ? '8px' : '12px',
    boxShadow: '0 8px 32px rgba(0, 0, 0, 0.08), 0 2px 8px rgba(0, 0, 0, 0.04)',
    border: '1px solid rgba(255, 255, 255, 0.8)',
    overflow: 'hidden'
  }

  const headerStyle = {
    textAlign: 'center',
    padding: isMobile ? '24px 20px 16px' : '32px 32px 24px',
    background: 'linear-gradient(180deg, #fafbfc 0%, #ffffff 100%)',
    borderBottom: '1px solid #f0f0f0'
  }

  const logoStyle = {
    width: '56px',
    height: '56px',
    borderRadius: '12px',
    background: 'linear-gradient(135deg, #1890ff 0%, #096dd9 100%)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    margin: '0 auto 16px',
    boxShadow: '0 4px 12px rgba(24, 144, 255, 0.3)'
  }

  const formContainerStyle = {
    padding: isMobile ? '20px' : '32px'
  }

  const languageSelectorStyle = {
    display: 'flex',
    alignItems: 'center',
    justifyContent: isMobile ? 'center' : (isRTL ? 'flex-start' : 'flex-end'),
    marginBottom: isMobile ? '16px' : '0',
    marginTop: isMobile ? '0' : '-8px'
  }

  const inputStyle = {
    transition: 'all 0.3s ease'
  }

  return (
    <div style={containerStyle}>
      <Card style={cardStyle} styles={{ body: { padding: 0 } }}>
        <div style={headerStyle}>
          <div style={logoStyle}>
            <CloudServerOutlined style={{ fontSize: '28px', color: '#fff' }} />
          </div>
          <Title level={3} style={{ margin: 0, fontWeight: 600, color: '#1a1a1a' }}>
            SteadyDNS
          </Title>
          <Text type="secondary" style={{ fontSize: '14px', display: 'block', marginTop: '4px' }}>
            {t('login.title')}
          </Text>
        </div>
        
        <div style={formContainerStyle}>
          <div style={languageSelectorStyle}>
            <Space size={8}>
              <GlobalOutlined style={{ color: '#8c8c8c', fontSize: '14px' }} />
              <Select
                value={i18n.language}
                style={{ width: 110 }}
                onChange={handleLanguageChange}
                size="small"
                popupMatchSelectWidth={false}
              >
                <Option value="zh-CN">中文</Option>
                <Option value="en-US">English</Option>
                <Option value="ar-SA">العربية</Option>
              </Select>
            </Space>
          </div>

          <Form
            name="login"
            onFinish={handleSubmit}
            layout="vertical"
            style={{ marginTop: isMobile ? '12px' : '16px' }}
          >
            <Form.Item
              name="username"
              label={t('login.username')}
              rules={[{ required: true, message: t('login.username') }]}
            >
              <Input 
                prefix={<UserOutlined style={{ color: '#bfbfbf' }} />} 
                placeholder={t('login.username')}
                style={inputStyle}
                size="large"
              />
            </Form.Item>
            
            <Form.Item
              name="password"
              label={t('login.password')}
              rules={[{ required: true, message: t('login.password') }]}
            >
              <Input.Password 
                prefix={<LockOutlined style={{ color: '#bfbfbf' }} />} 
                placeholder={t('login.password')}
                style={inputStyle}
                size="large"
              />
            </Form.Item>
            
            <Form.Item style={{ marginBottom: 0, marginTop: '24px' }}>
              <Button 
                type="primary" 
                htmlType="submit" 
                style={{ 
                  width: '100%',
                  height: '44px',
                  fontSize: '15px',
                  fontWeight: 500,
                  borderRadius: '6px',
                  boxShadow: '0 2px 8px rgba(24, 144, 255, 0.2)'
                }}
                loading={loading}
              >
                {t('login.login')}
              </Button>
            </Form.Item>
          </Form>
        </div>
      </Card>
    </div>
  )
}

export default Login
