import { useQuery } from '@tanstack/react-query';
import { ShieldCheck } from 'lucide-react';
import { queryKeys } from '../../api/queryKeys';
import { systemApi } from '../../api/system.api';
import { getDeploymentChannelMeta } from './deploymentChannel';

export default function DeploymentBadge() {
  const { data } = useQuery({
    queryKey: queryKeys.system.health,
    queryFn: ({ signal }) => systemApi.getHealth(signal),
    staleTime: Number.POSITIVE_INFINITY,
  });

  if (!data) return null;

  const meta = getDeploymentChannelMeta(data.deployment_channel);

  return (
    <span
      className={`deployment-badge deployment-badge--${meta.tone}`}
      title={`${meta.label}，应用版本 ${data.version}，数据库 schema ${data.schema_version}`}
    >
      <ShieldCheck size={12} aria-hidden="true" />
      <span>{meta.label}</span>
      <span className="deployment-badge__version">{data.version}</span>
    </span>
  );
}
