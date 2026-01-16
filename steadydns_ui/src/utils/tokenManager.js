// src/utils/tokenManager.js

/**
 * Token management utility for handling JWT tokens
 */

// Token storage keys
const ACCESS_TOKEN_KEY = 'steadyDNS_access_token';
const REFRESH_TOKEN_KEY = 'steadyDNS_refresh_token';
const TOKEN_EXPIRES_AT_KEY = 'steadyDNS_token_expires_at';

// Token refresh debounce
let refreshPromise = null;

/**
 * Store tokens and expiration time
 * @param {string} accessToken - Access token from login response
 * @param {string} refreshToken - Refresh token from login response
 * @param {number} expiresIn - Token expiration time in seconds
 */
export const storeTokens = (accessToken, refreshToken, expiresIn) => {
  // Store access token in sessionStorage (more secure than localStorage)
  sessionStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
  sessionStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
  
  // Calculate and store expiration timestamp
  const expiresAt = Date.now() + (expiresIn * 1000);
  sessionStorage.setItem(TOKEN_EXPIRES_AT_KEY, expiresAt.toString());
};

/**
 * Get access token
 * @returns {string|null} Access token or null if not found
 */
export const getAccessToken = () => {
  return sessionStorage.getItem(ACCESS_TOKEN_KEY);
};

/**
 * Get refresh token
 * @returns {string|null} Refresh token or null if not found
 */
export const getRefreshToken = () => {
  return sessionStorage.getItem(REFRESH_TOKEN_KEY);
};

/**
 * Check if token is expired or about to expire
 * @param {number} bufferTime - Time in milliseconds to refresh before expiration
 * @returns {boolean} True if token needs refresh, false otherwise
 */
export const shouldRefreshToken = (bufferTime = 5 * 60 * 1000) => {
  const expiresAtStr = sessionStorage.getItem(TOKEN_EXPIRES_AT_KEY);
  if (!expiresAtStr) return true;
  
  const expiresAt = parseInt(expiresAtStr);
  const now = Date.now();
  
  // Return true if token is expired or will expire within buffer time
  return (expiresAt - now) < bufferTime;
};

/**
 * Check if there's a valid access token
 * @returns {boolean} True if there's a valid access token, false otherwise
 */
export const hasValidToken = () => {
  const accessToken = getAccessToken();
  const expiresAtStr = sessionStorage.getItem(TOKEN_EXPIRES_AT_KEY);
  
  if (!accessToken || !expiresAtStr) {
    return false;
  }
  
  const expiresAt = parseInt(expiresAtStr);
  const now = Date.now();
  
  // Return true if token is not expired
  return now < expiresAt;
};

/**
 * Clear all tokens
 */
export const clearTokens = () => {
  sessionStorage.removeItem(ACCESS_TOKEN_KEY);
  sessionStorage.removeItem(REFRESH_TOKEN_KEY);
  sessionStorage.removeItem(TOKEN_EXPIRES_AT_KEY);
};

/**
 * Refresh token using refresh token endpoint
 * @returns {Promise<string|null>} New access token or null if refresh failed
 */
export const refreshToken = async () => {
  // If there's already a refresh in progress, return the existing promise
  if (refreshPromise) {
    return refreshPromise;
  }
  
  const refreshTokenValue = getRefreshToken();
  if (!refreshTokenValue) {
    clearTokens();
    return null;
  }
  
  // Create a new refresh promise
  refreshPromise = (async () => {
    try {
      const response = await fetch('/api/refresh-token', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ refresh_token: refreshTokenValue })
      });
      
      // Handle rate limit errors
      if (response.status === 429) {
        const errorData = await response.json();
        console.error('Token refresh rate limit exceeded:', errorData.message);
        // Don't clear tokens for rate limit errors, just return null
        return null;
      }
      
      if (!response.ok) {
        // Refresh failed, clear tokens
        clearTokens();
        return null;
      }
      
      const data = await response.json();
      if (data.success) {
        // Store new tokens
        storeTokens(
          data.data.access_token,
          data.data.refresh_token,
          data.data.expires_in
        );
        return data.data.access_token;
      } else {
        // Refresh failed, clear tokens
        clearTokens();
        return null;
      }
    } catch (error) {
      console.error('Token refresh failed:', error);
      // Don't clear tokens for network errors, just return null
      return null;
    } finally {
      // Reset refresh promise after completion
      refreshPromise = null;
    }
  })();
  
  return refreshPromise;
};

/**
 * Logout by calling logout endpoint and clearing tokens
 * @returns {Promise<boolean>} True if logout successful, false otherwise
 */
export const logout = async () => {
  const refreshToken = getRefreshToken();
  
  try {
    if (refreshToken) {
      await fetch('/api/logout', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ refresh_token: refreshToken })
      });
    }
  } catch (error) {
    console.error('Logout request failed:', error);
  } finally {
    // Always clear tokens regardless of logout request result
    clearTokens();
  }
  
  return true;
};