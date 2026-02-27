// src/utils/tokenManager.js

/**
 * Token management utility for handling JWT tokens
 */

// Token storage keys
const ACCESS_TOKEN_KEY = 'steadyDNS_access_token';
const REFRESH_TOKEN_KEY = 'steadyDNS_refresh_token';
const TOKEN_EXPIRES_AT_KEY = 'steadyDNS_token_expires_at';
const TOKEN_EXPIRES_IN_KEY = 'steadyDNS_token_expires_in';

// Token refresh debounce
let refreshPromise = null;

// Token refresh interval
let refreshInterval = null;

// Session timeout timer
let sessionTimeoutTimer = null;

// Session storage key for tracking session timeout status
const SESSION_TIMEOUT_RUNNING_KEY = 'steadyDNS_session_timeout_running';

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
  
  // Store original expiresIn value for session timeout reset
  sessionStorage.setItem(TOKEN_EXPIRES_IN_KEY, expiresIn.toString());
  
  // Start token refresh interval
  startTokenRefreshInterval();
  
  // Note: Session timeout is NOT started here
  // It should be started explicitly after login or by user activity
  // This ensures token refresh doesn't reset session timeout
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
 * Clear all tokens and user information
 */
export const clearTokens = () => {
  sessionStorage.removeItem(ACCESS_TOKEN_KEY);
  sessionStorage.removeItem(REFRESH_TOKEN_KEY);
  sessionStorage.removeItem(TOKEN_EXPIRES_AT_KEY);
  sessionStorage.removeItem(TOKEN_EXPIRES_IN_KEY);
  sessionStorage.removeItem('steadyDNS_last_activity');
  sessionStorage.removeItem('steadyDNS_user');
  // Clear the token refresh started flag so it can be started again on next login
  sessionStorage.removeItem('steadyDNS_token_refresh_started');
  // Clear session timeout running flag
  sessionStorage.removeItem(SESSION_TIMEOUT_RUNNING_KEY);
  
  // Clear intervals and timers
  clearTokenRefreshInterval();
  clearSessionTimeoutTimer();
};

/**
 * Start token refresh interval
 */
export const startTokenRefreshInterval = () => {
  // Clear existing interval if any
  clearTokenRefreshInterval();
  
  // Check token status every minute
  refreshInterval = setInterval(async () => {
    if (shouldRefreshToken()) {
      await refreshToken();
    }
  }, 60 * 1000); // Check every minute
};

/**
 * Clear token refresh interval
 */
export const clearTokenRefreshInterval = () => {
  if (refreshInterval) {
    clearInterval(refreshInterval);
    refreshInterval = null;
  }
};

/**
 * Start session timeout timer
 * @param {number} expiresIn - Token expiration time in seconds
 */
export const startSessionTimeoutTimer = (expiresIn) => {
  // Check if session timeout is already running using sessionStorage
  const isRunning = sessionStorage.getItem(SESSION_TIMEOUT_RUNNING_KEY);
  if (isRunning === 'true') {
    return;
  }
  
  // Clear existing timer if any
  clearSessionTimeoutTimer();
  
  // Set session timeout based on expiresIn
  sessionTimeoutTimer = setTimeout(() => {
    console.log('=== TokenManager: Session timeout triggered ===');
    // Session timed out, clear tokens and redirect to login
    clearTokens();
    window.location.href = '/login';
  }, expiresIn * 1000);
  
  // Mark session timeout as running in sessionStorage
  sessionStorage.setItem(SESSION_TIMEOUT_RUNNING_KEY, 'true');
};

/**
 * Clear session timeout timer
 */
export const clearSessionTimeoutTimer = () => {
  if (sessionTimeoutTimer) {
    clearTimeout(sessionTimeoutTimer);
    sessionTimeoutTimer = null;
    // Reset the running flag in sessionStorage
    sessionStorage.removeItem(SESSION_TIMEOUT_RUNNING_KEY);
  }
};

/**
 * Reset session timeout timer
 */
export const resetSessionTimeoutTimer = () => {
  const expiresInStr = sessionStorage.getItem(TOKEN_EXPIRES_IN_KEY);
  if (expiresInStr) {
    const expiresIn = parseInt(expiresInStr);
    
    // Reset the running flag in sessionStorage to allow starting a new timer
    sessionStorage.removeItem(SESSION_TIMEOUT_RUNNING_KEY);
    startSessionTimeoutTimer(expiresIn);
    
    // Record last activity time
    const now = Date.now();
    sessionStorage.setItem('steadyDNS_last_activity', now.toString());
  } else {
    // Try to get from expiresAt as fallback
    const expiresAtStr = sessionStorage.getItem(TOKEN_EXPIRES_AT_KEY);
    if (expiresAtStr) {
      const expiresAt = parseInt(expiresAtStr);
      const now = Date.now();
      const expiresIn = Math.max(0, Math.floor((expiresAt - now) / 1000));
      console.log('Fallback: Using remaining expiresIn:', expiresIn);
      startSessionTimeoutTimer(expiresIn);
    }
  }
};

/**
 * Refresh token using refresh token endpoint
 * @returns {Promise<string|null>} New access token or null if refresh failed
 */
export const refreshToken = async () => {
  // If there's already a refresh in progress, return the existing promise
  if (refreshPromise) {
    console.log('Token refresh: Using existing refresh promise');
    return refreshPromise;
  }
  
  const refreshTokenValue = getRefreshToken();
  if (!refreshTokenValue) {
    console.log('Token refresh: No refresh token available');
    clearTokens();
    return null;
  }
  
  console.log('Token refresh: Starting new token refresh process');
  
  // Create a new refresh promise
  refreshPromise = (async () => {
    let retries = 2;
    let lastError = null;
    
    while (retries >= 0) {
      try {
        console.log('Token refresh: Attempting to refresh token, retries left:', retries);
        const response = await fetch('/api/refresh-token', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({ refresh_token: refreshTokenValue })
        });
        
        console.log('Token refresh: Received response with status:', response.status);
        
        // Handle rate limit errors
        if (response.status === 429) {
          const errorData = await response.json();
          console.error('Token refresh rate limit exceeded:', errorData.message);
          // Don't clear tokens for rate limit errors, just return null
          return null;
        }
        
        if (!response.ok) {
          // For server errors, retry if we have retries left
          if (response.status >= 500 && retries > 0) {
            retries--;
            lastError = new Error(`Server error ${response.status}`);
            console.error('Token refresh server error, will retry:', lastError.message);
            // Wait for 1 second before retrying
            await new Promise(resolve => setTimeout(resolve, 1000));
            continue;
          }
          // Refresh failed, clear tokens and user info
          console.log('Token refresh failed, clearing tokens');
          clearTokens();
          // Clear user info from sessionStorage
          sessionStorage.removeItem('steadyDNS_user');
          return null;
        }
        
        const data = await response.json();
        if (data.success) {
          console.log('Token refresh successful, storing new tokens');
          // Store new tokens
          storeTokens(
            data.data.access_token,
            data.data.refresh_token,
            data.data.expires_in
          );
          return data.data.access_token;
        } else {
          // Refresh failed, clear tokens and user info
          console.log('Token refresh failed (unsuccessful response), clearing tokens');
          clearTokens();
          // Clear user info from sessionStorage
          sessionStorage.removeItem('steadyDNS_user');
          return null;
        }
      } catch (error) {
        console.error('Token refresh failed with error:', error);
        lastError = error;
        
        // For network errors, retry if we have retries left
        if (retries > 0) {
          retries--;
          console.log('Token refresh network error, will retry:', retries);
          // Wait for 1 second before retrying
          await new Promise(resolve => setTimeout(resolve, 1000));
          continue;
        }
        
        // Don't clear tokens for network errors, just return null
        return null;
      }
    }
    
    // If we exhausted all retries
    console.error('Token refresh exhausted all retries:', lastError);
    return null;
  })();
  
  // Ensure refreshPromise is reset to null after completion
  refreshPromise.finally(() => {
    console.log('Token refresh: Resetting refreshPromise to null');
    refreshPromise = null;
  });
  
  return refreshPromise;
};

/**
 * Get token status information
 * @returns {Object} Token status information
 */
export const getTokenStatus = () => {
  const accessToken = getAccessToken();
  const expiresAtStr = sessionStorage.getItem(TOKEN_EXPIRES_AT_KEY);
  const lastActivityStr = sessionStorage.getItem('steadyDNS_last_activity');
  
  if (!accessToken || !expiresAtStr) {
    return {
      isLoggedIn: false,
      accessToken: null,
      expiresAt: null,
      expiresIn: null,
      lastActivity: null,
      timeRemaining: null,
      shouldRefresh: false
    };
  }
  
  const expiresAt = parseInt(expiresAtStr);
  const lastActivity = lastActivityStr ? parseInt(lastActivityStr) : null;
  const now = Date.now();
  const expiresIn = (expiresAt - now) / 1000;
  const timeRemaining = expiresAt - now;
  
  return {
    isLoggedIn: true,
    accessToken,
    expiresAt,
    expiresIn,
    lastActivity,
    timeRemaining,
    shouldRefresh: timeRemaining > 0 && timeRemaining < 5 * 60 * 1000 // Refresh 5 minutes before expiration
  };
};

/**
 * Logout by calling logout endpoint and clearing tokens
 * @returns {Promise<boolean>} True if logout successful, false otherwise
 */
export const logout = async () => {
  const refreshToken = getRefreshToken();
  const accessToken = getAccessToken();
  
  try {
    if (refreshToken) {
      await fetch('/api/logout', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${accessToken}`
        },
        body: JSON.stringify({ refresh_token: refreshToken })
      });
    }
  } catch (error) {
    console.error('Logout request failed:', error);
  } finally {
    clearTokens();
  }
  
  return true;
};