import { useMemo, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Activity,
  ArrowLeft,
  CheckCircle,
  Copy,
  Globe,
  KeyRound,
  Monitor,
  Settings,
  Terminal,
} from 'lucide-react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { getWorkspaceClaims } from '@/lib/jwt'

export default function InstallPage() {
  const navigate = useNavigate()
  const token = localStorage.getItem('authToken')
  const claims = getWorkspaceClaims(token)
  const isAdmin = claims?.wrole === 'admin' || claims?.wrole === 'owner'

  if (!token || !claims) {
    return (
      <div className="flex min-h-[calc(100vh-3.5rem)] items-center justify-center bg-background p-4">
        <div className="w-full max-w-md space-y-4 rounded-xl border bg-card p-8 text-center shadow-sm">
          <p className="text-sm text-muted-foreground">Invalid or expired session.</p>
          <Button variant="outline" onClick={() => navigate('/login', { replace: true })}>
            Go to Login
          </Button>
        </div>
      </div>
    )
  }

  const platform = detectPlatform()
  const detectedHost = window.location.hostname || '127.0.0.1'
  const tenantSlug = claims.wslug
  const clientPort = '19515'
  const socksAddr = '127.0.0.1:1080'
  const controllerUrl = `http://${detectedHost}:8081`
  const connectUrl = `http://127.0.0.1:${clientPort}/connect?tenant=${encodeURIComponent(tenantSlug)}`

  const runCommand = useMemo(() => {
    return [
      'cd /path/to/zero-trust/services/ztna-client',
      `CONTROLLER_URL="${controllerUrl}" ZTNA_TENANT="${tenantSlug}" SOCKS5_ADDR="${socksAddr}" cargo run`,
    ].join('\n')
  }, [controllerUrl, tenantSlug])

  const smokeTestCommand = useMemo(() => {
    return `curl -v --socks5 ${socksAddr} http://example.com`
  }, [socksAddr])

  const copyText = async (text: string, label: string) => {
    try {
      await navigator.clipboard.writeText(text)
      toast.success(`${label} copied`)
    } catch {
      toast.error(`Failed to copy ${label.toLowerCase()}`)
    }
  }

  return (
    <div className="min-h-[calc(100vh-3.5rem)] bg-background p-4 md:p-6">
      <div className="mx-auto flex w-full max-w-5xl flex-col gap-6">
        <div className="flex items-center justify-between">
          <Button variant="ghost" className="gap-2" onClick={() => navigate('/app')}>
            <ArrowLeft className="h-4 w-4" />
            Back to Home
          </Button>
          {isAdmin && (
            <Button variant="outline" className="gap-2" onClick={() => navigate('/dashboard/groups', { replace: true })}>
              <Settings className="h-4 w-4" />
              Go to Dashboard
            </Button>
          )}
        </div>

        <div className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
          <Card className="border-primary/20 bg-gradient-to-br from-card via-card to-primary/5">
            <CardHeader className="space-y-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-primary/10">
                <Monitor className="h-6 w-6 text-primary" />
              </div>
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <CardTitle className="text-2xl">Install ZTNA Client</CardTitle>
                  <Badge variant="outline">{platform}</Badge>
                </div>
                <CardDescription className="max-w-2xl text-sm">
                  Run the local client, authenticate to <span className="font-medium text-foreground">{tenantSlug}</span>,
                  then point your browser or CLI tools at the local SOCKS5 proxy to reach protected resources.
                </CardDescription>
              </div>
            </CardHeader>
            <CardContent className="grid gap-4 sm:grid-cols-3">
              <InfoBlock
                icon={<Globe className="h-4 w-4 text-primary" />}
                label="Controller"
                value={controllerUrl}
              />
              <InfoBlock
                icon={<KeyRound className="h-4 w-4 text-primary" />}
                label="Tenant"
                value={tenantSlug}
              />
              <InfoBlock
                icon={<Activity className="h-4 w-4 text-primary" />}
                label="Local SOCKS5"
                value={socksAddr}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Setup Checklist</CardTitle>
              <CardDescription>Use this order to get the client working cleanly.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              {[
                'Run the ztna-client locally on your machine.',
                `Open ${connectUrl} and complete sign-in.`,
                `Configure your app or terminal to use SOCKS5 at ${socksAddr}.`,
                'Verify access with a curl or nc test to a protected resource.',
              ].map((step, index) => (
                <div key={step} className="flex items-start gap-3 rounded-lg border p-3">
                  <div className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
                    {index + 1}
                  </div>
                  <p className="text-sm text-muted-foreground">{step}</p>
                </div>
              ))}
            </CardContent>
          </Card>
        </div>

        <Tabs defaultValue="run" className="space-y-4">
          <TabsList>
            <TabsTrigger value="run">Run Client</TabsTrigger>
            <TabsTrigger value="auth">Authenticate</TabsTrigger>
            <TabsTrigger value="test">Test Access</TabsTrigger>
          </TabsList>

          <TabsContent value="run">
            <CommandCard
              icon={<Terminal className="h-4 w-4" />}
              title="Start the Client"
              description="This starts the local auth service and the SOCKS5 listener for user traffic."
              command={runCommand}
              copyLabel="Run command"
              onCopy={copyText}
            />
          </TabsContent>

          <TabsContent value="auth">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Authenticate the User Session</CardTitle>
                <CardDescription>
                  The local client handles the callback flow itself. Start auth from the local client, not from a controller endpoint.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="rounded-xl border bg-muted/40 p-4">
                  <pre className="overflow-x-auto whitespace-pre-wrap font-mono text-xs">{connectUrl}</pre>
                </div>
                <div className="flex flex-wrap gap-3">
                  <Button className="gap-2" onClick={() => void copyText(connectUrl, 'Connect URL')}>
                    <Copy className="h-4 w-4" />
                    Copy Connect URL
                  </Button>
                  <Button variant="outline" onClick={() => window.open(connectUrl, '_blank', 'noopener,noreferrer')}>
                    Open Local Login
                  </Button>
                </div>
                <div className="flex items-center gap-2 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700">
                  <CheckCircle className="h-4 w-4 shrink-0" />
                  After sign-in, the client stores your device session locally and can pre-check ACLs before opening a connector tunnel.
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="test">
            <CommandCard
              icon={<Activity className="h-4 w-4" />}
              title="Smoke Test the Proxy"
              description="Point a tool at the local SOCKS5 proxy. Replace example.com with a protected resource once your tunnel path is ready."
              command={smokeTestCommand}
              copyLabel="Smoke test command"
              onCopy={copyText}
            />
          </TabsContent>
        </Tabs>

        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Current Workspace</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              <KeyValue label="Workspace" value={tenantSlug} />
              <KeyValue label="Trust Domain" value={`${tenantSlug}.zerotrust.com`} mono />
              <KeyValue label="Role" value={claims.wrole} capitalize />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">What This Client Does</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm text-muted-foreground">
              <p>Runs a local auth callback service on `127.0.0.1:{clientPort}`.</p>
              <p>Exposes a local SOCKS5 proxy on `{socksAddr}` for protected traffic.</p>
              <p>Uses client-side ACL pre-checks for fast split-tunnel decisions before connector-side enforcement.</p>
            </CardContent>
          </Card>
        </div>

        <div className="flex items-center gap-2 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700">
          <CheckCircle className="h-4 w-4 shrink-0" />
          <p>Your workspace access is ready. Install and run the local client on the machine that will access protected resources.</p>
        </div>
      </div>
    </div>
  )
}

function InfoBlock({ icon, label, value }: { icon: ReactNode; label: string; value: string }) {
  return (
    <div className="rounded-xl border bg-background/70 p-4">
      <div className="mb-2 flex items-center gap-2 text-sm font-medium">
        {icon}
        {label}
      </div>
      <p className="font-mono text-xs text-muted-foreground">{value}</p>
    </div>
  )
}

function CommandCard({
  icon,
  title,
  description,
  command,
  copyLabel,
  onCopy,
}: {
  icon: ReactNode
  title: string
  description: string
  command: string
  copyLabel: string
  onCopy: (text: string, label: string) => Promise<void>
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          {icon}
          {title}
        </CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="rounded-xl border bg-muted/40 p-4">
          <pre className="overflow-x-auto whitespace-pre-wrap font-mono text-xs">{command}</pre>
        </div>
        <Button className="gap-2" onClick={() => void onCopy(command, copyLabel)}>
          <Copy className="h-4 w-4" />
          Copy Command
        </Button>
      </CardContent>
    </Card>
  )
}

function KeyValue({
  label,
  value,
  mono,
  capitalize,
}: {
  label: string
  value: string
  mono?: boolean
  capitalize?: boolean
}) {
  return (
    <div className="flex justify-between gap-4">
      <span className="text-muted-foreground">{label}</span>
      <span className={`${mono ? 'font-mono text-xs' : ''} ${capitalize ? 'capitalize' : ''}`}>{value}</span>
    </div>
  )
}

function detectPlatform(): string {
  const ua = navigator.userAgent.toLowerCase()
  if (ua.includes('linux')) return 'Linux'
  if (ua.includes('mac')) return 'macOS'
  if (ua.includes('win')) return 'Windows'
  return 'Unknown'
}
