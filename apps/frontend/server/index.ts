import express, { Request, Response } from 'express'
import cors from 'cors'
import compression from 'compression'
import path from 'path'
import fs from 'fs'
import { BACKEND_URL } from '../lib/proxy'

import groupsRouter from './routes/groups'
import usersRouter from './routes/users'
import resourcesRouter from './routes/resources'
import connectorsRouter from './routes/connectors'
import remoteNetworksRouter from './routes/remote-networks'
import accessRulesRouter from './routes/access-rules'
import subjectsRouter from './routes/subjects'
import tokensRouter from './routes/tokens'
import serviceAccountsRouter from './routes/service-accounts'
import tunnelersRouter from './routes/tunnelers'
import policyRouter from './routes/policy'
import auditLogsRouter from './routes/audit-logs'

const app = express()

// Load apps/frontend/.env for the BFF server (Vite doesn't load it automatically for Node).
const envPath = path.resolve(__dirname, '../.env')
if (fs.existsSync(envPath)) {
  const contents = fs.readFileSync(envPath, 'utf8')
  contents.split(/\r?\n/).forEach((line) => {
    const trimmed = line.trim()
    if (!trimmed || trimmed.startsWith('#')) return
    const idx = trimmed.indexOf('=')
    if (idx === -1) return
    const key = trimmed.slice(0, idx).trim()
    const value = trimmed.slice(idx + 1).trim()
    if (key && process.env[key] === undefined) {
      process.env[key] = value
    }
  })
}

app.use(cors())
app.use(compression())
app.use(express.json())

app.use('/api/groups', groupsRouter)
app.use('/api/users', usersRouter)
app.use('/api/resources', resourcesRouter)
app.use('/api/connectors', connectorsRouter)
app.use('/api/remote-networks', remoteNetworksRouter)
app.use('/api/access-rules', accessRulesRouter)
app.use('/api/subjects', subjectsRouter)
app.use('/api/tokens', tokensRouter)
app.use('/api/service-accounts', serviceAccountsRouter)
app.use('/api/tunnelers', tunnelersRouter)
app.use('/api/policy', policyRouter)
app.use('/api/audit-logs', auditLogsRouter)

// POST /api/auth/logout — forwards to controller OAuth logout, then signals client to clear token
app.post('/api/auth/logout', async (_req: Request, res: Response) => {
  try {
    await fetch(`${BACKEND_URL}/oauth/logout`, { method: 'POST' })
  } catch {
    // Best-effort
  }
  res.json({ status: 'logged out' })
})

// Serve built Vite app in production
if (process.env.NODE_ENV === 'production') {
  const dist = path.resolve(__dirname, '../dist')
  app.use(express.static(dist))
  app.get('*', (_req, res) => res.sendFile(path.join(dist, 'index.html')))
}

const PORT = process.env.PORT || 3001
app.listen(PORT, () => {
  console.log(`Express BFF server running on :${PORT}`)
})
