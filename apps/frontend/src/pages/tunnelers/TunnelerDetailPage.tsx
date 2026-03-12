import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Link } from 'react-router-dom';
import { getTunneler, deleteTunneler, revokeTunneler, grantTunneler } from '@/lib/mock-api';
import { Tunneler, RemoteNetwork } from '@/lib/types';
import { Button } from '@/components/ui/button';
import { Loader2, ArrowLeft, AlertTriangle, ShieldOff, ShieldCheck, Trash2 } from 'lucide-react';
import { TunnelerInfoSection } from '@/components/dashboard/tunnelers/tunneler-info-section';
import { ConnectorLogs } from '@/components/dashboard/connectors/connector-logs';
import { TunnelerInstall } from '@/components/dashboard/tunnelers/tunneler-install';
import { toast } from 'sonner';

interface LogEntry {
  id: number;
  timestamp: string;
  message: string;
}

export default function TunnelerDetailPage() {
  const { tunnelerId } = useParams();
  const navigate = useNavigate();
  const [tunneler, setTunneler] = useState<Tunneler | null>(null);
  const [network, setNetwork] = useState<RemoteNetwork | undefined>(undefined);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [isRevoking, setIsRevoking] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  const loadTunnelerData = async (opts?: { silent?: boolean }) => {
    if (!opts?.silent) {
      setLoading(true);
    }
    try {
      const { tunneler: fetchedTunneler, network: fetchedNetwork, logs: fetchedLogs } = await getTunneler(tunnelerId as string);
      setTunneler(fetchedTunneler);
      setNetwork(fetchedNetwork);
      setLogs(fetchedLogs);
    } catch (error) {
      console.error('Failed to load tunneler details:', error);
    } finally {
      if (!opts?.silent) {
        setLoading(false);
      }
    }
  };

  useEffect(() => {
    if (tunnelerId) {
      loadTunnelerData();
    }
  }, [tunnelerId]);

  const handleDelete = async () => {
    if (!tunnelerId) return;
    if (!window.confirm(`Delete tunneler "${tunneler?.name ?? tunnelerId}"? This will remove it from the controller.`)) return;
    setIsDeleting(true);
    try {
      await deleteTunneler(tunnelerId);
      toast.success('Tunneler deleted successfully.');
      navigate('/dashboard/tunnelers');
    } catch (error) {
      toast.error('Failed to delete tunneler. Check that the backend is running.');
    } finally {
      setIsDeleting(false);
    }
  };

  const handleRevoke = async () => {
    if (!tunnelerId) return;
    setIsRevoking(true);
    try {
      await revokeTunneler(tunnelerId);
      toast.success('Tunneler access revoked.');
      await loadTunnelerData({ silent: true });
    } catch (error) {
      toast.error('Failed to revoke tunneler access.');
    } finally {
      setIsRevoking(false);
    }
  };

  const handleGrant = async () => {
    if (!tunnelerId) return;
    setIsRevoking(true);
    try {
      await grantTunneler(tunnelerId);
      toast.success('Tunneler access granted.');
      await loadTunnelerData({ silent: true });
    } catch (error) {
      toast.error('Failed to grant tunneler access.');
    } finally {
      setIsRevoking(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center p-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!tunneler) {
    return (
      <div className="space-y-6 p-6">
        <Link to="/dashboard/tunnelers">
          <Button variant="ghost" className="gap-2">
            <ArrowLeft className="h-4 w-4" />
            Back to Tunnelers
          </Button>
        </Link>
        <div className="text-center py-20">
          <AlertTriangle className="mx-auto h-16 w-16 text-muted-foreground" />
          <h2 className="mt-4 text-2xl font-bold">Tunneler Not Found</h2>
          <p className="mt-2 text-muted-foreground">
            This tunneler is not registered yet. Use the install command below to enroll it.
          </p>
        </div>
        <TunnelerInstall initialTunnelerId={tunnelerId} />
      </div>
    );
  }

  return (
    <div className="space-y-6 p-6">
      {/* Breadcrumb & Header */}
      <div className="flex items-center justify-between">
        <div className="space-y-2">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Link to="/dashboard/tunnelers" className="hover:text-foreground">
              Tunnelers
            </Link>
            <span>/</span>
            <span>{tunneler.name}</span>
          </div>
          <h1 className="text-2xl font-bold">{tunneler.name}</h1>
        </div>
        <div className="flex gap-2">
          {tunneler.status !== 'revoked' && (
            <Button
              variant="outline"
              className="gap-2 text-orange-500 border-orange-500 hover:text-orange-600 hover:border-orange-600"
              onClick={handleRevoke}
              disabled={isRevoking}
            >
              {isRevoking ? <Loader2 className="h-4 w-4 animate-spin" /> : <ShieldOff className="h-4 w-4" />}
              Revoke
            </Button>
          )}
          {tunneler.status === 'revoked' && (
            <Button
              variant="outline"
              className="gap-2 text-green-500 border-green-500 hover:text-green-600 hover:border-green-600"
              onClick={handleGrant}
              disabled={isRevoking}
            >
              {isRevoking ? <Loader2 className="h-4 w-4 animate-spin" /> : <ShieldCheck className="h-4 w-4" />}
              Grant
            </Button>
          )}
          <Button variant="destructive" className="gap-2" onClick={handleDelete} disabled={isDeleting}>
            {isDeleting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
            Delete
          </Button>
        </div>
      </div>

      {/* Tunneler Info Section */}
      <TunnelerInfoSection tunneler={tunneler} network={network} />

      {/* Logs Section */}
      <ConnectorLogs logs={logs} />

      {/* Install card when not installed */}
      {!tunneler.installed && (
        <TunnelerInstall
          initialTunnelerId={tunnelerId}
          initialConnectorId={tunneler.connectorId}
        />
      )}
    </div>
  );
}
