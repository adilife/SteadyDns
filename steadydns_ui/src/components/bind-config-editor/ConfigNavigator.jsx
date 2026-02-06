import { useState, useEffect, useCallback } from 'react'
import {
  Menu,
  Input,
  Empty,
  Spin
} from 'antd'
import {
  SearchOutlined,
  FolderOutlined,
  FileOutlined,
  RightOutlined,
  DownOutlined
} from '@ant-design/icons'
import { t } from '../../i18n'


const { Search } = Input

const ConfigNavigator = ({ config, activeBlock, onSelectBlock }) => {
  const [searchText, setSearchText] = useState('')
  const [filteredMenu, setFilteredMenu] = useState([])
  const [loading, setLoading] = useState(false)

  // 生成导航菜单
  const generateMenu = useCallback((configData, parentKey = '') => {
    if (!configData) {
      return []
    }

    // 处理API返回的顶层数据结构
    if (configData.data) {
      configData = configData.data
    }

    // 处理根元素和块元素的childElements
    const elements = configData.childElements || []
    
    return elements.map((element, index) => {
      // 生成唯一的key，使用统一的格式
      // 使用 | 作为分隔符，避免与元素名中的 _ 和值中的 . 冲突
      const itemKey = parentKey ? `${parentKey}|${element.name}|${index}` : `${element.name}|${index}`
      
      // 处理有子元素的块
      if (element.childElements && element.childElements.length > 0) {
        return {
          key: itemKey,
          label: element.name + (element.value ? ` (${element.value})` : ''),
          icon: <FolderOutlined />,
          children: generateMenu(element, itemKey)
        }
      } else {
        // 处理简单指令
        return {
          key: itemKey,
          label: element.name + (element.value ? ` (${element.value})` : ''),
          icon: <FileOutlined />
        }
      }
    })
  }, [])

  // 过滤菜单
  const filterMenu = (menu, search) => {
    if (!search) {
      return menu
    }

    const filtered = []

    const filterItem = (item) => {
      if (item.label.toLowerCase().includes(search.toLowerCase())) {
        return true
      }
      if (item.children) {
        const filteredChildren = item.children.filter(filterItem)
        if (filteredChildren.length > 0) {
          item.children = filteredChildren
          return true
        }
      }
      return false
    }

    menu.forEach(item => {
      const clonedItem = JSON.parse(JSON.stringify(item))
      if (filterItem(clonedItem)) {
        filtered.push(clonedItem)
      }
    })

    return filtered
  }

  // 处理搜索
  const handleSearch = (value) => {
    setSearchText(value)
  }

  // 处理菜单选择
  const handleMenuSelect = (info) => {
    // Ant Design Menu组件的onSelect事件对象结构与onClick不同
    // onSelect事件对象包含key属性，直接传递给onSelectBlock
    onSelectBlock(info.key)
  }
  
  // 处理菜单点击
  const handleMenuClick = (e) => {
    // 保持原有逻辑，确保文件项点击时也能触发编辑区响应
    onSelectBlock(e.key)
  }
  
  // 处理菜单展开/折叠
  const handleMenuOpenChange = (openKeys) => {
    // 对于inline模式的Menu组件，有子元素的菜单项（目录项）点击时只会展开/折叠子菜单
    // 不会触发onSelect事件，所以我们需要使用onOpenChange事件来处理
    // 但是onOpenChange事件的事件对象是一个数组，包含当前展开的菜单项的key值
    // 我们需要获取最后展开的菜单项的key值，然后传递给onSelectBlock
    // 注意：当用户点击一个已经展开的菜单项时，它会被从openKeys数组中移除
    // 此时我们不应该调用onSelectBlock，因为用户可能只是想折叠菜单
    // 所以我们只在openKeys数组长度大于0时调用onSelectBlock
    if (openKeys.length > 0) {
      const lastOpenKey = openKeys[openKeys.length - 1];
      onSelectBlock(lastOpenKey);
    }
  }

  // 初始化菜单
  useEffect(() => {
    setLoading(true)
    try {
      const menu = generateMenu(config)
      setFilteredMenu(menu)
    } catch (error) {
      console.error('Error generating menu:', error)
    } finally {
      setLoading(false)
    }
  }, [config, generateMenu])

  // 处理搜索过滤
  useEffect(() => {
    const menu = generateMenu(config)
    const filtered = filterMenu(menu, searchText)
    setFilteredMenu(filtered)
  }, [searchText, config, generateMenu])

  if (loading) {
    return <Spin size="small" style={{ margin: '20px' }} />
  }

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* 搜索框 */}
      <div style={{ padding: '12px' }}>
        <Search
          placeholder={t('configNavigator.searchConfigBlock')}
          value={searchText}
          onChange={(e) => handleSearch(e.target.value)}
          allowClear
          prefix={<SearchOutlined />}
          size="small"
        />
      </div>

      {/* 菜单 */}
      <div style={{ flex: 1, overflow: 'auto' }}>
        {filteredMenu.length > 0 ? (
          <Menu
            mode="inline"
            selectedKeys={activeBlock ? [activeBlock] : []}
            onClick={handleMenuClick}
            onSelect={handleMenuSelect}
            onOpenChange={handleMenuOpenChange}
            items={filteredMenu}
            style={{ border: 'none' }}
            inlineCollapsed={false}
          />
        ) : (
          <Empty description={t('configNavigator.noConfigData')} style={{ margin: '40px 0' }} />
        )}
      </div>
    </div>
  )
}

export default ConfigNavigator
