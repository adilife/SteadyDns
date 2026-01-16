# SteadyDNS API接口文档

## 1. 概述

### 1.1 API架构
SteadyDNS提供了一套完整的RESTful API，用于管理DNS服务器配置、监控系统状态和操作缓存。API基于HTTP协议，使用JSON格式进行数据交换。

### 1.2 认证方式
- **认证机制**：JWT (JSON Web Token)
- **令牌类型**：访问令牌(access token)和刷新令牌(refresh token)
- **认证流程**：
  1. 用户通过登录接口获取访问令牌和刷新令牌
  2. 访问受保护的API时，在请求头中携带访问令牌
  3. 访问令牌过期后，使用刷新令牌获取新的访问令牌

### 1.3 基础路径
- API基础路径：`/api`
- 所有API请求都必须以该路径为前缀

### 1.4 响应格式
所有API响应都使用统一的JSON格式：

#### 成功响应
```json
{
  "success": true,
  "data": {},
  "message": "操作成功"
}
```

#### 错误响应
```json
{
  "success": false,
  "error": "错误信息",
  "code": 400
}
```

### 1.5 状态码
- `200 OK`：请求成功
- `201 Created`：资源创建成功
- `400 Bad Request`：请求参数错误
- `401 Unauthorized`：未授权，令牌无效或过期
- `403 Forbidden`：禁止访问
- `404 Not Found`：资源不存在
- `500 Internal Server Error`：服务器内部错误

## 2. 认证接口

### 2.1 登录
- **路径**：`/api/login`
- **方法**：`POST`
- **功能**：用户登录并获取访问令牌和刷新令牌
- **请求体**：
  ```json
  {
    "username": "admin",
    "password": "password123"
  }
  ```
- **响应**：
  ```json
  {
    "success": true,
    "data": {
      "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
      "refresh_token": "1_1640995200_1640995200",
      "user": {
        "id": 1,
        "username": "admin",
        "email": "admin@example.com"
      },
      "expires_in": 1800
    },
    "message": "登录成功"
  }
  ```

### 2.2 刷新令牌
- **路径**：`/api/refresh-token`
- **方法**：`POST`
- **功能**：使用刷新令牌获取新的访问令牌和刷新令牌
- **请求体**：
  ```json
  {
    "refresh_token": "1_1640995200_1640995200"
  }
  ```
- **响应**：
  ```json
  {
    "success": true,
    "data": {
      "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
      "refresh_token": "1_1640996200_1640996200",
      "user": {
        "id": 1,
        "username": "admin",
        "email": "admin@example.com"
      },
      "expires_in": 1800
    },
    "message": "令牌刷新成功"
  }
  ```

### 2.3 登出
- **路径**：`/api/logout`
- **方法**：`POST`
- **功能**：用户登出，使刷新令牌失效
- **请求体**：
  ```json
  {
    "refresh_token": "1_1640995200_1640995200"
  }
  ```
- **响应**：
  ```json
  {
    "success": true,
    "data": null,
    "message": "登出成功"
  }
  ```

## 3. 转发组管理接口

### 3.1 获取所有转发组
- **路径**：`/api/forward-groups`
- **方法**：`GET`
- **功能**：获取所有转发组的列表
- **响应**：
  ```json
  {
    "success": true,
    "data": [
      {
        "id": 1,
        "name": "默认转发组",
        "description": "系统默认的DNS转发组",
        "created_at": "2023-01-01T00:00:00Z",
        "updated_at": "2023-01-01T00:00:00Z"
      }
    ],
    "message": "获取转发组列表成功"
  }
  ```

### 3.2 创建转发组
- **路径**：`/api/forward-groups`
- **方法**：`POST`
- **功能**：创建新的转发组
- **请求体**：
  ```json
  {
    "name": "国内DNS",
    "description": "国内DNS服务器组"
  }
  ```
- **响应**：
  ```json
  {
    "success": true,
    "data": {
      "id": 2,
      "name": "国内DNS",
      "description": "国内DNS服务器组",
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-01T00:00:00Z"
    },
    "message": "转发组创建成功"
  }
  ```

### 3.3 批量删除转发组
- **路径**：`/api/forward-groups?batch=true`
- **方法**：`DELETE`
- **功能**：批量删除多个转发组
- **请求体**：
  ```json
  [2, 3, 4]
  ```
- **响应**：
  ```json
  {
    "success": true,
    "data": null,
    "message": "成功删除 3 个转发组"
  }
  ```

### 3.4 获取单个转发组
- **路径**：`/api/forward-groups/{id}`
- **方法**：`GET`
- **功能**：根据ID获取单个转发组的详细信息
- **响应**：
  ```json
  {
    "success": true,
    "data": {
      "id": 1,
      "name": "默认转发组",
      "description": "系统默认的DNS转发组",
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-01T00:00:00Z"
    },
    "message": "获取转发组成功"
  }
  ```

### 3.5 更新转发组
- **路径**：`/api/forward-groups/{id}`
- **方法**：`PUT`
- **功能**：更新指定ID的转发组信息
- **请求体**：
  ```json
  {
    "name": "默认转发组",
    "description": "系统默认的DNS转发组（已更新）"
  }
  ```
- **响应**：
  ```json
  {
    "success": true,
    "data": {
      "id": 1,
      "name": "默认转发组",
      "description": "系统默认的DNS转发组（已更新）",
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-01T00:00:00Z"
    },
    "message": "转发组更新成功"
  }
  ```

### 3.6 删除转发组
- **路径**：`/api/forward-groups/{id}`
- **方法**：`DELETE`
- **功能**：删除指定ID的转发组（注意：ID=1的默认转发组不可删除）
- **响应**：
  ```json
  {
    "success": true,
    "data": null,
    "message": "转发组删除成功"
  }
  ```

## 4. 转发服务器管理接口

### 4.1 批量添加服务器
- **路径**：`/api/forward-servers?batch=true`
- **方法**：`POST`
- **功能**：批量添加多个DNS转发服务器
- **请求体**：
  ```json
  [
    {
      "group_id": 1,
      "address": "8.8.8.8",
      "port": 53,
      "protocol": "udp",
      "weight": 10,
      "enabled": true
    },
    {
      "group_id": 1,
      "address": "8.8.4.4",
      "port": 53,
      "protocol": "udp",
      "weight": 10,
      "enabled": true
    }
  ]
  ```
- **响应**：
  ```json
  {
    "success": true,
    "data": {
      "success_count": 2,
      "total_count": 2
    },
    "message": "成功添加 2 个服务器"
  }
  ```

### 4.2 批量删除服务器
- **路径**：`/api/forward-servers?batch=true`
- **方法**：`DELETE`
- **功能**：批量删除多个DNS转发服务器
- **请求体**：
  ```json
  [2, 3]
  ```
- **响应**：
  ```json
  {
    "success": true,
    "data": {
      "success_count": 2,
      "total_count": 2
    },
    "message": "成功删除 2 个服务器"
  }
  ```

### 4.3 获取单个服务器
- **路径**：`/api/forward-servers/{id}`
- **方法**：`GET`
- **功能**：根据ID获取单个DNS转发服务器的详细信息
- **响应**：
  ```json
  {
    "success": true,
    "data": {
      "id": 1,
      "group_id": 1,
      "address": "8.8.8.8",
      "port": 53,
      "protocol": "udp",
      "weight": 10,
      "enabled": true,
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-01T00:00:00Z"
    },
    "message": "获取服务器成功"
  }
  ```

### 4.4 检查服务器健康状态
- **路径**：`/api/forward-servers/{id}?health=true`
- **方法**：`GET`
- **功能**：检查指定DNS服务器的健康状态
- **响应**：
  ```json
  {
    "success": true,
    "data": {
      "server_id": 1,
      "address": "8.8.8.8",
      "port": 53,
      "is_healthy": true,
      "response_time": 15.2
    },
    "message": "服务器健康检查完成"
  }
  ```

### 4.5 更新服务器
- **路径**：`/api/forward-servers/{id}`
- **方法**：`PUT`
- **功能**：更新指定ID的DNS转发服务器信息
- **请求体**：
  ```json
  {
    "group_id": 1,
    "address": "8.8.8.8",
    "port": 53,
    "protocol": "udp",
    "weight": 15,
    "enabled": true
  }
  ```
- **响应**：
  ```json
  {
    "success": true,
    "data": {
      "id": 1,
      "group_id": 1,
      "address": "8.8.8.8",
      "port": 53,
      "protocol": "udp",
      "weight": 15,
      "enabled": true,
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-01T00:00:00Z"
    },
    "message": "服务器更新成功"
  }
  ```

### 4.6 删除服务器
- **路径**：`/api/forward-servers/{id}`
- **方法**：`DELETE`
- **功能**：删除指定ID的DNS转发服务器
- **响应**：
  ```json
  {
    "success": true,
    "data": null,
    "message": "服务器删除成功"
  }
  ```

## 5. 仪表盘接口

### 5.1 获取仪表盘摘要
- **路径**：`/api/dashboard/summary`
- **方法**：`GET`
- **功能**：获取系统概览数据，包括查询统计、服务器状态等
- **响应**：
  ```json
  {
    "success": true,
    "data": {
      "systemStats": {
        "totalQueries": 10000,
        "qps": 12.5,
        "cacheHitRate": 0,
        "systemHealth": 95,
        "activeServers": 2
      },
      "forwardServers": [
        {
          "id": 1,
          "address": "8.8.8.8",
          "qps": 12.5,
          "latency": 15.2,
          "status": "healthy"
        },
        {
          "id": 2,
          "address": "1.1.1.1",
          "qps": 10.2,
          "latency": 12.5,
          "status": "healthy"
        }
      ],
      "cacheStats": {
        "size": "1.2 GB",
        "maxSize": "2 GB",
        "hitRate": 85,
        "missRate": 15,
        "items": 150000
      },
      "systemResources": {
        "cpu": 45,
        "memory": 68,
        "disk": 32,
        "network": {
          "inbound": "12 MB/s",
          "outbound": "8 MB/s"
        }
      }
    },
    "message": "获取dashboard综合数据成功"
  }
  ```

### 5.2 获取仪表盘趋势
- **路径**：`/api/dashboard/trends`
- **方法**：`GET`
- **参数**：
  - `timeRange`：时间范围，可选值：`1h`、`6h`、`24h`、`7d`（默认：`1h`）
- **功能**：获取系统趋势数据，包括QPS趋势、延迟分布和资源使用情况
- **响应**：
  ```json
  {
    "success": true,
    "data": {
      "qpsTrend": [
        {"time": "12:00", "qps": 10.5},
        {"time": "12:01", "qps": 11.2},
        {"time": "12:02", "qps": 9.8}
      ],
      "latencyData": [
        {"range": "0-10ms", "count": 800},
        {"range": "10-50ms", "count": 150},
        {"range": "50ms+", "count": 50}
      ],
      "resourceUsage": [
        {"time": "12:00", "cpu": 40, "memory": 65, "disk": 30},
        {"time": "12:01", "cpu": 45, "memory": 68, "disk": 32},
        {"time": "12:02", "cpu": 42, "memory": 66, "disk": 31}
      ]
    },
    "message": "获取dashboard趋势数据成功"
  }
  ```

### 5.3 获取仪表盘排行榜
- **路径**：`/api/dashboard/top`
- **方法**：`GET`
- **参数**：
  - `limit`：返回数量限制，默认：`10`
- **功能**：获取热门域名和客户端的排行榜数据
- **响应**：
  ```json
  {
    "success": true,
    "data": {
      "topDomains": [
        {"rank": 1, "domain": "example.com", "queries": 1000, "percentage": 10.0},
        {"rank": 2, "domain": "google.com", "queries": 800, "percentage": 8.0},
        {"rank": 3, "domain": "facebook.com", "queries": 600, "percentage": 6.0}
      ],
      "topClients": [
        {"rank": 1, "ip": "192.168.1.100", "queries": 2000, "percentage": 20.0},
        {"rank": 2, "ip": "192.168.1.101", "queries": 1500, "percentage": 15.0},
        {"rank": 3, "ip": "192.168.1.102", "queries": 1000, "percentage": 10.0}
      ]
    },
    "message": "获取dashboard排行榜数据成功"
  }
  ```

## 6. 缓存接口

### 6.1 缓存接口（预留）
- **状态**：尚未实现
- **说明**：缓存相关的API接口将在后续版本中实现，包括缓存管理、统计等功能

## 7. 数据结构定义

### 7.1 用户相关

#### User
```go
type User struct {
    ID        uint      `json:"id"`
    Username  string    `json:"username"`
    Email     string    `json:"email"`
    Password  string    `json:"-"` // 不在JSON响应中包含密码
    CreatedAt time.Time `json:"created_at"`
}
```

#### LoginRequest
```go
type LoginRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
}
```

#### TokenResponse
```go
type TokenResponse struct {
    AccessToken  string      `json:"access_token"`
    RefreshToken string      `json:"refresh_token"`
    User         interface{} `json:"user"`
    ExpiresIn    int64       `json:"expires_in"` // 访问令牌过期时间（秒）
}
```

#### RefreshTokenRequest
```go
type RefreshTokenRequest struct {
    RefreshToken string `json:"refresh_token"`
}
```

#### LogoutRequest
```go
type LogoutRequest struct {
    RefreshToken string `json:"refresh_token"`
}
```

#### Claims
```go
type Claims struct {
    UserID   uint   `json:"user_id"`
    Username string `json:"username"`
    jwt.RegisteredClaims
}
```

### 7.2 转发组相关

#### ForwardGroup
```go
type ForwardGroup struct {
    ID          uint      `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

### 7.3 服务器相关

#### DNSServer
```go
type DNSServer struct {
    ID        uint      `json:"id"`
    GroupID   uint      `json:"group_id"`
    Address   string    `json:"address"`
    Port      int       `json:"port"`
    Protocol  string    `json:"protocol"`
    Weight    int       `json:"weight"`
    Enabled   bool      `json:"enabled"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### 7.4 仪表盘相关

#### SystemStats
```go
type SystemStats struct {
    TotalQueries  int     `json:"totalQueries"`
    QPS           float64 `json:"qps"`
    CacheHitRate  float64 `json:"cacheHitRate"`
    SystemHealth  int     `json:"systemHealth"`
    ActiveServers int     `json:"activeServers"`
}
```

#### ForwardServerStatus
```go
type ForwardServerStatus struct {
    ID      int     `json:"id"`
    Address string  `json:"address"`
    QPS     float64 `json:"qps"`
    Latency float64 `json:"latency"`
    Status  string  `json:"status"`
}
```

#### CacheStats
```go
type CacheStats struct {
    Size     string `json:"size"`
    MaxSize  string `json:"maxSize"`
    HitRate  int    `json:"hitRate"`
    MissRate int    `json:"missRate"`
    Items    int    `json:"items"`
}
```

#### SystemResources
```go
type SystemResources struct {
    CPU     int `json:"cpu"`
    Memory  int `json:"memory"`
    Disk    int `json:"disk"`
    Network struct {
        Inbound  string `json:"inbound"`
        Outbound string `json:"outbound"`
    } `json:"network"`
}
```

#### QPSTrend
```go
type QPSTrend struct {
    Time string  `json:"time"`
    QPS  float64 `json:"qps"`
}
```

#### LatencyData
```go
type LatencyData struct {
    Range string `json:"range"`
    Count int    `json:"count"`
}
```

#### ResourceUsage
```go
type ResourceUsage struct {
    Time   string `json:"time"`
    CPU    int    `json:"cpu"`
    Memory int    `json:"memory"`
    Disk   int    `json:"disk"`
}
```

#### TopDomain
```go
type TopDomain struct {
    Rank       int     `json:"rank"`
    Domain     string  `json:"domain"`
    Queries    int     `json:"queries"`
    Percentage float64 `json:"percentage"`
}
```

#### TopClient
```go
type TopClient struct {
    Rank       int     `json:"rank"`
    IP         string  `json:"ip"`
    Queries    int     `json:"queries"`
    Percentage float64 `json:"percentage"`
}
```

#### DashboardSummaryResponse
```go
type DashboardSummaryResponse struct {
    SystemStats     SystemStats           `json:"systemStats"`
    ForwardServers  []ForwardServerStatus `json:"forwardServers"`
    CacheStats      CacheStats            `json:"cacheStats"`
    SystemResources SystemResources       `json:"systemResources"`
}
```

#### DashboardTrendsResponse
```go
type DashboardTrendsResponse struct {
    QPSTrend      []QPSTrend      `json:"qpsTrend"`
    LatencyData   []LatencyData   `json:"latencyData"`
    ResourceUsage []ResourceUsage `json:"resourceUsage"`
}
```

#### DashboardTopResponse
```go
type DashboardTopResponse struct {
    TopDomains []TopDomain `json:"topDomains"`
    TopClients []TopClient `json:"topClients"`
}
```

## 8. 认证中间件

### 8.1 认证流程
1. 客户端通过登录接口获取JWT令牌
2. 客户端在请求受保护的API时，在请求头中携带令牌：
   ```
   Authorization: Bearer <access_token>
   ```
3. 服务器验证令牌的有效性
4. 令牌过期后，客户端使用刷新令牌获取新的访问令牌

### 8.2 令牌验证
- 验证令牌签名
- 检查令牌是否过期
- 验证令牌中的用户信息

## 9. 安全说明

### 9.1 注意事项
- 所有API请求都应使用HTTPS协议
- 访问令牌应妥善保管，避免泄露
- 刷新令牌应存储在安全的地方
- 定期更新密码和密钥

### 9.2 速率限制
- API请求可能会受到速率限制，以防止滥用
- 具体限制策略将根据部署环境进行配置

## 10. 版本控制

### 10.1 API版本
- 当前版本：v1
- API路径中不包含版本号，后续版本升级将通过配置或新路径实现

### 10.2 向后兼容
- 后续版本将尽量保持向后兼容性
- 重大变更将在文档中明确说明