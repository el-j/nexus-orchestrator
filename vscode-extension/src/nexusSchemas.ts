/**
 * nexusSchemas.ts — zod schemas that validate critical daemon API responses.
 */

import { z } from 'zod';

// ---- Zod schemas for runtime validation of critical API responses ----

export const TaskStatusSchema = z.enum([
  'QUEUED',
  'WAITING',
  'PROCESSING',
  'COMPLETED',
  'FAILED',
  'CANCELLED',
  'TOO_LARGE',
  'NO_PROVIDER',
  'DRAFT',
  'BACKLOG',
]);

/** Runtime shape of a Task returned by the daemon. */
export const TaskSchema = z
  .object({
    id: z.string(),
    projectPath: z.string(),
    targetFile: z.string(),
    instruction: z.string(),
    contextFiles: z
      .array(z.string())
      .nullish()
      .transform((v) => v ?? []),
    modelId: z.string().optional(),
    providerHint: z.string().optional(),
    ranBy: z.string().optional(),
    command: z.enum(['plan', 'execute', 'auto', '']).optional(),
    status: TaskStatusSchema,
    createdAt: z.string(),
    updatedAt: z.string(),
    logs: z.string().optional(),
    aiSessionId: z.string().optional(),
    claimedBy: z.string().optional(),
    claimedAt: z.string().optional(),
    retryCount: z.number().optional(),
    waitUntil: z.string().optional(),
  })
  .passthrough();

/** Runtime shape of an AI session returned by the daemon. */
export const AISessionSchema = z
  .object({
    id: z.string(),
    agentName: z.string(),
    source: z.string(),
    status: z.string(),
    lastActivity: z.string(),
    projectPath: z.string().optional(),
    delegatedToNexus: z.boolean().optional(),
    delegationTimestamp: z.string().optional(),
    agentCapabilities: z.array(z.string()).optional(),
    detectionMethod: z.string().optional(),
  })
  .passthrough();

/** Runtime shape of a provider entry returned by the daemon. */
export const ProviderSchema = z
  .object({
    name: z.string(),
    active: z.boolean(),
    kind: z.string().optional(),
    activeModel: z.string().optional(),
    models: z.array(z.string()).optional(),
    baseURL: z.string().optional(),
    details: z.string().optional(),
    error: z.string().optional(),
    contextLimit: z.number().optional(),
    rateLimited: z.boolean().optional(),
    rateLimitedUntil: z.string().optional(),
    cooldownReason: z.string().optional(),
    quota: z
      .object({
        window: z.string(),
        status: z.string(),
        utilization: z.number(),
        resetsAt: z.string().optional(),
      })
      .optional(),
  })
  .passthrough();

/** Runtime shape of a persisted provider configuration. */
export const ProviderConfigSchema = z
  .object({
    id: z.string(),
    name: z.string(),
    kind: z.string(),
    baseURL: z.string().optional(),
    model: z.string().optional(),
    enabled: z.boolean(),
  })
  .passthrough();
