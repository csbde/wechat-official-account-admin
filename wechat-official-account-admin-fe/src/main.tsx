import React from 'react'
import ReactDOM from 'react-dom/client'
import { App } from './App'
import './index.css'

// 初始化主题
const savedTheme = localStorage.getItem('woaa_theme')
if (savedTheme === 'dark') {
  document.documentElement.classList.add('dark')
} else {
  document.documentElement.classList.remove('dark')
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
