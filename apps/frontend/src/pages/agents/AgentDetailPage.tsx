import { useParams } from 'react-router-dom';
import { AgentInstall } from '@/components/dashboard/agents/agent-install';

export default function AgentDetailPage() {
  const { agentId } = useParams();
  return <AgentInstall initialAgentId={agentId} />;
}
