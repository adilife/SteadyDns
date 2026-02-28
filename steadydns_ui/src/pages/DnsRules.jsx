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
import { useTranslation } from 'react-i18next'

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

const DnsRules = () => {
  const { t } = useTranslation()
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
        message.success(t('dnsRules.ruleUpdated'))
      } else {
        // Add new rule
        const newRule = {
          id: rules.length + 1,
          ...values
        }
        setRules([...rules, newRule])
        message.success(t('dnsRules.ruleAdded'))
      }
      setIsModalOpen(false)
      setEditingRule(null)
      form.resetFields()
    }).catch(() => {
      message.error(t('dnsRules.checkFormFields'))
    })
  }
  const handleDelete = (id) => {
    setRules(rules.filter(rule => rule.id !== id))
    message.success(t('dnsRules.ruleDeleted'))
  }
  const columns = [
    {
      title: t('dnsRules.id'),
      dataIndex: 'id',
      key: 'id',
      width: 60
    },
    {
      title: t('dnsRules.domain'),
      dataIndex: 'domain',
      key: 'domain',
      ellipsis: true
    },
    {
      title: t('dnsRules.type'),
      dataIndex: 'type',
      key: 'type',
      width: 100
    },
    {
      title: t('dnsRules.value'),
      dataIndex: 'value',
      key: 'value',
      ellipsis: true
    },
    {
      title: t('dnsRules.priority'),
      dataIndex: 'priority',
      key: 'priority',
      width: 100
    },
    {
      title: t('dnsRules.description'),
      dataIndex: 'description',
      key: 'description',
      ellipsis: true
    },
    {
      title: t('dnsRules.actions'),
      key: 'actions',
      width: 150,
      render: (_, record) => (
        <Space size="middle">
          <Tooltip title={t('dnsRules.edit')}>
            <Button
              icon={<EditOutlined />}
              size="small"
              onClick={() => showModal(record)}
            />
          </Tooltip>
          <Tooltip title={t('dnsRules.delete')}>
            <Popconfirm
              title={t('dnsRules.confirmDelete')}
              onConfirm={() => handleDelete(record.id)}
              okText={t('dnsRules.yes')}
              cancelText={t('dnsRules.no')}
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
        <h2>{t('dnsRules.title')}</h2>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => showModal()}
        >
          {t('dnsRules.addRule')}
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
        title={editingRule ? t('dnsRules.editRule') : t('dnsRules.addNewRule')}
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
            label={t('dnsRules.domain')}
            rules={[{ required: true, message: t('dnsRules.pleaseInputDomain') }]}
          >
            <Input placeholder={t('dnsRules.enterDomainName')} />
          </Form.Item>

          <Form.Item
            name="type"
            label={t('dnsRules.type')}
            rules={[{ required: true, message: t('dnsRules.pleaseSelectType') }]}
          >
            <Select placeholder={t('dnsRules.selectDnsType')}>
              <Option value="A">A</Option>
              <Option value="CNAME">CNAME</Option>
              <Option value="MX">MX</Option>
              <Option value="TXT">TXT</Option>
              <Option value="AAAA">AAAA</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="value"
            label={t('dnsRules.value')}
            rules={[{ required: true, message: t('dnsRules.pleaseInputValue') }]}
          >
            <Input placeholder={t('dnsRules.enterDnsValue')} />
          </Form.Item>

          <Form.Item
            name="priority"
            label={t('dnsRules.priority')}
            rules={[{ required: true, message: t('dnsRules.pleaseInputPriority') }]}
          >
            <Input type="number" min={1} max={10} placeholder={t('dnsRules.enterPriority')} />
          </Form.Item>

          <Form.Item
            name="description"
            label={t('dnsRules.description')}
          >
            <Input.TextArea rows={3} placeholder={t('dnsRules.enterDescription')} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default DnsRules