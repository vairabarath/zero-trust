import { Router, Request, Response } from 'express'
import { proxyToBackend } from '../../lib/proxy'

const router = Router()

// GET /api/tunnelers
router.get('/', async (_req: Request, res: Response) => {
  try {
    const tunnelers = await proxyToBackend('/api/tunnelers')
    res.json(tunnelers)
  } catch (error) {
    res.status(500).json({ error: (error as Error).message })
  }
})

// POST /api/tunnelers
router.post('/', async (req: Request, res: Response) => {
  try {
    const result = await proxyToBackend('/api/tunnelers', {
      method: 'POST',
      body: JSON.stringify(req.body),
    })
    res.json(result)
  } catch (error) {
    res.status(500).json({ error: (error as Error).message })
  }
})

// GET /api/tunnelers/:tunneledId
router.get('/:tunneledId', async (req: Request, res: Response) => {
  try {
    const { tunneledId } = req.params
    const result = await proxyToBackend(`/api/tunnelers/${tunneledId}`)
    res.json(result)
  } catch (error) {
    res.status(500).json({ error: (error as Error).message })
  }
})

// DELETE /api/tunnelers/:tunneledId
router.delete('/:tunneledId', async (req: Request, res: Response) => {
  try {
    const { tunneledId } = req.params
    const result = await proxyToBackend(`/api/tunnelers/${tunneledId}`, {
      method: 'DELETE',
    })
    res.json(result)
  } catch (error) {
    res.status(500).json({ error: (error as Error).message })
  }
})

// POST /api/tunnelers/:tunneledId/revoke
router.post('/:tunneledId/revoke', async (req: Request, res: Response) => {
  try {
    const { tunneledId } = req.params
    const result = await proxyToBackend(`/api/tunnelers/${tunneledId}/revoke`, {
      method: 'POST',
    })
    res.json(result)
  } catch (error) {
    res.status(500).json({ error: (error as Error).message })
  }
})

// POST /api/tunnelers/:tunneledId/grant
router.post('/:tunneledId/grant', async (req: Request, res: Response) => {
  try {
    const { tunneledId } = req.params
    const result = await proxyToBackend(`/api/tunnelers/${tunneledId}/grant`, {
      method: 'POST',
    })
    res.json(result)
  } catch (error) {
    res.status(500).json({ error: (error as Error).message })
  }
})

export default router
