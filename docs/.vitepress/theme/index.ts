import DefaultTheme from 'vitepress/theme'
import './custom.css'

const languagePreferenceKey = 'ua3f-docs-language'

function normalizePath(pathname: string) {
  return pathname.replace(/\/index\.html$/, '/') || '/'
}

function detectBrowserLanguage() {
  const languages = navigator.languages?.length ? navigator.languages : [navigator.language]
  return languages.some((language) => language.toLowerCase().startsWith('zh')) ? 'zh' : 'en'
}

function rememberManualLanguageChoice(event: MouseEvent) {
  const link = (event.target as Element | null)?.closest?.('a')

  if (!(link instanceof HTMLAnchorElement) || link.origin !== window.location.origin) {
    return
  }

  const path = normalizePath(link.pathname)

  if (path === '/' || path.startsWith('/zh/')) {
    localStorage.setItem(languagePreferenceKey, path.startsWith('/zh/') ? 'zh' : 'en')
  }
}

function redirectOnFirstVisit() {
  const savedLanguage = localStorage.getItem(languagePreferenceKey)

  if (savedLanguage) {
    return
  }

  const path = normalizePath(window.location.pathname)
  const detectedLanguage = detectBrowserLanguage()

  if (path === '/' && detectedLanguage === 'zh') {
    localStorage.setItem(languagePreferenceKey, 'zh')
    window.location.replace('/zh/')
    return
  }

  localStorage.setItem(languagePreferenceKey, 'en')
}

export default {
  extends: DefaultTheme,
  enhanceApp() {
    if (typeof window === 'undefined') {
      return
    }

    redirectOnFirstVisit()
    window.addEventListener('click', rememberManualLanguageChoice)
  }
}
