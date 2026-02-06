# DNS权威区域管理 API 接口文档

## 1. 概述

本文档描述了DNS权威区域管理的API接口，用于前端开发人员与后端服务进行交互。这些API允许用户创建、查询、更新和删除DNS权威区域，以及管理区域的历史记录。

## 2. 基础信息

### 2.1 API前缀

所有API端点的前缀为：`/api/bind-zones/`

### 2.2 响应格式

所有API返回JSON格式响应，包含以下字段：

```json
{
  "success": true,  // 布尔值，表示请求是否成功
  "data": {}        // 响应数据，类型根据请求而定
}
```

错误响应格式：

```json
{
  "success": false,  // 布尔值，表示请求失败
  "error": "错误信息" // 字符串，描述错误原因
}
```

## 3. API端点

### 3.1 获取所有权威域

**功能**：获取所有已配置的DNS权威域列表

**请求方法**：GET

**请求路径**：`/api/bind-zones/`

**请求参数**：无

**响应数据**：
- 类型：数组
- 元素类型：`AuthZone`（权威域信息，详见数据结构说明）

**示例响应**：

```json
{
  "success": true,
  "data": [
    {
      "domain": "example.com",
      "type": "master",
      "file": "example.com.zone",
      "allow_query": "any",
      "soa": {...},  // SOA记录
      "ns": [...],   // NS记录列表
      "a": [...],    // A记录列表
      "aaaa": [...], // AAAA记录列表
      "cname": [...],// CNAME记录列表
      "mx": [...],   // MX记录列表
      "other": [...] // 其他记录列表
    },
    // 更多权威域...
  ]
}
```

### 3.2 获取单个权威域

**功能**：获取指定域名的权威域详细信息

**请求方法**：GET

**请求路径**：`/api/bind-zones/{domain}`

**路径参数**：
- `domain`：字符串，要查询的域名

**响应数据**：
- 类型：`AuthZone`（权威域信息，详见数据结构说明）

**示例响应**：

```json
{
  "success": true,
  "data": {
    "domain": "example.com",
    "type": "master",
    "file": "example.com.zone",
    "allow_query": "any",
    "soa": {
      "primary_ns": "ns1.example.com",
      "admin_email": "admin.example.com",
      "serial": "2023010101",
      "refresh": "3600",
      "retry": "1800",
      "expire": "604800",
      "minimum_ttl": "86400"
    },
    "ns": [
      {
        "name": "@",
        "value": "ns1.example.com"
      },
      {
        "name": "@",
        "value": "ns2.example.com"
      }
    ],
    "a": [
      {
        "name": "@",
        "value": "192.168.1.1"
      },
      {
        "name": "www",
        "value": "192.168.1.2"
      }
    ],
    "aaaa": [],
    "cname": [],
    "mx": [
      {
        "priority": 10,
        "value": "mail.example.com"
      }
    ],
    "other": []
  }
}
```

### 3.3 创建权威域

**功能**：创建新的DNS权威域

**请求方法**：POST

**请求路径**：`/api/bind-zones/`

**请求体**：
- 类型：`AuthZone`（权威域信息，详见数据结构说明）
- 必需字段：`domain`、`soa`、`ns`

**示例请求体**：

```json
{
  "domain": "example.com",
  "type": "master",
  "allow_query": "any",
  "soa": {
    "primary_ns": "ns1.example.com",
    "admin_email": "admin.example.com",
    "serial": "2023010101",
    "refresh": "3600",
    "retry": "1800",
    "expire": "604800",
    "minimum_ttl": "86400"
  },
  "ns": [
    {
      "name": "@",
      "value": "ns1.example.com"
    },
    {
      "name": "@",
      "value": "ns2.example.com"
    }
  ],
  "a": [
    {
      "name": "@",
      "value": "192.168.1.1"
    }
  ],
  "mx": [
    {
      "priority": 10,
      "value": "mail.example.com"
    }
  ]
}
```

**响应数据**：
- 类型：对象
- 字段：
  - `message`：字符串，成功消息
  - `domain`：字符串，创建的域名

**示例响应**：

```json
{
  "success": true,
  "data": {
    "message": "权威域创建成功",
    "domain": "example.com"
  }
}
```

### 3.4 更新权威域

**功能**：更新指定域名的权威域信息

**请求方法**：PUT

**请求路径**：`/api/bind-zones/{domain}`

**路径参数**：
- `domain`：字符串，要更新的域名

**请求体**：
- 类型：`AuthZone`（权威域信息，详见数据结构说明）
- 注意：请求体中的`domain`字段会被路径参数覆盖

**响应数据**：
- 类型：对象
- 字段：
  - `message`：字符串，成功消息
  - `domain`：字符串，更新的域名

**示例响应**：

```json
{
  "success": true,
  "data": {
    "message": "权威域更新成功",
    "domain": "example.com"
  }
}
```

### 3.5 删除权威域

**功能**：删除指定域名的权威域

**请求方法**：DELETE

**请求路径**：`/api/bind-zones/{domain}`

**路径参数**：
- `domain`：字符串，要删除的域名

**请求参数**：无

**响应数据**：
- 类型：对象
- 字段：
  - `message`：字符串，成功消息
  - `domain`：字符串，删除的域名

**示例响应**：

```json
{
  "success": true,
  "data": {
    "message": "权威域删除成功",
    "domain": "example.com"
  }
}
```

### 3.6 刷新权威域

**功能**：刷新BIND服务器配置，使权威域变更生效

**请求方法**：POST

**请求路径**：`/api/bind-zones/{domain}/reload`

**路径参数**：
- `domain`：字符串，要刷新的域名

**请求参数**：无

**响应数据**：
- 类型：对象
- 字段：
  - `message`：字符串，成功消息
  - `domain`：字符串，刷新的域名

**示例响应**：

```json
{
  "success": true,
  "data": {
    "message": "BIND服务器刷新成功",
    "domain": "example.com"
  }
}
```

### 3.7 获取操作历史

**功能**：获取权威域的操作历史记录

**请求方法**：GET

**请求路径**：`/api/bind-zones/history`

**请求参数**：无

**响应数据**：
- 类型：数组
- 元素类型：`IndexEntry`（历史记录条目，详见数据结构说明）

**示例响应**：

```json
{
  "success": true,
  "data": [
    {
      "RecordID": 1,
      "Operation": 0,  // 0: Create, 1: Update, 2: Delete, 3: Rollback
      "Domain": "example.com",
      "Timestamp": 1672531200,  // Unix时间戳
      "ExpiryTime": 1704067200  // Unix时间戳，记录过期时间
    },
    // 更多历史记录...
  ]
}
```

### 3.8 恢复历史记录

**功能**：恢复指定历史记录ID的权威域状态（注：该功能当前暂未实现完整逻辑）

**请求方法**：POST

**请求路径**：`/api/bind-zones/history/{historyID}`

**路径参数**：
- `historyID`：字符串，要恢复的历史记录ID

**请求参数**：无

**响应数据**：
- 类型：对象
- 字段：
  - `message`：字符串，提示信息
  - `history_id`：字符串，请求的历史记录ID

**示例响应**：

```json
{
  "success": true,
  "data": {
    "message": "历史记录恢复功能暂未实现",
    "history_id": "1"
  }
}
```

## 4. 数据结构

### 4.1 AuthZone（权威域信息）

| 字段名       | 类型              | 描述                                  |
|-------------|------------------|---------------------------------------|
| `domain`    | 字符串            | 域名                                  |
| `type`      | 字符串            | 区域类型（如：master）                |
| `file`      | 字符串            | 区域文件路径                          |
| `allow_query` | 字符串          | 允许查询的IP范围（如：any）           |
| `soa`       | `SOARecord`       | SOA记录                               |
| `ns`        | 数组（`NSRecord`）| NS记录列表                            |
| `a`         | 数组（`ARecord`） | A记录列表                             |
| `aaaa`      | 数组（`AAAARecord`）| AAAA记录列表                          |
| `cname`     | 数组（`CNAMERecord`）| CNAME记录列表                         |
| `mx`        | 数组（`MXRecord`） | MX记录列表                             |
| `other`     | 数组（`OtherRecord`）| 其他类型记录列表                      |

### 4.2 SOARecord（SOA记录）

| 字段名          | 类型    | 描述                                  |
|----------------|---------|---------------------------------------|
| `primary_ns`   | 字符串  | 主域名服务器                          |
| `admin_email`  | 字符串  | 管理员邮箱                            |
| `serial`       | 字符串  | 序列号                                |
| `refresh`      | 字符串  | 刷新时间（秒）                        |
| `retry`        | 字符串  | 重试时间（秒）                        |
| `expire`       | 字符串  | 过期时间（秒）                        |
| `minimum_ttl`  | 字符串  | 最小TTL值（秒）                       |

### 4.3 NSRecord（NS记录）

| 字段名    | 类型    | 描述                                  |
|----------|---------|---------------------------------------|
| `name`   | 字符串  | 记录名称（如：@ 表示域名本身）        |
| `value`  | 字符串  | 域名服务器地址                        |

### 4.4 ARecord（A记录）

| 字段名    | 类型    | 描述                                  |
|----------|---------|---------------------------------------|
| `name`   | 字符串  | 记录名称（如：www）                   |
| `value`  | 字符串  | IPv4地址                              |

### 4.5 AAAARecord（AAAA记录）

| 字段名    | 类型    | 描述                                  |
|----------|---------|---------------------------------------|
| `name`   | 字符串  | 记录名称                              |
| `value`  | 字符串  | IPv6地址                              |

### 4.6 CNAMERecord（CNAME记录）

| 字段名    | 类型    | 描述                                  |
|----------|---------|---------------------------------------|
| `name`   | 字符串  | 别名记录名称                          |
| `value`  | 字符串  | 目标域名                              |

### 4.7 MXRecord（MX记录）

| 字段名      | 类型    | 描述                                  |
|------------|---------|---------------------------------------|
| `priority` | 整数    | 优先级（数值越小，优先级越高）        |
| `value`    | 字符串  | 邮件服务器地址                        |

### 4.8 OtherRecord（其他记录）

| 字段名    | 类型    | 描述                                  |
|----------|---------|---------------------------------------|
| `name`   | 字符串  | 记录名称                              |
| `type`   | 字符串  | 记录类型（如：TXT、SRV等）            |
| `value`  | 字符串  | 记录值                                |

### 4.9 IndexEntry（历史记录条目）

| 字段名        | 类型    | 描述                                  |
|--------------|---------|---------------------------------------|
| `RecordID`   | 整数    | 记录ID                                |
| `Operation`  | 整数    | 操作类型（0: Create, 1: Update, 2: Delete, 3: Rollback） |
| `Domain`     | 字符串  | 操作的域名                            |
| `Timestamp`  | 整数    | 操作时间戳（Unix时间）                |
| `ExpiryTime` | 整数    | 记录过期时间戳（Unix时间）            |

## 5. 错误处理

API可能返回的常见错误类型：

| HTTP状态码 | 错误类型          | 描述                                  |
|-----------|------------------|---------------------------------------|
| 400       | Bad Request      | 请求参数无效或格式错误                |
| 404       | Not Found        | 请求的资源不存在                      |
| 500       | Internal Server Error | 服务器内部错误，如文件操作失败等      |

所有错误响应都会包含详细的错误信息，前端可以根据`success`字段和`error`字段进行错误处理和用户提示。

## 6. 注意事项

1. 所有API请求都需要进行身份验证（具体验证方式根据项目配置而定）
2. 创建和更新权威域时，系统会自动验证配置的合法性
3. 操作成功后，系统会自动刷新BIND服务器配置
4. 系统会为每个操作创建备份，支持历史记录查询和恢复
5. 操作历史记录有过期时间，过期后会被自动清理

## 7. 回退保护功能

系统实现了回退保护功能，当执行回退操作时，会自动备份当前的`history.record`文件，确保回退操作本身可完全被回退。

### 7.1 回退保护文件命名规则

备份文件命名格式：`history.record.rollback.{transactionID}`

- `transactionID`：回退操作的事务ID，用于关联回退操作和对应的备份文件

### 7.2 回退保护配置

回退保护的记录数量可以通过配置文件中的`ROLLBACK_PROTECTION_MAX_RECORDS`参数进行配置，默认值为10，可在5-20之间调整。当备份文件数量超过配置的最大值时，最旧的文件会被自动删除。

### 7.3 回退回退操作

系统支持回退回退操作，即恢复到回退前的状态。前端可以通过调用相应的API来实现这一功能。

## 8. 版本信息

- API版本：1.0
- 文档更新时间：2026-01-22
- 支持的BIND版本：兼容主流BIND 9.x版本
