import { Router, Request, Response } from 'express'
import { BACKEND_URL } from '../../lib/proxy'

const router = Router()

// GET /api/config - returns controller addresses for UI auto-population
router.get('/', (req: Request, res: Response) => {
  let controllerHost = 'localhost'
  let controllerHttpPort = '8081'
  try {
    const url = new URL(BACKEND_URL)
    controllerHost = url.hostname
    controllerHttpPort = url.port || '8081'
  } catch {}

  // If backend URL is localhost, prefer the incoming request host for LAN access.
  if (
    controllerHost === 'localhost' ||
    controllerHost === '127.0.0.1' ||
    controllerHost === '0.0.0.0'
  ) {
    const forwardedHost = req.header('x-forwarded-host')
    const hostHeader = forwardedHost || req.header('host') || ''
    const host = hostHeader.split(',')[0]?.trim()
    if (host) {
      controllerHost = host.split(':')[0]
    }
  }

  // CONTROLLER_GRPC_ADDR can be explicitly set; otherwise derive from backend host + default port
  const controllerGrpcAddr =
    process.env.CONTROLLER_GRPC_ADDR || `${controllerHost}:8443`
  const controllerHttpAddr = `${controllerHost}:${controllerHttpPort}`

  res.json({ controllerHttpAddr, controllerGrpcAddr })
})

export default router
