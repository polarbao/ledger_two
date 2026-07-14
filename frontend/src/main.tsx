import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import './styles/tokens.css'
import './styles/ui-primitives.css'
import App from './App.tsx'
import { applyTheme, readStoredTheme, resolveInitialTheme } from './theme/theme.ts'

applyTheme(resolveInitialTheme(readStoredTheme(), document.documentElement.dataset.theme))

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
