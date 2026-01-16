// src/utils/apiClient.js

import { message } from 'antd'
import { getAccessToken, refreshToken as refreshAuthToken, hasValidToken } from './tokenManager'

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
  }

  /**
   * Make an API request with error handling
   * @param {string} endpoint - API endpoint
   * @param {object} options - Fetch options
   * @param {boolean} retry - Whether to retry on 401 errors
   * @returns {Promise<any>} Response data
   */
  async request(endpoint, options = {}, retry = true) {
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
        const refreshed = await this.refreshToken()
        if (refreshed) {
          // Retry request with new token
          return this.request(endpoint, options, false)
        } else {
          // Token refresh failed, redirect to login
          window.location.href = '/login'
          throw new Error('登录已过期，请重新登录')
        }
      }
      
      // Handle other errors
      if (!response.ok) {
        const errorData = await response.json()
        const errorMessage = errorData.message || '请求失败'
        message.error(errorMessage)
        throw new Error(errorMessage)
      }
      
      return await response.json()
    } catch (error) {
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
}

// Export singleton instance
export const apiClient = new APIClient()
export default apiClient
