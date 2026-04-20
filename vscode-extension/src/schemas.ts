/**
 * schemas.ts — Zod runtime validation schemas for all nexusOrchestrator API responses.
 *
 * Import these schemas in nexusClient.ts to validate API responses at runtime.
 * All schemas are strict z.object({...}) definitions mirroring the TypeScript types.
 */

import { z } from 'zod';

// ---- Validation Error ----

export class NexusValidationError extends Error {
  constructor(
    public readonly path: string,
    public readonly issues: z.ZodIssue[],
  ) {
    super(`Nexus validation failed at ${path}: ${issues.map((i) => i.message).join(', ')}`);
    this.name = 'NexusValidationError';
  }
}

// ---- AI Activity Schema ----

export const AIActivitySchema = z
  .object({
    id: z.string(),
    sessionId: z.string().optional(),
    agentName: z.string(),
    activityType: z.enum(['message', 'tool_use', 'thinking', 'file_edit', 'generation']),
    summary: z.string(),
    projectPath: z.string().optional(),
    model: z.string().optional(),
    tokensIn: z.number().optional(),
    tokensOut: z.number().optional(),
    timestamp: z.string(),
    metadata: z.record(z.string(), z.string()).optional(),
  })
  .passthrough();

// ---- AI Session Schema ----

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

// ---- Discovered Agent Schema ----

export const DiscoveredAgentSchema = z
  .object({
    id: z.string(),
    kind: z.string(),
    name: z.string(),
    detectionMethod: z.string(),
    processName: z.string().optional(),
    cliPath: z.string().optional(),
    configPath: z.string().optional(),
    mcpEndpoint: z.string().optional(),
    isRunning: z.boolean(),
    lastSeen: z.string(),
  })
  .passthrough();

// ---- Brain Status Schema ----

export const BrainStatusSchema = z
  .object({
    projectPath: z.string(),
    initialized: z.boolean(),
    entryCount: z.number(),
    kindCounts: z.record(z.string(), z.number()),
    totalTokens: z.number(),
    lastUpdated: z.string().optional(),
  })
  .passthrough();

// ---- Context Section Schema ----

export const ContextSectionSchema = z
  .object({
    title: z.string().optional(),
    topic: z.string(),
    kind: z.string(),
    content: z.string(),
    tokens: z.number(),
    source: z.string(),
  })
  .passthrough();

// ---- Context Response Schema ----

export const ContextResponseSchema = z
  .object({
    projectPath: z.string(),
    sections: z.array(ContextSectionSchema),
    totalTokens: z.number(),
    truncated: z.boolean(),
  })
  .passthrough();

// ---- Project Knowledge Schema ----

export const ProjectKnowledgeSchema = z
  .object({
    id: z.string(),
    projectPath: z.string(),
    kind: z.string(),
    topic: z.string(),
    content: z.string(),
    tokenCount: z.number(),
    relevanceScore: z.number(),
    createdAt: z.string(),
    updatedAt: z.string(),
  })
  .passthrough();

// ---- Delegate Response Schema ----

export const DelegateResponseSchema = z
  .object({
    instruction: z.string(),
    sessionId: z.string(),
  })
  .passthrough();

// ---- Ingest Knowledge Response Schema ----

export const IngestKnowledgeResponseSchema = z
  .object({
    ingestedSections: z.number(),
  })
  .passthrough();

// ---- Search Knowledge Response Schema ----

export const SearchKnowledgeResponseSchema = z
  .object({
    results: z.array(ContextSectionSchema),
  })
  .passthrough();

// ---- File Map Response Schema ----

export const FileMapResponseSchema = z
  .object({
    filePaths: z.array(z.string()),
  })
  .passthrough();
