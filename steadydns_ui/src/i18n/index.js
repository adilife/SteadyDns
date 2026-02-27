import zhCN from './zh-CN'
import enUS from './en-US'
import arSA from './ar-SA'

// Get saved language from localStorage or use default
const getSavedLanguage = () => {
  return localStorage.getItem('steadyDNS_language') || 'zh-CN'
}

// Save language to localStorage
const saveLanguage = (language) => {
  localStorage.setItem('steadyDNS_language', language)
}

// Translation function
const t = (key, lang = getSavedLanguage(), replacements = {}) => {
  // Split key into parts (e.g., 'login.title' -> ['login', 'title'])
  const keys = key.split('.')
  let result = lang === 'zh-CN' ? zhCN : lang === 'ar-SA' ? arSA : enUS
  
  // Traverse the language object to find the translation
  for (const k of keys) {
    if (result && result[k] !== undefined) {
      result = result[k]
    } else {
      return key // Return original key if translation not found
    }
  }
  
  // Handle replacements (e.g., 'Welcome, {{username}}' -> 'Welcome, John')
  if (typeof result === 'string' && Object.keys(replacements).length > 0) {
    return result.replace(/{{(.*?)}}/g, (match, placeholder) => {
      return replacements[placeholder] || match
    })
  }
  
  return result
}

// Language switch function
const switchLanguage = (language) => {
  saveLanguage(language)
  // You might want to emit an event here to notify components of language change
}

export {
  t,
  switchLanguage,
  getSavedLanguage,
  saveLanguage,
  zhCN,
  enUS,
  arSA
}