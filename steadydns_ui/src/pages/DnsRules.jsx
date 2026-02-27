import { useState } from 'react'
import {
  Table,
  Button,
  Modal,
  Form,
  Input,
  Select,
  message,
  Space,
  Popconfirm,
  Tooltip
} from 'antd'
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SaveOutlined
} from '@ant-design/icons'
import { t } from '../i18n'

const { Option } = Select

// Mock data for DNS rules
const mockRules = [
  {
    id: 1,
    domain: 'example.com',
    type: 'A',
    value: '192.168.1.1',
    priority: 1,
    description: 'Example domain'
  },
  {
    id: 2,
    domain: 'test.com',
    type: 'CNAME',
    value: 'example.com',
    priority: 2,
    description: 'Test domain'
  },
  {
    id: 3,
    domain: 'api.example.com',
    type: 'A',
    value: '192.168.1.2',
    priority: 1,
    description: 'API server'
  }
]

const DnsRules = ({ currentLanguage }) => {
  const [rules, setRules] = useState(mockRules)
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editingRule, setEditingRule] = useState(null)
  const [form] = Form.useForm()

  const showModal = (rule = null) => {
    setEditingRule(rule)
    if (rule) {
      form.setFieldsValue(rule)
    } else {
      form.resetFields()
    }
    setIsModalOpen(true)
  }

  const handleCancel = () => {
    setIsModalOpen(false)
    setEditingRule(null)
    form.resetFields()
  }

  const handleOk = () => {
    form.validateFields().then(values => {
      if (editingRule) {
        // Update existing rule
        setRules(rules.map(rule => rule.id === editingRule.id ? { ...rule, ...values } : rule))
        message.success(t('dnsRules.ruleUpdated', currentLanguage))
      } else {
        // Add new rule
        const newRule = {
          id: rules.length + 1,
          ...values
        }
        setRules([...rules, newRule])
        message.success(t('dnsRules.ruleAdded', currentLanguage))
      }
      setIsModalOpen(false)
      setEditingRule(null)
      form.resetFields()
    }).catch(() => {
      message.error(currentLanguage === 'zh-CN' ? '请检查表单字段' : 'Please check form fields')
    })
  }
  const handleDelete = (id) => {
    setRules(rules.filter(rule => rule.id !== id))
    message.success(t('dnsRules.ruleDeleted', currentLanguage))
  }
  const columns = [
    {
      title: t('dnsRules.id', currentLanguage),
      dataIndex: 'id',
      key: 'id',
      width: 60
    },
    {
      title: t('dnsRules.domain', currentLanguage),
      dataIndex: 'domain',
      key: 'domain',
      ellipsis: true
    },
    {
      title: t('dnsRules.type', currentLanguage),
      dataIndex: 'type',
      key: 'type',
      width: 100
    },
    {
      title: t('dnsRules.value', currentLanguage),
      dataIndex: 'value',
      key: 'value',
      ellipsis: true
    },
    {
      title: t('dnsRules.priority', currentLanguage),
      dataIndex: 'priority',
      key: 'priority',
      width: 100
    },
    {
      title: t('dnsRules.description', currentLanguage),
      dataIndex: 'description',
      key: 'description',
      ellipsis: true
    },
    {
      title: t('dnsRules.actions', currentLanguage),
      key: 'actions',
      width: 150,
      render: (_, record) => (
        <Space size="middle">
          <Tooltip title={t('dnsRules.edit', currentLanguage)}>
            <Button
              icon={<EditOutlined />}
              size="small"
              onClick={() => showModal(record)}
            />
          </Tooltip>
          <Tooltip title={t('dnsRules.delete', currentLanguage)}>
            <Popconfirm
              title={t('dnsRules.confirmDelete', currentLanguage)}
              onConfirm={() => handleDelete(record.id)}
              okText={t('dnsRules.yes', currentLanguage)}
              cancelText={t('dnsRules.no', currentLanguage)}
            >
              <Button
                icon={<DeleteOutlined />}
                size="small"
                danger
              />
            </Popconfirm>
          </Tooltip>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2>{t('dnsRules.title', currentLanguage)}</h2>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => showModal()}
        >
          {t('dnsRules.addRule', currentLanguage)}
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={rules}
        rowKey="id"
        pagination={{
          showSizeChanger: true,
          pageSizeOptions: ['10', '20', '50'],
          defaultPageSize: 10
        }}
        scroll={{ x: 'max-content' }}
      />

      <Modal
        title={editingRule ? t('dnsRules.editRule', currentLanguage) : t('dnsRules.addNewRule', currentLanguage)}
        open={isModalOpen}
        onOk={handleOk}
        onCancel={handleCancel}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            type: 'A',
            priority: 1
          }}
        >
          <Form.Item
            name="domain"
            label={t('dnsRules.domain', currentLanguage)}
            rules={[{ required: true, message: currentLanguage === 'zh-CN' ? '请输入域名' : 'Please input domain' }]}
          >
            <Input placeholder={currentLanguage === 'zh-CN' ? '请输入域名' : 'Enter domain name'} />
          </Form.Item>

          <Form.Item
            name="type"
            label={t('dnsRules.type', currentLanguage)}
            rules={[{ required: true, message: currentLanguage === 'zh-CN' ? '请选择类型' : 'Please select type' }]}
          >
            <Select placeholder={currentLanguage === 'zh-CN' ? '选择DNS类型' : 'Select DNS type'}>
              <Option value="A">A</Option>
              <Option value="CNAME">CNAME</Option>
              <Option value="MX">MX</Option>
              <Option value="TXT">TXT</Option>
              <Option value="AAAA">AAAA</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="value"
            label={t('dnsRules.value', currentLanguage)}
            rules={[{ required: true, message: currentLanguage === 'zh-CN' ? '请输入值' : 'Please input value' }]}
          >
            <Input placeholder={currentLanguage === 'zh-CN' ? '请输入DNS值' : 'Enter DNS value'} />
          </Form.Item>

          <Form.Item
            name="priority"
            label={t('dnsRules.priority', currentLanguage)}
            rules={[{ required: true, message: currentLanguage === 'zh-CN' ? '请输入优先级' : 'Please input priority' }]}
          >
            <Input type="number" min={1} max={10} placeholder={currentLanguage === 'zh-CN' ? '请输入优先级' : 'Enter priority'} />
          </Form.Item>

          <Form.Item
            name="description"
            label={t('dnsRules.description', currentLanguage)}
          >
            <Input.TextArea rows={3} placeholder={currentLanguage === 'zh-CN' ? '请输入描述' : 'Enter description'} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default DnsRules