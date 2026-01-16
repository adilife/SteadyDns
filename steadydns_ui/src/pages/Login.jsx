import { useState } from 'react'
import { Card, Form, Input, Button, message, Typography, Select } from 'antd'
import { UserOutlined, LockOutlined } from '@ant-design/icons'

import { storeTokens } from '../utils/tokenManager'
import { apiClient } from '../utils/apiClient'

const { Title } = Typography
const { Option } = Select

const Login = ({ onLogin, currentLanguage, onLanguageChange }) => {
  const [loading, setLoading] = useState(false)
  const handleSubmit = async (values) => {
    setLoading(true)
    try {
      const response = await apiClient.login(values.username, values.password)
      
      if (response.success) {
        // Store tokens using token manager
        storeTokens(response.data.access_token, response.data.refresh_token, response.data.expires_in)
        
        message.success(response.message || (currentLanguage === 'zh-CN' ? '登录成功！' : 'Login successful!'))
        onLogin({
          user: response.data.user,
          message: response.message
        })
      } else {
        console.error('Login failed:', response.message)
        message.error(response.message || (currentLanguage === 'zh-CN' ? '登录失败，请检查用户名和密码' : 'Login failed, please check username and password'))
      }
    } catch (error) {
      console.error('Login error:', error)
      // Error messages are already handled by apiClient
    } finally {
      setLoading(false)
    }
  }
  const loginText = {
    'zh-CN': {
      title: 'SteadyDNS 管理系统',
      username: '用户名',
      password: '密码',
      login: '登录',
      welcome: '欢迎登录',
      chinese: '中文',
      english: '英文',
      language: '语言'
    },
    'en-US': {
      title: 'SteadyDNS Management System',
      username: 'Username',
      password: 'Password',
      login: 'Login',
      welcome: 'Welcome',
      chinese: 'Chinese',
      english: 'English',
      language: 'Language'
    }
  }

  const text = loginText[currentLanguage]

  return (
    <div style={{ 
      minHeight: '100vh', 
      display: 'flex', 
      alignItems: 'center', 
      justifyContent: 'center', 
      background: '#f0f2f5',
      padding: '24px'
    }}>
      <Card 
        style={{ 
          width: 400, 
          maxWidth: '100%',
          borderRadius: '8px',
          boxShadow: '0 4px 12px rgba(0, 0, 0, 0.15)'
        }}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
          <div style={{ textAlign: 'center' }}>
            <Title level={3}>{text.title}</Title>
            <div style={{ fontSize: '16px', color: '#666', marginTop: '8px' }}>
              {text.welcome}
            </div>
          </div>          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <span style={{ fontSize: '14px', color: '#666' }}>{text.language}:</span>
            <Select
              value={currentLanguage}
              style={{ width: 120 }}
              onChange={onLanguageChange}
            >
              <Option value="zh-CN">{text.chinese}</Option>
              <Option value="en-US">{text.english}</Option>
            </Select>
          </div>        </div>
        
        <Form
          name="login"
          onFinish={handleSubmit}
          layout="vertical"
        >
          <Form.Item
            name="username"
            label={text.username}
            rules={[{ required: true, message: currentLanguage === 'zh-CN' ? '请输入用户名' : 'Please enter username' }]}
          >
            <Input prefix={<UserOutlined />} placeholder={currentLanguage === 'zh-CN' ? '请输入用户名' : 'Please enter username'} />
          </Form.Item>
          
          <Form.Item
            name="password"
            label={text.password}
            rules={[{ required: true, message: currentLanguage === 'zh-CN' ? '请输入密码' : 'Please enter password' }]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder={currentLanguage === 'zh-CN' ? '请输入密码' : 'Please enter password'} />
          </Form.Item>
          
          <Form.Item>
            <Button 
              type="primary" 
              htmlType="submit" 
              style={{ width: '100%' }}
              loading={loading}
            >
              {text.login}
            </Button>
          </Form.Item>
        </Form>
        
      </Card>
    </div>
  )
}

export default Login