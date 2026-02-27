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

import { Modal, Typography, Space, Divider, Alert, Flex } from 'antd'
import { BugOutlined, FileTextOutlined } from '@ant-design/icons'
import { t } from '../../i18n'
import { VERSION_INFO } from '../../config/version'

const { Text, Paragraph, Link } = Typography

/**
 * 关于弹窗组件
 * @param {Object} props - 组件属性
 * @param {boolean} props.open - 弹窗是否打开
 * @param {Function} props.onCancel - 关闭弹窗回调
 * @param {string} props.currentLanguage - 当前语言
 */
const AboutModal = ({ open, onCancel, currentLanguage }) => {
  const isRTL = currentLanguage === 'ar-SA'

  return (
    <Modal
      title={t('about.title', currentLanguage)}
      open={open}
      onCancel={onCancel}
      footer={null}
      centered
      width={480}
      style={{ direction: isRTL ? 'rtl' : 'ltr' }}
    >
      <Flex vertical gap="middle" style={{ width: '100%' }}>
        {/* 版本信息 */}
        <div style={{ textAlign: isRTL ? 'right' : 'left' }}>
          <Text strong>{t('about.version', currentLanguage)}: </Text>
          <Text code>{VERSION_INFO.version}</Text>
        </div>

        {/* 测试版本警告 */}
        {VERSION_INFO.isBeta && (
          <Alert
            type="warning"
            showIcon
            style={{ textAlign: isRTL ? 'right' : 'left' }}
          >
            {t('about.betaWarning', currentLanguage)}
          </Alert>
        )}

        {/* 项目描述 */}
        <Paragraph style={{ textAlign: isRTL ? 'right' : 'left', marginBottom: 0 }}>
          {t('about.description', currentLanguage)}
        </Paragraph>

        <Divider style={{ margin: '12px 0' }} />

        {/* 许可证和版权 */}
        <div style={{ textAlign: isRTL ? 'right' : 'left' }}>
          <Text type="secondary">{t('about.license', currentLanguage)}: </Text>
          <Text>{VERSION_INFO.license}</Text>
        </div>
        <div style={{ textAlign: isRTL ? 'right' : 'left' }}>
          <Text type="secondary">{VERSION_INFO.copyright}</Text>
        </div>

        <Divider style={{ margin: '12px 0' }} />

        {/* 链接 */}
        <Flex vertical gap="small" style={{ width: '100%' }}>
          <Link 
            href={VERSION_INFO.issuesUrl} 
            target="_blank" 
            rel="noopener noreferrer"
            style={{ display: 'flex', alignItems: 'center', gap: '8px' }}
          >
            <BugOutlined />
            {t('about.reportBug', currentLanguage)}
          </Link>
          <Link 
            href={VERSION_INFO.changelogUrl} 
            target="_blank" 
            rel="noopener noreferrer"
            style={{ display: 'flex', alignItems: 'center', gap: '8px' }}
          >
            <FileTextOutlined />
            {t('about.changelog', currentLanguage)}
          </Link>
        </Flex>
      </Flex>
    </Modal>
  )
}

export default AboutModal
