import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { App } from './app/App'
import { configFromEnv } from './app/config'
import './shared/ui/tokens.css'
import './shared/ui/base.css'

const root = document.getElementById('root')
if (!root) throw new Error('missing #root')

createRoot(root).render(
  <StrictMode>
    <App config={configFromEnv(import.meta.env)} />
  </StrictMode>,
)
