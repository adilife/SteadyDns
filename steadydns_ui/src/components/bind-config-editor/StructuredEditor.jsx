import { useState, useEffect, useRef, useCallback } from 'react'
import {
  Card,
  Form,
  Input,
  InputNumber,
  Switch,
  Select,
  Button,
  Space,
  Alert,
  Empty,
  List,
  Typography,
  Divider,
  Modal
} from 'antd'
import {
  PlusOutlined,
  MinusOutlined,
  EditOutlined,
  SaveOutlined,
  CloseOutlined
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'


const { TextArea } = Input
const { Option } = Select
const { Text } = Typography

const StructuredEditor = ({ config, comments, activeBlock, onConfigChange }) => {
  // 国际化
  const { t } = useTranslation()
  
  const [form] = Form.useForm()
  const [editingComment, setEditingComment] = useState(null)
  const [commentValue, setCommentValue] = useState('')
  
  // 弹窗相关状态
  const [addModalVisible, setAddModalVisible] = useState(false)
  const [deleteModalVisible, setDeleteModalVisible] = useState(false)
  const [addForm] = Form.useForm()
  
  // 防抖相关的ref
  const debounceTimeoutRef = useRef(null)
  const handleConfigChangeRef = useRef(null)

  // 获取当前活动配置块的数据
  const getActiveBlockData = (configData, blockPath) => {
    try {
      if (!configData || !blockPath) {
        return configData
      }

      // 处理API返回的顶层数据结构
      if (configData.data) {
        configData = configData.data
      }

      // 检查configData本身是否就是目标元素
      // 当configData包含name、type等属性，且路径的最后一部分与configData.name匹配时，直接返回
      const pathParts = blockPath.split('|')
      
      // 获取路径的最后一个元素名
      let lastElementName = null;
      if (pathParts.length >= 2) {
        lastElementName = pathParts[pathParts.length - 2];
      } else if (pathParts.length === 1) {
        lastElementName = pathParts[0];
      }
      
      // 检查configData是否为对象且包含name、type等属性，且与路径的最后一部分匹配
      if (typeof configData === 'object' && configData !== null && 
          configData.name && configData.type && 
          lastElementName && configData.name === lastElementName) {
        return configData
      }

      let current = configData

      for (let i = 0; i < pathParts.length; i += 2) {
        const elementName = pathParts[i]
        const elementIndex = pathParts[i + 1] ? parseInt(pathParts[i + 1]) : 0
        
        // 检查current是否为对象
        if (typeof current !== 'object' || current === null) {
          return null
        }
        
        // 处理childElements数组
        if (current.childElements) {
          let foundElement
          
          // 查找元素，使用索引确保找到正确的元素
          // 首先找到所有同名元素
          const elementsWithSameName = current.childElements.filter(element => 
            element.name === elementName
          )
          
          // 根据索引找到正确的元素
          foundElement = elementsWithSameName[elementIndex]
          
          // 如果没找到，尝试直接使用索引查找（兼容其他情况）
          if (!foundElement && elementIndex < current.childElements.length) {
            foundElement = current.childElements[elementIndex]
          }
          
          if (foundElement) {
            current = foundElement
            
            // 检查是否已经到达路径的末尾
            if (i === pathParts.length - 2) {
              return current;
            }
          } else {
            return null
          }
        } else if (current[elementName] !== undefined) {
          // 兼容旧的数据结构
          current = current[elementName]
          
          // 检查是否已经到达路径的末尾
          if (i === pathParts.length - 2) {
            return current;
          }
        } else {
          return null
        }
      }

      return current
    } catch (error) {
      console.error('getActiveBlockData执行出错:', error);
      return null;
    }
  }

  // 根据路径更新配置树
  const updateConfigByPath = useCallback((configData, blockPath, updatedElement) => {
    try {
      // 检查configData是否包含data属性
      const hasDataProperty = configData.data !== undefined;
      
      // 处理API返回的顶层数据结构
      let config = hasDataProperty ? { ...configData.data } : { ...configData };
      
      // 解析路径
      const pathParts = blockPath.split('|');
      
      // 遍历路径，找到目标元素的父级和原始元素
      let current = config;
      let targetIndex = -1;
      let targetName = '';
      let originalElement = null;
      
      // 检查路径长度
      if (pathParts.length < 2) {
        console.error('路径格式错误:', blockPath);
        return configData;
      }
      
      // 遍历路径，找到目标元素的直接父级
      for (let i = 0; i < pathParts.length - 2; i += 2) {
        const elementName = pathParts[i];
        const elementIndex = parseInt(pathParts[i + 1]) || 0;
        
        // 继续遍历到父级
        if (current.childElements) {
          // 找到同名元素的指定索引
          const elementsWithSameName = current.childElements.filter(el => el.name === elementName);
          if (elementsWithSameName[elementIndex]) {
            current = elementsWithSameName[elementIndex];
          } else if (current.childElements[elementIndex]) {
            // 尝试直接使用索引
            current = current.childElements[elementIndex];
          } else {
            console.error('路径部分不存在:', elementName, elementIndex);
            return configData;
          }
        } else {
          console.error('当前元素没有childElements:', current);
          return configData;
        }
      }
      
      // 处理路径的最后一个部分，记录目标信息
      targetName = pathParts[pathParts.length - 2];
      targetIndex = parseInt(pathParts[pathParts.length - 1]) || 0;
      
      // 确定目标元素的直接父级
      const targetParent = current;
      
      // 更新目标元素
      if (targetParent && targetParent.childElements) {
        // 找到目标元素在childElements中的实际索引
        let actualIndex = -1;
        const elementsWithSameName = targetParent.childElements.filter(el => el.name === targetName);
        if (elementsWithSameName[targetIndex]) {
          actualIndex = targetParent.childElements.indexOf(elementsWithSameName[targetIndex]);
          // 保存原始元素
          originalElement = targetParent.childElements[actualIndex];
        } else if (targetIndex < targetParent.childElements.length) {
          actualIndex = targetIndex;
          // 保存原始元素
          originalElement = targetParent.childElements[actualIndex];
        }
        
        if (actualIndex !== -1 && originalElement) {
          // 合并更新：保留原始元素的属性，只更新updatedElement中包含的属性
          const mergedElement = {
            ...originalElement,
            ...updatedElement
          };
          
          // 更新childElements数组
          const updatedChildElements = [...targetParent.childElements];
          updatedChildElements[actualIndex] = mergedElement;
          targetParent.childElements = updatedChildElements;
        } else {
          console.error('未找到目标元素:', targetName, targetIndex, '在', targetParent.name, '的childElements中');
        }
      } else {
        console.error('目标元素的父级没有childElements:', targetParent);
      }
      
      // 根据原始configData的结构，决定返回的结构
      let result;
      if (hasDataProperty) {
        // 如果原始configData包含data属性，返回的结果也应该包含data属性
        result = {
          ...configData,
          data: config
        };
      } else {
        // 如果原始configData不包含data属性，直接返回config
        result = config;
      }
      
      return result;
    } catch (error) {
      console.error('updateConfigByPath执行出错:', error);
      return configData;
    }
  }, []);

  // 处理配置变更
  const handleConfigChange = useCallback((values) => {
    try {
      // 处理 comments 属性，将字符串转换为数组
      const processedValues = {
        ...values,
        comments: typeof values.comments === 'string' ? values.comments.split('\n').filter(line => line.trim() !== '') : values.comments
      };
      
      if (activeBlock) {
        // 根据路径更新配置树
        const updatedConfig = updateConfigByPath(config, activeBlock, processedValues);
        onConfigChange(updatedConfig);
      } else {
        // 更新整个配置
        onConfigChange(processedValues)
      }
    } catch (error) {
      console.error('处理配置变更时出错:', error);
    }
  }, [activeBlock, config, onConfigChange, updateConfigByPath])

  // 处理注释编辑
  const handleCommentEdit = (key) => {
    setEditingComment(key)
    setCommentValue(comments[key] || '')
  }

  // 处理注释保存
  const handleCommentSave = (key) => {
    try {
      // 这里需要实现注释保存逻辑
      // 可以通过 form.setFieldsValue 更新对应字段
      if (key.includes('comments')) {
        form.setFieldsValue({ comments: [...(form.getFieldValue('comments') || []), commentValue] });
      } else if (key.includes('lineComment')) {
        form.setFieldsValue({ lineComment: commentValue });
      }
      setEditingComment(null);
    } catch (error) {
      console.error('保存注释时出错:', error);
      setEditingComment(null);
    }
  }

  // 处理注释取消
  const handleCommentCancel = () => {
    setEditingComment(null)
  }
  
  // 处理增加项
  const handleAddItem = () => {
    // 重置表单
    addForm.resetFields()
    // 显示弹窗
    setAddModalVisible(true)
  }
  
  // 处理增加项确认
  const handleAddItemConfirm = () => {
    addForm.validateFields().then(values => {
      // 构建新元素
      const newElement = {
        name: values.name,
        type: 'simple', // 默认为simple且不可编辑
        value: values.value || '',
        lineComment: values.lineComment || '',
        comments: (values.comments || '').split('\n').filter(line => line.trim() !== ''),
        childElements: []
      }
      
      // 获取当前活动块数据
      const activeData = getActiveBlockData(config, activeBlock)
      
      // 检查当前块是否为block类型
      if (activeData && activeData.type === 'block') {
        // 更新当前块的childElements
        const updatedElement = {
          ...activeData,
          childElements: [...(activeData.childElements || []), newElement]
        }
        
        // 更新配置树
        const updatedConfig = updateConfigByPath(config, activeBlock, updatedElement)
        
        // 通知父组件
        onConfigChange(updatedConfig)
        
        // 关闭弹窗
        setAddModalVisible(false)
      }
    }).catch(error => {
      console.error('增加项表单验证失败:', error)
    })
  }
  
  // 处理删除项
  const handleDeleteItem = () => {
    // 显示确认弹窗
    setDeleteModalVisible(true)
  }
  
  // 处理删除项确认
  const handleDeleteItemConfirm = () => {
    // 获取当前活动块数据
    const activeData = getActiveBlockData(config, activeBlock)
    
    if (activeData) {
      // 构建删除后的配置
      let updatedConfig
      
      // 检查当前块是否有父级
      if (activeBlock && activeBlock.includes('|')) {
        // 构建父级路径
        const pathParts = activeBlock.split('|')
        const parentPath = pathParts.slice(0, -2).join('|')
        
        // 获取父级数据
        const parentData = getActiveBlockData(config, parentPath)
        
        if (parentData && parentData.childElements) {
          // 找到当前元素在父级childElements中的索引
          const elementName = pathParts[pathParts.length - 2]
          
          // 过滤掉当前元素
          const updatedChildElements = parentData.childElements.filter((elem, index) => {
            // 找到同名元素的指定索引
            const elementsWithSameName = parentData.childElements.filter(e => e.name === elementName)
            return elementsWithSameName[index] !== elem
          })
          
          // 更新父级元素
          const updatedParent = {
            ...parentData,
            childElements: updatedChildElements
          }
          
          // 更新配置树
          updatedConfig = updateConfigByPath(config, parentPath, updatedParent)
        }
      } else {
        // 如果没有父级，直接返回空配置
        updatedConfig = null
      }
      
      // 通知父组件
      if (updatedConfig) {
        onConfigChange(updatedConfig)
      }
      
      // 关闭弹窗
      setDeleteModalVisible(false)
    }
  }
  
  // 更新handleConfigChangeRef，确保它始终引用最新的handleConfigChange函数
  useEffect(() => {
    handleConfigChangeRef.current = handleConfigChange
  }, [handleConfigChange])
  
  // 清理函数，在组件卸载时清理防抖定时器
  useEffect(() => {
    return () => {
      if (debounceTimeoutRef.current) {
        clearTimeout(debounceTimeoutRef.current)
      }
    }
  }, [])
  
  // 防抖函数，延迟配置变更处理
  const debouncedHandleConfigChange = useCallback((values) => {
    // 清除之前的定时器
    if (debounceTimeoutRef.current) {
      clearTimeout(debounceTimeoutRef.current)
    }
    
    // 设置新的定时器
    debounceTimeoutRef.current = setTimeout(() => {
      // 调用最新的handleConfigChange函数
      if (handleConfigChangeRef.current) {
        handleConfigChangeRef.current(values)
      }
    }, 300) // 300ms防抖延迟
  }, [])

  // 渲染配置编辑字段
  const renderConfigField = (element, path = '', index = 0) => {
    try {
      // 处理不同的输入形式
      let key, value, type, comments, lineComment, childElements
      
      if (typeof element === 'object' && element !== null) {
        // API返回的元素结构
        key = element.name
        value = element.value
        type = element.type
        comments = element.comments || []
        lineComment = element.lineComment || ''
        childElements = element.childElements || []
      } else {
        // 兼容旧的数据结构
        key = element
        value = element
        type = 'simple'
        comments = []
        lineComment = ''
        childElements = null
      }
      
      // 为元素生成唯一的key，使用统一的格式
      // 使用 | 作为分隔符，避免与元素名中的 _ 和值中的 . 冲突
      let fieldPath;
      if (path) {
        // 如果有父路径，使用父路径和 `${key}|${index}` 格式
        fieldPath = `${path}|${key}|${index}`;
      } else {
        // 如果没有父路径，使用 `${key}|${index}` 格式
        fieldPath = `${key}|${index}`;
      }
      const fieldComment = comments[fieldPath] || comments.join('\n') || lineComment

      // 处理块元素
      if (type === 'block' || (childElements && childElements.length > 0)) {
        return (
          <Card
            key={fieldPath}
            title={
              <Space>
                <Text strong>{key}</Text>
                {fieldComment && (
                  <Button
                    icon={<EditOutlined />}
                    size="small"
                    onClick={() => handleCommentEdit(fieldPath)}
                  >
                    {t('structuredEditor.editComment')}
                  </Button>
                )}
              </Space>
            }
            style={{ marginBottom: 16 }}
          >
            {fieldComment && editingComment !== fieldPath && (
              <Alert
                title={t('structuredEditor.comment')}
                description={fieldComment}
                type="info"
                style={{ marginBottom: 16 }}
              />
            )}
            {editingComment === fieldPath && (
              <div style={{ marginBottom: 16 }}>
                <TextArea
                  value={commentValue}
                  onChange={(e) => setCommentValue(e.target.value)}
                  placeholder={t('structuredEditor.enterConfigComment')}
                  style={{ marginBottom: 8 }}
                />
                <Space>
                  <Button icon={<SaveOutlined />} onClick={() => handleCommentSave(fieldPath)}>
                    {t('structuredEditor.save')}
                  </Button>
                  <Button icon={<CloseOutlined />} onClick={handleCommentCancel}>
                    {t('structuredEditor.cancel')}
                  </Button>
                </Space>
              </div>
            )}
            <div>
              {childElements && childElements.length > 0 ? (
                childElements.map((childElement, index) => (
                  renderConfigField(childElement, fieldPath, index)
                ))
              ) : Object.entries(value || {}).map(([subKey, subValue], index) => (
                renderConfigField({ name: subKey, value: subValue, type: 'simple' }, fieldPath, index)
              ))}
            </div>
          </Card>
        )
      } else {
        // 处理简单元素
        return (
          <Form.Item
            key={fieldPath}
            label={
              <Space>
                {key}
                {fieldComment && (
                  <Button
                    icon={<EditOutlined />}
                    size="small"
                    onClick={() => handleCommentEdit(fieldPath)}
                  >
                    {t('structuredEditor.editComment')}
                  </Button>
                )}
              </Space>
            }
            name={key}
            style={{ marginBottom: 16 }}
          >
            <div>
              {fieldComment && editingComment !== fieldPath && (
                <Alert
                  title={t('structuredEditor.comment')}
                  description={fieldComment}
                  type="info"
                  style={{ marginBottom: 8 }}
                />
              )}
              {editingComment === fieldPath && (
                <div style={{ marginBottom: 8 }}>
                  <TextArea
                    value={commentValue}
                    onChange={(e) => setCommentValue(e.target.value)}
                    placeholder={t('structuredEditor.enterConfigComment')}
                    style={{ marginBottom: 8 }}
                  />
                  <Space>
                    <Button icon={<SaveOutlined />} onClick={() => handleCommentSave(fieldPath)}>
                      {t('structuredEditor.save')}
                    </Button>
                    <Button icon={<CloseOutlined />} onClick={handleCommentCancel}>
                      {t('structuredEditor.cancel')}
                    </Button>
                  </Space>
                </div>
              )}
              {!editingComment && renderFieldControl(key, value)}
            </div>
          </Form.Item>
        )
      }
    } catch (error) {
      console.error('渲染配置字段时出错:', error);
      return (
        <Form.Item label={t('structuredEditor.error')}>
          <Alert
            title={t('structuredEditor.renderError')}
            description={t('structuredEditor.renderErrorDetail', { message: error.message })}
            type="error"
            showIcon
          />
        </Form.Item>
      );
    }
  }

  // 根据值类型渲染不同的控件
  const renderFieldControl = (key, value) => {
    if (typeof value === 'string') {
      return <TextArea value={value} rows={4} />
    } else if (typeof value === 'number') {
      return <InputNumber value={value} style={{ width: '100%' }} />
    } else if (typeof value === 'boolean') {
      return <Switch checked={value} />
    } else if (Array.isArray(value)) {
      return (
        <List
          dataSource={value}
          renderItem={(item, index) => (
            <List.Item
              actions={[
                <Button
                    icon={<MinusOutlined />}
                    size="small"
                    danger
                    onClick={() => {
                      const newArray = [...value]
                      newArray.splice(index, 1)
                      form.setFieldsValue({ [key]: newArray })
                    }}
                  >
                    {t('structuredEditor.deleteItem')}
                  </Button>
              ]}
            >
              <Input value={item} />
            </List.Item>
          )}
          footer={
            <Button
              icon={<PlusOutlined />}
              onClick={() => {
                const newArray = [...value, '']
                form.setFieldsValue({ [key]: newArray })
              }}
            >
              {t('structuredEditor.addItem')}
            </Button>
          }
        />
      )
    } else {
      return <TextArea value={String(value)} rows={4} />
    }
  }

  // 初始化当前配置
  useEffect(() => {
    try {
      const activeData = getActiveBlockData(config, activeBlock)
      
      // 避免在useEffect中直接调用setState，这里我们只需要设置表单值
      if (activeData !== null) {
        // 处理API返回的元素结构
        if (typeof activeData === 'object' && activeData !== null) {
          // 对于复杂对象的情况
          if ((activeData.type === 'simple' || activeData.type === 'block') && activeData.name) {
            // 对于simple或block类型的元素，构建完整的表单值对象
            const formValues = {
              name: activeData.name,
              type: activeData.type,
              value: activeData.value,
              comments: (activeData.comments || []).join('\n'),
              lineComment: activeData.lineComment || ''
            };
            form.setFieldsValue(formValues);
          } else if (activeData.childElements && activeData.childElements.length > 0) {
            // 对于有childElements但不是simple或block类型的元素
            const formValues = {}
            activeData.childElements.forEach(element => {
              formValues[element.name] = element.value
            })
            form.setFieldsValue(formValues)
          } else {
            form.setFieldsValue(activeData)
          }
        } else {
          // 对于简单值类型（字符串、数字、布尔值等）
          form.setFieldsValue({ value: activeData })
        }
      }
    } catch (error) {
      console.error('useEffect执行出错:', error);
    }
  }, [config, activeBlock, form])

  if (!config) {
    return <Empty description={t('structuredEditor.noConfigData')} style={{ margin: '40px 0' }} />
  }

  const activeData = getActiveBlockData(config, activeBlock)

  if (activeBlock && !activeData) {
    return <Empty description={t('structuredEditor.configBlockNotFound')} style={{ margin: '40px 0' }} />
  }

  return (
    <div>
      <Card title={activeBlock ? t('structuredEditor.editConfigBlock', { block: activeBlock }) : t('structuredEditor.editConfig')}>
        <Form
          form={form}
          layout="vertical"
          onValuesChange={(changedValues, allValues) => {
            debouncedHandleConfigChange(allValues)
          }}
        >
          {activeData && typeof activeData === 'object' && activeData !== null ? (
            <div>
              {/* 操作按钮 */}
              <div style={{ marginBottom: 16, display: 'flex', gap: 12 }}>
                {/* 增加项按钮，仅在当前元素为block类型时显示 */}
                {activeData.type === 'block' && (
                  <Button type="primary" onClick={handleAddItem}>
                    {t('structuredEditor.addItem')}
                  </Button>
                )}
                
                {/* 删除项按钮 */}
                <Button danger onClick={handleDeleteItem}>
                  {t('structuredEditor.deleteItem')}
                </Button>
              </div>
              
              {/* 检查是否为simple或block类型元素 */}
              {(activeData.type === 'simple' || activeData.type === 'block') && activeData.name ? (
                <Card title={activeData.name} style={{ marginBottom: 16 }}>
                  {/* 只读属性 */}
                  <Form.Item label={activeData.type === 'simple' && activeData.value === '' ? t('structuredEditor.value') : t('structuredEditor.name')} name="name">
                    <Input disabled={!(activeData.type === 'simple' && activeData.value === '')} />
                  </Form.Item>
                  <Form.Item label={t('structuredEditor.type')} name="type">
                    <Input disabled />
                  </Form.Item>
                  
                  {/* 可编辑属性 */}
                  {activeData.type !== 'block' && !(activeData.type === 'simple' && activeData.value === '') && (
                    <Form.Item label={t('structuredEditor.value')} name="value">
                      {renderFieldControl('value', activeData.value)}
                    </Form.Item>
                  )}
                  <Form.Item label={t('structuredEditor.lineComment')} name="lineComment">
                    <Input placeholder={t('structuredEditor.enterLineComment')} />
                  </Form.Item>
                  <Form.Item label={t('structuredEditor.configComment')} name="comments">
                    <TextArea 
                      placeholder={t('structuredEditor.enterConfigComment')} 
                      rows={4}
                    />
                  </Form.Item>
                </Card>
              ) : activeData.childElements && activeData.childElements.length > 0 ? (
                // 如果不是simple或block类型元素，但有childElements，显示childElements
                activeData.childElements.map((element, index) => (
                  renderConfigField(element, activeBlock, index)
                ))
              ) : (
                // 检查是否为简单值包装对象
                Object.keys(activeData).length === 1 && 'value' in activeData ? (
                  <Form.Item label={t('structuredEditor.value')}>
                    {renderFieldControl('value', activeData.value)}
                  </Form.Item>
                ) : (
                  Object.entries(activeData).map(([key, value], index) => (
                    renderConfigField({ name: key, value: value, type: 'simple' }, activeBlock, index)
                  ))
                )
              )}
            </div>
          ) : (
            <Form.Item label={t('structuredEditor.value')}>
              {renderFieldControl('value', activeData)}
            </Form.Item>
          )}
        </Form>
      </Card>
      
      {!activeBlock && (
        <Card title={t('structuredEditor.editTip')} style={{ marginTop: 16 }}>
          <Alert
            title={t('structuredEditor.editTip')}
            description={t('structuredEditor.editTipDescription')}
            type="info"
            showIcon
          />
        </Card>
      )}
      
      {/* 增加项弹窗 */}
      <Modal
        title={t('structuredEditor.addItemModal')}
        open={addModalVisible}
        onCancel={() => setAddModalVisible(false)}
        onOk={handleAddItemConfirm}
      >
        {/* 显示父对象信息 */}
        {activeData && (
          <div style={{ marginBottom: 16, padding: 12, backgroundColor: '#f5f5f5', borderRadius: 4 }}>
            <p><strong>{t('structuredEditor.parentObjectName')}</strong> {activeData.name}</p>
            <p><strong>{t('structuredEditor.parentObjectType')}</strong> {activeData.type}</p>
          </div>
        )}
        <Form
          form={addForm}
          layout="vertical"
        >
          <Form.Item
            label={t('structuredEditor.name')}
            name="name"
            rules={[{ required: true, message: t('structuredEditor.enterName') }]}
          >
            <Input placeholder={t('structuredEditor.namePlaceholder')} />
          </Form.Item>
          <Form.Item
            label={t('structuredEditor.value')}
            name="value"
          >
            <TextArea placeholder={t('structuredEditor.valuePlaceholder')} rows={4} />
          </Form.Item>
          <Form.Item
            label={t('structuredEditor.lineComment')}
            name="lineComment"
          >
            <Input placeholder={t('structuredEditor.lineCommentPlaceholder')} />
          </Form.Item>
          <Form.Item
            label={t('structuredEditor.configComment')}
            name="comments"
          >
            <TextArea
              placeholder={t('structuredEditor.configCommentPlaceholder')}
              rows={4}
            />
          </Form.Item>
        </Form>
      </Modal>
      
      {/* 删除项确认弹窗 */}
      <Modal
        title={t('structuredEditor.deleteItemConfirm')}
        open={deleteModalVisible}
        onCancel={() => setDeleteModalVisible(false)}
        onOk={handleDeleteItemConfirm}
        okText={t('structuredEditor.confirmDelete')}
        cancelText={t('structuredEditor.cancel')}
        okType="danger"
      >
        <p>{t('structuredEditor.confirmDeleteMessage')}</p>
        {activeData && (
          <div style={{ marginTop: 16, padding: 12, backgroundColor: '#f5f5f5', borderRadius: 4 }}>
            <p><strong>{t('structuredEditor.objectName')}</strong> {activeData.name}</p>
            <p><strong>{t('structuredEditor.objectType')}</strong> {activeData.type}</p>
            {activeData.type === 'block' && activeData.childElements && (
              <p><strong>{t('structuredEditor.childObjectCount')}</strong> {activeData.childElements.length}</p>
            )}
          </div>
        )}
      </Modal>
    </div>
  )
}


export default StructuredEditor
