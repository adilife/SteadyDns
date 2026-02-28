import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

import zhCN from './zh-CN';
import enUS from './en-US';
import arSA from './ar-SA';

/**
 * i18next初始化配置
 * 支持语言检测、持久化和三种语言（zh-CN、en-US、ar-SA）
 */
i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      'zh-CN': { translation: zhCN },
      'en-US': { translation: enUS },
      'ar-SA': { translation: arSA }
    },
    fallbackLng: 'en-US',
    supportedLngs: ['zh-CN', 'en-US', 'ar-SA'],
    detection: {
      order: ['localStorage', 'navigator'],
      lookupLocalStorage: 'steadyDNS_language',
      caches: ['localStorage']
    },
    interpolation: {
      escapeValue: false
    }
  });

export default i18n;
