import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'
import { queryClient } from './lib/queryClient'
import { ThemeBootstrap } from './lib/theme'
import App from './App'
import './index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        {/* ThemeBootstrap mounts before App so the persisted theme/locale
            are applied to <html> on first paint — avoids the brief
            light-mode flash for users who set dark. */}
        <ThemeBootstrap />
        <App />
      </BrowserRouter>
    </QueryClientProvider>
  </React.StrictMode>,
)
