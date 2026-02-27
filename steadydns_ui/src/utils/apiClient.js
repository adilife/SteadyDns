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

// src/utils/apiClient.js

import { message } from 'antd'
import { getAccessToken, refreshToken as refreshAuthToken, hasValidToken, clearTokens } from './tokenManager'

/**
 * API Client for handling API requests with rate limit error management
 */
class APIClient {
  constructor() {
    this.baseURL = '/api'
    this.loginAttempts = 0
    this.maxLoginAttempts = 5
    this.loginCooldownTime = 60000 // 1 minute
    this.batchSize = 5
    this.batchDelay = 1000 // 1 second between batches
    this.isHandling401 = false // Flag to prevent duplicate 401 handling
  }

  /**
   * Make an API request with error handling
   * @param {string} endpoint - API endpoint
   * @param {object} options - Fetch options
   * @param {boolean} retry - Whether to retry on 401 errors
   * @returns {Promise<any>} Response data
   */
  async request(endpoint, options = {}, retry = true) {
    // Check if already handling 401 error to prevent duplicate processing
    if (this.isHandling401) {
      return
    }
    
    const url = `${this.baseURL}${endpoint}`
    
    // Default options
    const defaultOptions = {
      headers: {
        'Content-Type': 'application/json'
      }
    }
    
    // Add authentication token if available
    if (hasValidToken()) {
      defaultOptions.headers['Authorization'] = `Bearer ${getAccessToken()}`
    }
    
    // Merge options
    const fetchOptions = {
      ...defaultOptions,
      ...options,
      headers: {
        ...defaultOptions.headers,
        ...options.headers
      }
    }
    
    try {
      const response = await fetch(url, fetchOptions)
      
      // Handle rate limit errors
      if (response.status === 429) {
        const errorData = await response.json()
        const errorMessage = errorData.message || '请求过于频繁，请稍后再试'
        message.error(errorMessage)
        throw new Error(errorMessage)
      }
      
      // Handle token expiration
      if (response.status === 401 && retry) {
        this.isHandling401 = true
        try {
          const refreshed = await this.refreshToken()
          if (refreshed) {
            // Retry request with new token
            return this.request(endpoint, options, false)
          } else {
            // Token refresh failed, clear tokens and redirect to login
            clearTokens()
            // Clear user info from sessionStorage
            sessionStorage.removeItem('steadyDNS_user')
            // Redirect to login page
            window.location.href = '/login'
            return
          }
        } finally {
          this.isHandling401 = false
        }
      }
      
      // Handle other errors
      if (!response.ok) {
        let errorMessage = '请求失败'
        try {
          const errorData = await response.json()
          errorMessage = errorData.error || errorData.message || errorMessage
        } catch (parseError) {
          // Failed to parse error response
          console.error('Failed to parse error response:', parseError)
        }
        message.error(errorMessage)
        throw new Error(errorMessage)
      }
      
      return await response.json()
    } catch (error) {
      // Skip error handling if we're already handling 401
      if (this.isHandling401) {
        return
      }
      
      // Handle network errors
      if (error.name === 'TypeError' && error.message.includes('Failed to fetch')) {
        const networkErrorMsg = '网络连接失败，请检查您的网络连接'
        message.error(networkErrorMsg)
        throw new Error(networkErrorMsg)
      }
      
      console.error('API request failed:', error)
      throw error
    }
  }

  /**
   * Refresh authentication token
   * @returns {Promise<boolean>} Whether token refresh was successful
   */
  async refreshToken() {
    try {
      const newToken = await refreshAuthToken()
      return !!newToken
    } catch (error) {
      console.error('Token refresh failed:', error)
      return false
    }
  }

  /**
   * Handle login attempt with rate limit protection
   * @param {string} username - User username
   * @param {string} password - User password
   * @returns {Promise<any>} Login response
   */
  async login(username, password) {
    // Check if login attempts exceeded
    if (this.loginAttempts >= this.maxLoginAttempts) {
      const errorMessage = '登录尝试次数过多，请1分钟后再试'
      message.error(errorMessage)
      throw new Error(errorMessage)
    }
    
    this.loginAttempts++
    
    try {
      const response = await this.request('/login', {
        method: 'POST',
        body: JSON.stringify({ username, password })
      })
      
      // Reset login attempts on success
      this.loginAttempts = 0
      return response
    } catch (error) {
      // Set cooldown if max attempts reached
      if (this.loginAttempts >= this.maxLoginAttempts) {
        setTimeout(() => {
          this.loginAttempts = 0
        }, this.loginCooldownTime)
      }
      throw error
    }
  }

  /**
   * Batch operation with rate limit awareness
   * @param {string} endpoint - API endpoint
   * @param {Array} items - Items to process in batches
   * @param {string} method - HTTP method
   * @returns {Promise<Array>} Batch operation results
   */
  async batchOperation(endpoint, items, method = 'POST') {
    const batches = []
    const results = []
    
    // Create batches
    for (let i = 0; i < items.length; i += this.batchSize) {
      batches.push(items.slice(i, i + this.batchSize))
    }
    
    // Process batches with delay
    for (const batch of batches) {
      const response = await this.request(endpoint, {
        method,
        body: JSON.stringify(batch)
      })
      results.push(response)
      
      // Add delay between batches
      if (batches.indexOf(batch) < batches.length - 1) {
        await new Promise(resolve => setTimeout(resolve, this.batchDelay))
      }
    }
    
    return results
  }

  /**
   * Get request
   * @param {string} endpoint - API endpoint
   * @param {object} options - Fetch options
   * @returns {Promise<any>} Response data
   */
  async get(endpoint, options = {}) {
    return this.request(endpoint, {
      ...options,
      method: 'GET'
    })
  }

  /**
   * Post request
   * @param {string} endpoint - API endpoint
   * @param {object} data - Request data
   * @param {object} options - Fetch options
   * @returns {Promise<any>} Response data
   */
  async post(endpoint, data, options = {}) {
    return this.request(endpoint, {
      ...options,
      method: 'POST',
      body: JSON.stringify(data)
    })
  }

  /**
   * Put request
   * @param {string} endpoint - API endpoint
   * @param {object} data - Request data
   * @param {object} options - Fetch options
   * @returns {Promise<any>} Response data
   */
  async put(endpoint, data, options = {}) {
    return this.request(endpoint, {
      ...options,
      method: 'PUT',
      body: JSON.stringify(data)
    })
  }

  /**
   * Delete request
   * @param {string} endpoint - API endpoint
   * @param {object} options - Fetch options
   * @returns {Promise<any>} Response data
   */
  async delete(endpoint, options = {}) {
    return this.request(endpoint, {
      ...options,
      method: 'DELETE'
    })
  }

  /**
   * Test domain match with forward groups
   * @param {string} domain - Domain to test
   * @returns {Promise<any>} Matching forward group data
   */
  async testDomainMatch(domain) {
    return this.request(`/forward-groups/test-domain-match?domain=${encodeURIComponent(domain)}`, {
      method: 'GET'
    })
  }

  /**
   * Get server status
   * @returns {Promise<any>} Server status data
   */
  async getServerStatus() {
    return this.request('/server/status', {
      method: 'GET'
    })
  }

  /**
   * Control server (start/stop/restart)
   * @param {string} action - Action to perform (start/stop/restart)
   * @param {string} serverType - Server type (sdnsd/httpd)
   * @returns {Promise<any>} Response data
   */
  async controlServer(action, serverType = 'sdnsd') {
    return this.request(`/server/${serverType}/${action}`, {
      method: 'POST'
    })
  }

  /**
   * Reload forward groups
   * @returns {Promise<any>} Response data
   */
  async reloadForwardGroups() {
    return this.request('/server/reload-forward-groups', {
      method: 'POST'
    })
  }

  /**
   * Set log level
   * @param {string} level - Log level
   * @returns {Promise<any>} Response data
   */
  async setLogLevel(level) {
    return this.request('/server/logging/level', {
      method: 'POST',
      body: JSON.stringify({ level })
    })
  }

  /**
   * Set log levels for API and DNS
   * @param {object} levels - Log levels object with api_log_level and dns_log_level
   * @returns {Promise<any>} Response data
   */
  async setLogLevels(levels) {
    return this.request('/server/logging/level', {
      method: 'POST',
      body: JSON.stringify(levels)
    })
  }

  /**
   * Get cache stats
   * @returns {Promise<any>} Cache stats data
   */
  async getCacheStats() {
    return this.request('/cache/stats', {
      method: 'GET'
    })
  }

  /**
   * Clear cache
   * @param {string} domain - Domain to clear cache for (optional)
   * @returns {Promise<any>} Response data
   */
  async clearCache(domain = null) {
    if (domain) {
      return this.request(`/cache/clear/${domain}`, {
        method: 'POST'
      })
    }
    return this.request('/cache/clear', {
      method: 'POST'
    })
  }

  /**
   * Get configuration
   * @param {string} section - Config section (optional)
   * @param {string} key - Config key (optional)
   * @returns {Promise<any>} Config data
   */
  async getConfig(section = null, key = null) {
    if (section && key) {
      return this.request(`/config/${section}/${key}`, {
        method: 'GET'
      })
    } else if (section) {
      return this.request(`/config/${section}`, {
        method: 'GET'
      })
    }
    return this.request('/config', {
      method: 'GET'
    })
  }

  /**
   * Update configuration
   * @param {string} section - Config section
   * @param {string} key - Config key
   * @param {string} value - Config value
   * @param {string} user - User making the change
   * @returns {Promise<any>} Response data
   */
  async updateConfig(section, key, value, user) {
    return this.request(`/config/${section}/${key}`, {
      method: 'PUT',
      body: JSON.stringify({ value, user })
    })
  }

  /**
   * Reload configuration
   * @returns {Promise<any>} Response data
   */
  async reloadConfig() {
    return this.request('/config/reload', {
      method: 'POST'
    })
  }

  /**
   * Backup configuration
   * @param {string} comment - Backup comment
   * @param {string} user - User making the backup
   * @returns {Promise<any>} Response data
   */
  async backupConfig(comment, user) {
    return this.request('/config/backup', {
      method: 'POST',
      body: JSON.stringify({ comment, user })
    })
  }

  /**
   * Restore configuration
   * @param {string} backupFile - Backup file name
   * @param {string} user - User making the restore
   * @returns {Promise<any>} Response data
   */
  async restoreConfig(backupFile, user) {
    return this.request('/config/restore', {
      method: 'POST',
      body: JSON.stringify({ backup_file: backupFile, user })
    })
  }

  /**
   * Get configuration history
   * @param {number} limit - Limit number of records
   * @returns {Promise<any>} Config history data
   */
  async getConfigHistory(limit = 10) {
    return this.request(`/config/history?limit=${limit}`, {
      method: 'GET'
    })
  }

  /**
   * Reset configuration
   * @param {string} user - User making the reset
   * @returns {Promise<any>} Response data
   */
  async resetConfig(user) {
    return this.request('/config/reset', {
      method: 'POST',
      body: JSON.stringify({ user })
    })
  }

  /**
   * Get environment variables
   * @returns {Promise<any>} Environment variables data
   */
  async getEnvVars() {
    return this.request('/config/env', {
      method: 'GET'
    })
  }

  /**
   * Set environment variable
   * @param {string} key - Environment variable key
   * @param {string} value - Environment variable value
   * @param {string} user - User making the change
   * @returns {Promise<any>} Response data
   */
  async setEnvVar(key, value, user) {
    return this.request('/config/env', {
      method: 'POST',
      body: JSON.stringify({ key, value, user })
    })
  }

  /**
   * Validate configuration
   * @returns {Promise<any>} Response data
   */
  async validateConfig() {
    return this.request('/config/validate', {
      method: 'POST'
    })
  }

  /**
   * Get BIND server status
   * @returns {Promise<any>} BIND server status data
   */
  async getBindServerStatus() {
    return this.request('/bind-server/status', {
      method: 'GET'
    })
  }

  /**
   * Control BIND server
   * @param {string} action - Action to perform (start/stop/restart/reload)
   * @returns {Promise<any>} Response data
   */
  async controlBindServer(action) {
    return this.request(`/bind-server/${action}`, {
      method: 'POST'
    })
  }

  /**
   * Get BIND server stats
   * @returns {Promise<any>} BIND server stats data
   */
  async getBindServerStats() {
    return this.request('/bind-server/stats', {
      method: 'GET'
    })
  }

  /**
   * Check BIND server health
   * @returns {Promise<any>} BIND server health data
   */
  async checkBindServerHealth() {
    return this.request('/bind-server/health', {
      method: 'GET'
    })
  }

  /**
   * Validate BIND configuration
   * @returns {Promise<any>} Response data
   */
  async validateBindConfig() {
    return this.request('/bind-server/validate', {
      method: 'POST'
    })
  }

  /**
   * Get BIND configuration
   * @returns {Promise<any>} BIND configuration data
   */
  async getBindConfig() {
    return this.request('/bind-server/config', {
      method: 'GET'
    })
  }



  /**
   * Get BIND named.conf parsed structure
   * @returns {Promise<any>} Parsed configuration structure
   */
  async getBindNamedConfParse() {
    return this.request('/bind-server/named-conf/parse', {
      method: 'GET'
    })
  }

  /**
   * Get BIND named.conf raw content
   * @returns {Promise<any>} Raw configuration content
   */
  async getBindNamedConfContent() {
    return this.request('/bind-server/named-conf/content', {
      method: 'GET'
    })
  }

  /**
   * Update BIND named.conf configuration
   * @param {object} config - Configuration content or structure
   * @returns {Promise<any>} Update result with diff
   */
  async updateBindNamedConf(config) {
    return this.request('/bind-server/named-conf', {
      method: 'PUT',
      body: JSON.stringify(config)
    })
  }

  /**
   * Validate BIND named.conf configuration
   * @param {object} config - Configuration content
   * @returns {Promise<any>} Validation result
   */
  async validateBindNamedConf(config) {
    return this.request('/bind-server/named-conf/validate', {
      method: 'POST',
      body: JSON.stringify(config)
    })
  }

  /**
   * Get BIND named.conf configuration diff
   * @param {object} newConfig - New configuration content
   * @returns {Promise<any>} Configuration diff
   */
  async getBindNamedConfDiff(newConfig) {
    return this.request('/bind-server/named-conf/diff', {
      method: 'POST',
      body: JSON.stringify(newConfig)
    })
  }

  /**
   * Get BIND named.conf backups
   * @returns {Promise<any>} List of backup files
   */
  async getBindNamedConfBackups() {
    return this.request('/bind-server/named-conf/backups', {
      method: 'GET'
    })
  }

  /**
   * Restore BIND named.conf from backup
   * @param {string} backupPath - Path to backup file
   * @returns {Promise<any>} Restore result
   */
  async restoreBindNamedConfBackup(backupPath) {
    return this.request('/bind-server/named-conf/restore', {
      method: 'POST',
      body: JSON.stringify({ backupPath })
    })
  }

  /**
   * Delete BIND named.conf backup
   * @param {string} backupId - Backup file name or ID
   * @returns {Promise<any>} Delete result
   */
  async deleteBindNamedConfBackup(backupId) {
    return this.request(`/bind-server/named-conf/backups/${backupId}`, {
      method: 'DELETE'
    })
  }

  /**
   * Get BIND zones operation history
   * @returns {Promise<any>} Operation history records
   */
  async getBindZonesHistory() {
    return this.request('/bind-zones/history', {
      method: 'GET'
    })
  }

  /**
   * Restore BIND zone from history
   * @param {number} historyId - History record ID
   * @returns {Promise<any>} Restore result
   */
  async restoreBindZoneFromHistory(historyId) {
    return this.request(`/bind-zones/history/${historyId}`, {
      method: 'POST'
    })
  }

  /**
   * Get health check status
   * @returns {Promise<any>} Health check data
   */
  async getHealthStatus() {
    return this.request('/health', {
      method: 'GET'
    })
  }

  /**
   * Get Dashboard summary data
   * @returns {Promise<any>} Dashboard summary data
   */
  async getDashboardSummary() {
    return this.request('/dashboard/summary', {
      method: 'GET'
    })
  }

  /**
   * Get Dashboard trends data
   * @param {string} type - Data type (all, qps, resource)
   * @param {string} timeRange - Time range (1h, 6h, 24h, 7d)
   * @param {number} points - Number of data points
   * @returns {Promise<any>} Dashboard trends data
   */
  async getDashboardTrends(type = 'all', timeRange = '1h', points = 12) {
    return this.request(`/dashboard/trends?type=${type}&timeRange=${timeRange}&points=${points}`, {
      method: 'GET'
    })
  }

  /**
   * Get Dashboard top data
   * @param {number} limit - Limit number of records
   * @returns {Promise<any>} Dashboard top data
   */
  async getDashboardTop(limit = 10) {
    return this.request(`/dashboard/top?limit=${limit}`, {
      method: 'GET'
    })
  }

  /**
   * Get plugins status
   * @returns {Promise<any>} Plugins status data
   */
  async getPluginsStatus() {
    return this.request('/plugins/status', {
      method: 'GET'
    })
  }

  // ==================== 用户管理相关API ====================

  /**
   * 获取用户列表
   * @param {number} page - 页码，默认为1
   * @param {number} pageSize - 每页数量，默认为10
   * @returns {Promise<any>} 用户列表数据，包含users、total、page、pageSize
   */
  async getUsers(page = 1, pageSize = 10) {
    return this.get(`/users?page=${page}&pageSize=${pageSize}`)
  }

  /**
   * 创建用户
   * @param {object} userData - 用户数据，包含username、email、password
   * @returns {Promise<any>} 创建结果，包含id、username、email和message
   */
  async createUser(userData) {
    return this.post('/users', userData)
  }

  /**
   * 更新用户信息
   * @param {number} id - 用户ID
   * @param {object} userData - 用户数据，包含username、email
   * @returns {Promise<any>} 更新结果，包含id、username、email和message
   */
  async updateUser(id, userData) {
    return this.put(`/users/${id}`, userData)
  }

  /**
   * 删除用户
   * @param {number} id - 用户ID
   * @returns {Promise<any>} 删除结果，包含message
   */
  async deleteUser(id) {
    return this.delete(`/users/${id}`)
  }

  /**
   * 修改用户密码
   * @param {number} id - 用户ID
   * @param {object} passwordData - 密码数据，包含old_password和new_password
   * @returns {Promise<any>} 修改结果，包含message
   */
  async changePassword(id, passwordData) {
    return this.put(`/users/${id}/password`, passwordData)
  }
}

// Export singleton instance
export const apiClient = new APIClient()
export default apiClient
