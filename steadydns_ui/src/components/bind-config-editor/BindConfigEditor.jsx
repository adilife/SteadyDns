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
import React, { useState, useEffect, useCallback } from 'react'
import {
  Card,
  Row,
  Col,
  Button,
  Space,
  message,
  Spin,
  Tabs,
  Alert,
  Modal
} from 'antd'
import {
  SaveOutlined,
  CheckCircleOutlined,
  RollbackOutlined,
  DiffOutlined,
  EditOutlined,
  CodeOutlined,
  LoadingOutlined
} from '@ant-design/icons'
import { t } from '../../i18n'
import { apiClient } from '../../utils/apiClient'
import ConfigNavigator from './ConfigNavigator'
import StructuredEditor from './StructuredEditor'
import RawEditor from './RawEditor'
import ConfigValidator from './ConfigValidator'
import DiffViewer from './DiffViewer'

// 错误边界组件
class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = { hasError: false, error: null, errorInfo: null };
  }

  static getDerivedStateFromError(error) {
    return { hasError: true, error };
  }

  componentDidCatch(error, errorInfo) {
    console.error('子组件渲染错误:', error);
    console.error('错误信息:', errorInfo);
    this.setState({ errorInfo });
  }

  render() {
    if (this.state.hasError) {
      return (
        <div style={{ padding: 20, backgroundColor: '#fff1f0', borderRadius: 4 }}>
          <h3>组件渲染错误</h3>
          <p>{this.state.error?.message}</p>
          <pre>{this.state.error?.stack}</pre>
          {this.state.errorInfo && (
            <div>
              <h4>错误信息详情:</h4>
              <pre>{JSON.stringify(this.state.errorInfo, null, 2)}</pre>
            </div>
          )}
        </div>
      );
    }

    return this.props.children;
  }
}

const { TabPane } = Tabs

const BindConfigEditor = ({ visible, onClose }) => {
  // 状态管理
  const [configState, setConfigState] = useState({
    // 原始配置（加载后不再修改）
    original: {
      rawContent: '',
      structuredConfig: null
    },
    // 当前配置（用户修改的版本）
    current: {
      rawContent: '',
      structuredConfig: null,
      hasChanges: false
    },
    // 其他状态
    activeMode: 'structured',
    activeConfigBlock: null,
    comments: {},
    loading: false,
    lastSaved: null,
    // 验证状态
    validation: {
      isValid: null,
      errors: []
    }
  })

  // 模态框状态
  const [modalState, setModalState] = useState({
    validateModalVisible: false,
    diffModalVisible: false,
    saveConfirmModalVisible: false,
    validateResult: null,
    diffResult: null
  })

  // 加载配置数据
  const loadConfigData = useCallback(async () => {
    setConfigState(prev => ({
      ...prev, 
      loading: true 
    }))
    try {
      // 并行加载配置数据
      const [parseResponse, contentResponse] = await Promise.all([
        apiClient.getBindNamedConfParse(),
        apiClient.getBindNamedConfContent()
      ])

      // 处理错误消息
      if (!parseResponse.success) {
        message.error(parseResponse.message || t('bindConfigEditor.loadStructuredError'));
      }

      if (!contentResponse.success) {
        message.error(contentResponse.message || t('bindConfigEditor.loadRawError'));
      }

      // 合并状态更新，确保原子性
      setConfigState(prev => {
        const newState = { ...prev };
        
        // 初始化原始配置和当前配置
        const structuredConfig = parseResponse.success ? parseResponse.data : {};
        const rawContent = contentResponse.success ? contentResponse.data.content : '';
        
        // 设置原始配置（加载后不再修改）
        newState.original = {
          rawContent,
          structuredConfig
        };
        
        // 设置当前配置（初始为原始配置的副本）
        newState.current = {
          rawContent,
          structuredConfig,
          hasChanges: false
        };
        
        // 更新注释
        newState.comments = extractComments();
        
        // 查找并设置第一个可选择的配置块
        const firstSelectableBlock = findFirstSelectableBlock(structuredConfig);
        newState.activeConfigBlock = firstSelectableBlock;
        
        // 设置loading状态为false
        newState.loading = false;
        
        return newState;
      });

    } catch (error) {
      console.error('Error loading config data:', error)
      message.error(t('bindConfigEditor.loadError'))
      // 错误时也需要设置loading状态为false，并提供默认值
      setConfigState(prev => ({
        ...prev, 
        loading: false,
        original: {
          rawContent: '',
          structuredConfig: {}
        },
        current: {
          rawContent: '',
          structuredConfig: {},
          hasChanges: false
        },
        comments: {}
      }))
    }
  }, [])

  // 提取注释
  const extractComments = () => {
    // 这里需要实现注释提取逻辑
    // 暂时返回空对象
    return {}
  }

  // 查找第一个可选择的配置块
  const findFirstSelectableBlock = (configData, parentKey = '') => {
    if (!configData) {
      return null
    }

    // 处理API返回的顶层数据结构
    if (configData.data) {
      configData = configData.data
    }

    // 处理根元素和块元素的childElements
    const elements = configData.childElements || []

    // 遍历元素，找到第一个可选择的配置块
    for (let i = 0; i < elements.length; i++) {
      const element = elements[i]
      // 生成唯一的key，使用与ConfigNavigator一致的格式
      const itemKey = parentKey ? `${parentKey}|${element.name}|${i}` : `${element.name}|${i}`

      // 如果元素有子元素，优先选择它（因为它通常是主要的配置块）
      if (element.childElements && element.childElements.length > 0) {
        return itemKey
      }
    }

    // 如果没有找到有子元素的块元素，返回第一个简单元素
    if (elements.length > 0) {
      const firstElement = elements[0]
      return parentKey ? `${parentKey}|${firstElement.name}|0` : `${firstElement.name}|0`
    }

    return null
  }

  // 切换编辑模式
  const toggleEditMode = (mode) => {
    setConfigState(prev => ({
      ...prev,
      activeMode: mode
    }))
  }

  // 验证配置
  const validateConfig = async () => {
    setConfigState(prev => ({ ...prev, loading: true }))
    try {
      const configToValidate = {
        content: configState.activeMode === 'structured' 
          ? generateRawConfigFromStructured(configState.current.structuredConfig)
          : configState.current.rawContent
      }

      const response = await apiClient.validateBindNamedConf(configToValidate)
      
      if (response.success) {
        const isValid = response.data?.valid === true
        setModalState(prev => ({
          ...prev,
          validateModalVisible: true,
          validateResult: response.data
        }))
        setConfigState(prev => ({
          ...prev,
          validation: {
            isValid: isValid,
            errors: isValid ? [] : [response.data?.error || '配置验证失败']
          },
          loading: false
        }))
      } else {
        // 构建包含错误信息的 result 对象
        const errorResult = {
          valid: false,
          error: response.error || '配置验证失败'
        }
        setModalState(prev => ({
          ...prev,
          validateModalVisible: true,
          validateResult: errorResult  // 传递错误信息对象
        }))
        setConfigState(prev => ({
          ...prev,
          validation: {
            isValid: false,
            errors: [response.error || '配置验证失败']
          },
          loading: false
        }))
      }
    } catch (error) {
      console.error('Error validating config:', error)
      message.error(t('bindConfigEditor.validateError'))
      setConfigState(prev => ({ ...prev, loading: false }))
    }
  }

  // 从结构化配置生成原始配置
  const generateRawConfigFromStructured = (structuredConfig) => {
    // 这里需要实现从结构化配置到原始配置的转换
    // 示例实现：遍历结构化配置，生成对应的配置文本
    if (!structuredConfig) return ''
    
    // 递归函数，用于遍历结构化配置并生成配置文本
    const traverseConfig = (config, indent = 0) => {
      let result = ''
      const indentStr = '  '.repeat(indent)
      
      // 处理注释
      if (config.comments && config.comments.length > 0) {
        config.comments.forEach(comment => {
          result += `${indentStr}// ${comment}\n`
        })
        result += '\n'
      }
      
      // 处理不同类型的元素
      switch (config.type) {
        case 'root':
          // 根元素，遍历其子元素
          if (config.childElements && config.childElements.length > 0) {
            config.childElements.forEach(element => {
              result += traverseConfig(element, indent)
              // 在元素之间添加空行
              result += '\n'
            })
          }
          break
          
        case 'block':
          // 块元素，生成 name { ... } 格式
          result += `${indentStr}${config.name} {\n`
          
          if (config.childElements && config.childElements.length > 0) {
            result += '\n'
            config.childElements.forEach(element => {
              result += traverseConfig(element, indent + 1)
            })
            result += '\n'
          }
          
          result += `${indentStr}};${config.lineComment ? ' // ' + config.lineComment : ''}\n`
          break
          
        case 'simple':
          // 简单元素，生成 name value; 格式
          result += `${indentStr}${config.name} ${config.value || ''};${config.lineComment ? ' // ' + config.lineComment : ''}\n`
          break
          
        default:
          // 未知类型，跳过
          break
      }
      
      return result
    }
    
    // 调用递归函数，从根元素开始遍历
    return traverseConfig(structuredConfig)
  }

  // 查看差异
  const viewDiff = async () => {
    setConfigState(prev => ({ ...prev, loading: true }))
    try {
      const configToDiff = {
        oldContent: configState.original.rawContent,
        newContent: configState.activeMode === 'structured' 
          ? generateRawConfigFromStructured(configState.current.structuredConfig)
          : configState.current.rawContent
      }

      const response = await apiClient.getBindNamedConfDiff(configToDiff)
      
      if (response.success) {
        setModalState(prev => ({
          ...prev,
          diffModalVisible: true,
          diffResult: response.data
        }))
      } else {
        message.error(response.message || t('bindConfigEditor.diffError'))
      }
    } catch (error) {
      console.error('Error getting diff:', error)
      message.error(t('bindConfigEditor.diffError'))
    } finally {
      setConfigState(prev => ({ ...prev, loading: false }))
    }
  }

  // 保存配置
  const saveConfig = async () => {
    setConfigState(prev => ({ ...prev, loading: true }))
    try {
      const configToSave = {
        content: configState.activeMode === 'structured' 
          ? generateRawConfigFromStructured(configState.current.structuredConfig)
          : configState.current.rawContent
      }

      const response = await apiClient.updateBindNamedConf(configToSave)
      
      if (response.success) {
        message.success(t('bindConfigEditor.saveSuccess'))
        setConfigState(prev => ({
          ...prev,
          current: {
            ...prev.current,
            hasChanges: false
          },
          lastSaved: new Date().toISOString()
        }))
        // 重新加载配置以确保同步
        await loadConfigData()
      } else {
        message.error(response.message || t('bindConfigEditor.saveError'))
      }
    } catch (error) {
      console.error('Error saving config:', error)
      message.error(t('bindConfigEditor.saveError'))
    } finally {
      setConfigState(prev => ({ ...prev, loading: false }))
    }
  }

  // 处理配置变更
  const handleConfigChange = (newContent, mode) => {
    if (mode === 'structured') {
      setConfigState(prev => ({
        ...prev,
        current: {
          ...prev.current,
          structuredConfig: newContent,
          hasChanges: true
        }
      }))
    } else {
      setConfigState(prev => ({
        ...prev,
        current: {
          ...prev.current,
          rawContent: newContent,
          hasChanges: true
        }
      }))
    }
  }

  // 处理配置块选择
  const handleConfigBlockSelect = (blockId) => {
    setConfigState(prev => ({
      ...prev,
      activeConfigBlock: blockId
    }))
  }

  // 初始化加载
  useEffect(() => {
    if (visible) {
      loadConfigData()
    }
  }, [visible, loadConfigData])



  return (
    <Modal
        title={t('bindConfigEditor.title')}
        open={visible}
        onCancel={onClose}
        width={1200}
        style={{ height: 800 }}
        maskClosable={false}
        keyboard={false}
        footer={null}
        destroyOnHidden
      >
      <div style={{ height: 700, display: 'flex', flexDirection: 'column' }}>
        {/* 顶部工具栏 */}
        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Space>
            <Button
              type="primary"
              icon={<SaveOutlined />}
              onClick={() => setModalState(prev => ({ ...prev, saveConfirmModalVisible: true }))}
              loading={configState.loading}
              disabled={!configState.hasChanges || configState.loading}
              tooltip="保存配置更改"
            >
              {t('bindConfigEditor.save')}
            </Button>
            <Button
              icon={<CheckCircleOutlined />}
              onClick={validateConfig}
              loading={configState.loading}
              disabled={configState.loading}
              tooltip="验证配置有效性"
            >
              {t('bindConfigEditor.validate')}
            </Button>
            <Button
              icon={<RollbackOutlined />}
              onClick={loadConfigData}
              loading={configState.loading}
              disabled={configState.loading}
              tooltip="撤销更改，重新加载配置"
            >
              {t('bindConfigEditor.undo')}
            </Button>
            <Button
              icon={<DiffOutlined />}
              onClick={viewDiff}
              loading={configState.loading}
              disabled={configState.loading}
              tooltip="查看配置更改差异"
            >
              {t('bindConfigEditor.viewDiff')}
            </Button>
          </Space>
          <Space>
            <Button
              icon={<EditOutlined />}
              type={configState.activeMode === 'structured' ? 'primary' : 'default'}
              onClick={() => toggleEditMode('structured')}
            >
              {t('bindConfigEditor.structuredEdit')}
            </Button>
            <Button
              icon={<CodeOutlined />}
              type={configState.activeMode === 'raw' ? 'primary' : 'default'}
              onClick={() => toggleEditMode('raw')}
            >
              {t('bindConfigEditor.rawEdit')}
            </Button>
          </Space>
        </div>

        {/* 主内容区域 */}
        <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
          {/* 左侧导航 */}
          <div style={{ width: 250, borderRight: '1px solid #f0f0f0', overflow: 'auto' }}>
            <ConfigNavigator
              config={configState.current.structuredConfig}
              activeBlock={configState.activeConfigBlock}
              onSelectBlock={handleConfigBlockSelect}
            />
          </div>

          {/* 右侧编辑区 */}
          <div style={{ flex: 1, padding: 16, overflow: 'auto' }}>
          <Spin spinning={configState.loading} >
              {configState.activeMode === 'structured' ? (

                // 直接渲染StructuredEditor，不使用ErrorBoundary
                <StructuredEditor
                  config={configState.current.structuredConfig}
                  comments={configState.comments}
                  activeBlock={configState.activeConfigBlock}
                  onConfigChange={(newConfig) => handleConfigChange(newConfig, 'structured')}
                />
              ) : (
                <ErrorBoundary>
                  <RawEditor
                    content={configState.current.rawContent}
                    onContentChange={(newContent) => handleConfigChange(newContent, 'raw')}
                  />
                </ErrorBoundary>
              )}
            </Spin>
          </div>
        </div>

        {/* 底部状态栏 */}
        <div style={{ marginTop: 16, paddingTop: 16, borderTop: '1px solid #f0f0f0', display: 'flex', justifyContent: 'space-between' }}>
          <div>
            <Space>
              <span>{t('bindConfigEditor.status')}: </span>
              <span style={{ color: configState.validation.isValid === true ? '#52c41a' : configState.validation.isValid === false ? '#ff4d4f' : '#1890ff' }}>
                {configState.validation.isValid === true ? t('bindConfigEditor.valid') : configState.validation.isValid === false ? t('bindConfigEditor.invalid') : t('bindConfigEditor.unverified')}
              </span>
              {configState.current.hasChanges && (
                <span style={{ color: '#faad14' }}>{t('bindConfigEditor.unsavedChanges')}</span>
              )}
            </Space>
          </div>
          <div>
            {configState.lastSaved && (
              <span>{t('bindConfigEditor.lastSaved')}: {new Date(configState.lastSaved).toLocaleString()}</span>
            )}
          </div>
        </div>
      </div>

      {/* 验证结果模态框 */}
      <Modal
        title={t('bindConfigEditor.validateResult')}
        open={modalState.validateModalVisible}
        onCancel={() => setModalState(prev => ({ ...prev, validateModalVisible: false }))}
        footer={[
          <Button key="ok" type="primary" onClick={() => setModalState(prev => ({ ...prev, validateModalVisible: false }))}>
            {t('bindConfigEditor.ok')}
          </Button>
        ]}
      >
        <ConfigValidator result={modalState.validateResult} />
      </Modal>

      {/* 差异查看模态框 */}
      <Modal
        title={t('bindConfigEditor.configDiff')}
        open={modalState.diffModalVisible}
        onCancel={() => setModalState(prev => ({ ...prev, diffModalVisible: false }))}
        footer={[
          <Button key="ok" type="primary" onClick={() => setModalState(prev => ({ ...prev, diffModalVisible: false }))}>
            {t('bindConfigEditor.ok')}
          </Button>
        ]}
        width={1000}
      >
        <DiffViewer diff={modalState.diffResult} />
      </Modal>

      {/* 保存确认模态框 */}
      <Modal
        title={t('bindConfigEditor.saveConfirm')}
        open={modalState.saveConfirmModalVisible}
        onCancel={() => setModalState(prev => ({ ...prev, saveConfirmModalVisible: false }))}
        footer={[
          <Button key="cancel" onClick={() => setModalState(prev => ({ ...prev, saveConfirmModalVisible: false }))}>
            {t('bindConfigEditor.cancel')}
          </Button>,
          <Button key="ok" type="primary" onClick={() => {
            setModalState(prev => ({ ...prev, saveConfirmModalVisible: false }))
            saveConfig()
          }}>
            {t('bindConfigEditor.confirmSave')}
          </Button>
        ]}
      >
        <p>{t('bindConfigEditor.confirmSaveMessage')}</p>
        {configState.isValid === false && (
          <Alert
            message={t('bindConfigEditor.warning')}
            description={t('bindConfigEditor.invalidConfigWarning')}
            type="warning"
            showIcon
            style={{ marginTop: 16 }}
          />
        )}
      </Modal>
    </Modal>
  )
}

export default BindConfigEditor
