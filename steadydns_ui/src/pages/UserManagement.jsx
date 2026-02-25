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

import { useState, useEffect, useCallback } from 'react'
import {
  Table,
  Button,
  Modal,
  Form,
  Input,
  Pagination,
  message,
  Space,
  Popconfirm,
  Tooltip,
  Spin
} from 'antd'
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  KeyOutlined
} from '@ant-design/icons'
import { t } from '../i18n'
import { apiClient } from '../utils/apiClient'

/**
 * 用户管理页面组件
 * 提供用户的增删改查和密码修改功能
 * @param {Object} props - 组件属性
 * @param {string} props.currentLanguage - 当前语言
 */
const UserManagement = ({ currentLanguage }) => {
  // 用户列表数据
  const [users, setUsers] = useState([])
  // 分页信息
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0
  })
  // 加载状态
  const [loading, setLoading] = useState(false)
  // 创建用户弹窗状态
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
  // 编辑用户弹窗状态
  const [isEditModalOpen, setIsEditModalOpen] = useState(false)
  // 修改密码弹窗状态
  const [isPasswordModalOpen, setIsPasswordModalOpen] = useState(false)
  // 当前编辑的用户
  const [editingUser, setEditingUser] = useState(null)
  // 创建用户表单
  const [createForm] = Form.useForm()
  // 编辑用户表单
  const [editForm] = Form.useForm()
  // 修改密码表单
  const [passwordForm] = Form.useForm()

  /**
   * 加载用户列表
   * @param {number} page - 页码
   * @param {number} pageSize - 每页数量
   */
  const loadUsers = useCallback(async (page = 1, pageSize = 10) => {
    setLoading(true)
    try {
      const response = await apiClient.getUsers(page, pageSize)
      
      if (response.success) {
        setUsers(response.data.users || [])
        setPagination({
          current: response.data.page || page,
          pageSize: response.data.pageSize || pageSize,
          total: response.data.total || 0
        })
      } else {
        message.error(response.message || t('userManagement.fetchError', currentLanguage))
      }
    } catch (error) {
      console.error('Error loading users:', error)
      message.error(t('userManagement.fetchError', currentLanguage))
    } finally {
      setLoading(false)
    }
  }, [currentLanguage])

  /**
   * 组件挂载时加载用户列表
   */
  useEffect(() => {
    loadUsers()
  }, [loadUsers])

  /**
   * 显示创建用户弹窗
   */
  const showCreateModal = () => {
    createForm.resetFields()
    setIsCreateModalOpen(true)
  }

  /**
   * 取消创建用户弹窗
   */
  const handleCreateCancel = () => {
    setIsCreateModalOpen(false)
    createForm.resetFields()
  }

  /**
   * 提交创建用户表单
   */
  const handleCreateOk = () => {
    createForm.validateFields().then(async (values) => {
      setLoading(true)
      try {
        const response = await apiClient.createUser(values)
        
        if (response.success) {
          message.success(response.message || t('userManagement.userCreated', currentLanguage))
          setIsCreateModalOpen(false)
          createForm.resetFields()
          loadUsers(pagination.current, pagination.pageSize)
        } else {
          message.error(response.message || t('userManagement.createError', currentLanguage))
        }
      } catch (error) {
        console.error('Error creating user:', error)
        message.error(t('userManagement.createError', currentLanguage))
      } finally {
        setLoading(false)
      }
    }).catch(() => {
      message.error(t('userManagement.createError', currentLanguage))
    })
  }

  /**
   * 显示编辑用户弹窗
   * @param {Object} user - 用户对象
   */
  const showEditModal = (user) => {
    setEditingUser(user)
    editForm.setFieldsValue({
      username: user.username,
      email: user.email
    })
    setIsEditModalOpen(true)
  }

  /**
   * 取消编辑用户弹窗
   */
  const handleEditCancel = () => {
    setIsEditModalOpen(false)
    setEditingUser(null)
    editForm.resetFields()
  }

  /**
   * 提交编辑用户表单
   */
  const handleEditOk = () => {
    editForm.validateFields().then(async (values) => {
      setLoading(true)
      try {
        const response = await apiClient.updateUser(editingUser.id, values)
        
        if (response.success) {
          message.success(response.message || t('userManagement.userUpdated', currentLanguage))
          setIsEditModalOpen(false)
          setEditingUser(null)
          editForm.resetFields()
          loadUsers(pagination.current, pagination.pageSize)
        } else {
          message.error(response.message || t('userManagement.updateError', currentLanguage))
        }
      } catch (error) {
        console.error('Error updating user:', error)
        message.error(t('userManagement.updateError', currentLanguage))
      } finally {
        setLoading(false)
      }
    }).catch(() => {
      message.error(t('userManagement.updateError', currentLanguage))
    })
  }

  /**
   * 删除用户
   * @param {Object} user - 用户对象
   */
  const handleDelete = async (user) => {
    // 防止删除admin用户
    if (user.username === 'admin') {
      message.warning(t('userManagement.confirmDeleteAdmin', currentLanguage))
      return
    }
    
    setLoading(true)
    try {
      const response = await apiClient.deleteUser(user.id)
      
      if (response.success) {
        message.success(response.message || t('userManagement.userDeleted', currentLanguage))
        loadUsers(pagination.current, pagination.pageSize)
      } else {
        message.error(response.message || t('userManagement.deleteError', currentLanguage))
      }
    } catch (error) {
      console.error('Error deleting user:', error)
      message.error(t('userManagement.deleteError', currentLanguage))
    } finally {
      setLoading(false)
    }
  }

  /**
   * 显示修改密码弹窗
   * @param {Object} user - 用户对象
   */
  const showPasswordModal = (user) => {
    setEditingUser(user)
    passwordForm.resetFields()
    setIsPasswordModalOpen(true)
  }

  /**
   * 取消修改密码弹窗
   */
  const handlePasswordCancel = () => {
    setIsPasswordModalOpen(false)
    setEditingUser(null)
    passwordForm.resetFields()
  }

  /**
   * 提交修改密码表单
   */
  const handlePasswordOk = () => {
    passwordForm.validateFields().then(async (values) => {
      setLoading(true)
      try {
        const passwordData = {
          old_password: values.oldPassword,
          new_password: values.newPassword
        }
        const response = await apiClient.changePassword(editingUser.id, passwordData)
        
        if (response.success) {
          message.success(response.message || t('userManagement.passwordChanged', currentLanguage))
          setIsPasswordModalOpen(false)
          setEditingUser(null)
          passwordForm.resetFields()
        } else {
          message.error(response.message || t('userManagement.passwordError', currentLanguage))
        }
      } catch (error) {
        console.error('Error changing password:', error)
        message.error(t('userManagement.passwordError', currentLanguage))
      } finally {
        setLoading(false)
      }
    }).catch(() => {
      message.error(t('userManagement.passwordError', currentLanguage))
    })
  }

  /**
   * 分页变化处理
   * @param {number} page - 页码
   * @param {number} pageSize - 每页数量
   */
  const handlePaginationChange = (page, pageSize) => {
    loadUsers(page, pageSize)
  }

  /**
   * 表格列定义
   */
  const columns = [
    {
      title: t('userManagement.id', currentLanguage),
      dataIndex: 'id',
      key: 'id',
      width: 80
    },
    {
      title: t('userManagement.username', currentLanguage),
      dataIndex: 'username',
      key: 'username',
      ellipsis: true
    },
    {
      title: t('userManagement.email', currentLanguage),
      dataIndex: 'email',
      key: 'email',
      ellipsis: true
    },
    {
      title: t('userManagement.actions', currentLanguage),
      key: 'actions',
      width: 200,
      render: (_, record) => (
        <Space size="small">
          <Tooltip title={t('userManagement.edit', currentLanguage)}>
            <Button
              type="link"
              icon={<EditOutlined />}
              size="small"
              onClick={() => showEditModal(record)}
            />
          </Tooltip>
          <Tooltip title={t('userManagement.changePassword', currentLanguage)}>
            <Button
              type="link"
              icon={<KeyOutlined />}
              size="small"
              onClick={() => showPasswordModal(record)}
            />
          </Tooltip>
          <Popconfirm
            title={t('userManagement.confirmDelete', currentLanguage)}
            onConfirm={() => handleDelete(record)}
            okText={t('userManagement.confirm', currentLanguage)}
            cancelText={t('userManagement.cancel', currentLanguage)}
            disabled={record.username === 'admin'}
          >
            <Tooltip title={record.username === 'admin' ? t('userManagement.confirmDeleteAdmin', currentLanguage) : t('userManagement.delete', currentLanguage)}>
              <Button
                type="link"
                icon={<DeleteOutlined />}
                size="small"
                danger
                disabled={record.username === 'admin'}
              />
            </Tooltip>
          </Popconfirm>
        </Space>
      )
    }
  ]

  return (
    <div>
      {/* 标题和操作按钮区域 */}
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2>{t('userManagement.title', currentLanguage)}</h2>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={showCreateModal}
        >
          {t('userManagement.addUser', currentLanguage)}
        </Button>
      </div>

      {/* 用户列表表格 */}
      <Spin spinning={loading}>
        <Table
          columns={columns}
          dataSource={users}
          rowKey="id"
          pagination={false}
          scroll={{ x: 'max-content' }}
        />
        
        {/* 分页组件 */}
        <div style={{ marginTop: 16, display: 'flex', justifyContent: 'flex-end' }}>
          <Pagination
            current={pagination.current}
            pageSize={pagination.pageSize}
            total={pagination.total}
            onChange={handlePaginationChange}
            showSizeChanger
            showQuickJumper
            showTotal={(total) => 
              currentLanguage === 'zh-CN' 
                ? `共 ${total} 条记录` 
                : `Total ${total} records`
            }
            pageSizeOptions={['10', '20', '50', '100']}
          />
        </div>
      </Spin>

      {/* 创建用户弹窗 */}
      <Modal
        title={t('userManagement.addUser', currentLanguage)}
        open={isCreateModalOpen}
        onOk={handleCreateOk}
        onCancel={handleCreateCancel}
        okText={t('userManagement.confirm', currentLanguage)}
        cancelText={t('userManagement.cancel', currentLanguage)}
        confirmLoading={loading}
      >
        <Form
          form={createForm}
          layout="vertical"
        >
          <Form.Item
            name="username"
            label={t('userManagement.username', currentLanguage)}
            rules={[
              { required: true, message: t('userManagement.pleaseInputUsername', currentLanguage) },
              { min: 3, message: t('userManagement.usernameMinLength', currentLanguage) }
            ]}
          >
            <Input placeholder={t('userManagement.pleaseInputUsername', currentLanguage)} />
          </Form.Item>
          
          <Form.Item
            name="email"
            label={t('userManagement.email', currentLanguage)}
            rules={[
              { required: false },
              { type: 'email', message: t('userManagement.invalidEmail', currentLanguage) }
            ]}
          >
            <Input placeholder={t('userManagement.pleaseInputEmail', currentLanguage)} />
          </Form.Item>
          
          <Form.Item
            name="password"
            label={t('userManagement.password', currentLanguage)}
            rules={[
              { required: true, message: t('userManagement.pleaseInputPassword', currentLanguage) },
              { min: 6, message: t('userManagement.passwordMinLength', currentLanguage) }
            ]}
          >
            <Input.Password placeholder={t('userManagement.pleaseInputPassword', currentLanguage)} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 编辑用户弹窗 */}
      <Modal
        title={t('userManagement.editUser', currentLanguage)}
        open={isEditModalOpen}
        onOk={handleEditOk}
        onCancel={handleEditCancel}
        okText={t('userManagement.confirm', currentLanguage)}
        cancelText={t('userManagement.cancel', currentLanguage)}
        confirmLoading={loading}
      >
        <Form
          form={editForm}
          layout="vertical"
        >
          <Form.Item
            name="username"
            label={t('userManagement.username', currentLanguage)}
            rules={[
              { required: true, message: t('userManagement.pleaseInputUsername', currentLanguage) },
              { min: 3, message: t('userManagement.usernameMinLength', currentLanguage) }
            ]}
          >
            <Input placeholder={t('userManagement.pleaseInputUsername', currentLanguage)} />
          </Form.Item>
          
          <Form.Item
            name="email"
            label={t('userManagement.email', currentLanguage)}
            rules={[
              { required: false },
              { type: 'email', message: t('userManagement.invalidEmail', currentLanguage) }
            ]}
          >
            <Input placeholder={t('userManagement.pleaseInputEmail', currentLanguage)} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 修改密码弹窗 */}
      <Modal
        title={t('userManagement.changePassword', currentLanguage)}
        open={isPasswordModalOpen}
        onOk={handlePasswordOk}
        onCancel={handlePasswordCancel}
        okText={t('userManagement.confirm', currentLanguage)}
        cancelText={t('userManagement.cancel', currentLanguage)}
        confirmLoading={loading}
      >
        <Form
          form={passwordForm}
          layout="vertical"
        >
          <Form.Item
            name="oldPassword"
            label={t('userManagement.oldPassword', currentLanguage)}
            rules={[
              { required: true, message: t('userManagement.pleaseInputOldPassword', currentLanguage) }
            ]}
          >
            <Input.Password placeholder={t('userManagement.pleaseInputOldPassword', currentLanguage)} />
          </Form.Item>
          
          <Form.Item
            name="newPassword"
            label={t('userManagement.newPassword', currentLanguage)}
            rules={[
              { required: true, message: t('userManagement.pleaseInputNewPassword', currentLanguage) },
              { min: 6, message: t('userManagement.passwordMinLength', currentLanguage) }
            ]}
          >
            <Input.Password placeholder={t('userManagement.pleaseInputNewPassword', currentLanguage)} />
          </Form.Item>
          
          <Form.Item
            name="confirmPassword"
            label={t('userManagement.confirmPassword', currentLanguage)}
            dependencies={['newPassword']}
            rules={[
              { required: true, message: t('userManagement.pleaseConfirmPassword', currentLanguage) },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('newPassword') === value) {
                    return Promise.resolve()
                  }
                  return Promise.reject(new Error(t('userManagement.passwordNotMatch', currentLanguage)))
                }
              })
            ]}
          >
            <Input.Password placeholder={t('userManagement.pleaseConfirmPassword', currentLanguage)} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default UserManagement
