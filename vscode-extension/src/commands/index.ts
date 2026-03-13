/**
 * commands/index.ts — Barrel export for all Nexus command implementations.
 */

export { sendCurrentContextCommand, submitTaskCommand } from "./submitTask";
export { selectProviderCommand } from "./selectProvider";
export { viewQueueCommand } from "./viewQueue";

// eslint-disable-next-line @typescript-eslint/no-unused-vars
export async function showProvidersCommand(): Promise<void> {
  // Full implementation in TASK-135.
}
