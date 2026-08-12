export const isDemoMode = import.meta.env.VITE_DEMO_MODE === 'true'

export interface DemoServiceAccount {
  id: string
  name: string
  description: string
}

/** Public demo credentials — safe to expose in a portfolio deployment. */
export const demoConfig = {
  email: (import.meta.env.VITE_DEMO_EMAIL as string) || 'demo@halcyon.demo',
  password: (import.meta.env.VITE_DEMO_PASSWORD as string) || 'HalcyonDemo!2026',
  orgId: (import.meta.env.VITE_DEMO_ORG_ID as string) || '',
  primaryServiceAccountId:
    (import.meta.env.VITE_DEMO_SERVICE_ACCOUNT_ID as string) || 'svc_demoGatewayBot001',
  serviceAccounts: [
    {
      id: 'svc_demoGatewayBot001',
      name: 'API Gateway',
      description: 'Edge routing, authentication, and rate limiting',
    },
    {
      id: 'svc_demoStagingBot001',
      name: 'Staging Integrations',
      description: 'Webhook delivery and third-party connectors',
    },
    {
      id: 'svc_demoWorkflowJob01',
      name: 'Workflow Runner',
      description: 'Background jobs and workflow execution',
    },
  ] satisfies DemoServiceAccount[],
}

export function applyDemoSessionDefaults(
  setOrgId: (id: string | null) => void,
  setServiceAccountId: (id: string) => void,
): void {
  if (demoConfig.orgId) {
    setOrgId(demoConfig.orgId)
  }
  if (demoConfig.primaryServiceAccountId) {
    setServiceAccountId(demoConfig.primaryServiceAccountId)
  }
}
