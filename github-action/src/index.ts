import * as core from '@actions/core';
import * as os from 'os';
import * as path from 'path';
import * as fs from 'fs';

import { installDaemon } from './installer.js';
import { startDaemon, printDaemonLog } from './daemon.js';
import { NexusClient } from './submit.js';
import {
  resolveAgents,
  resolveCategory,
  buildSwarmPrompt,
} from './agents.js';
import type { ActionInputs, TaskRequest } from './types.js';

// ── Input parsing ──────────────────────────────────────────────────────────

function getInputs(): ActionInputs {
  const contextFilesRaw = core.getInput('context_files').trim();
  return {
    instruction: core.getInput('instruction').trim(),
    taskFile: core.getInput('task_file').trim(),
    projectPath:
      core.getInput('project_path').trim() ||
      (process.env['GITHUB_WORKSPACE'] ?? process.cwd()),
    targetFile: core.getInput('target_file').trim(),
    contextFiles:
      contextFilesRaw.length > 0
        ? contextFilesRaw.split(',').map((f) => f.trim()).filter(Boolean)
        : [],
    command: (core.getInput('command').trim() || 'execute') as 'execute' | 'plan',
    model: core.getInput('model').trim(),
    provider: core.getInput('provider').trim(),
    agent: core.getInput('agent').trim(),
    agents: core.getInput('agents').trim(),
    agentCategory: core.getInput('agent_category').trim(),
    agentRef: core.getInput('agent_ref').trim() || 'main',
    systemPrompt: core.getInput('system_prompt').trim(),
    daemonUrl:
      core.getInput('daemon_url').trim() || 'http://127.0.0.1:63987',
    startDaemon: core.getInput('start_daemon').trim() !== 'false',
    nexusVersion: core.getInput('nexus_version').trim() || 'latest',
    timeoutSeconds: parseInt(core.getInput('timeout_seconds').trim() || '300', 10),
    openaiApiKey: core.getInput('openai_api_key'),
    openaiModel: core.getInput('openai_model').trim(),
    anthropicApiKey: core.getInput('anthropic_api_key'),
    anthropicModel: core.getInput('anthropic_model').trim(),
    githubCopilotToken: core.getInput('github_copilot_token'),
    githubCopilotModel: core.getInput('github_copilot_model').trim(),
  };
}

// ── Agent resolution ───────────────────────────────────────────────────────

async function resolveSystemPrompt(inputs: ActionInputs): Promise<string> {
  // Manual system_prompt takes precedence
  if (inputs.systemPrompt.length > 0) {
    core.info('Using manual system_prompt input.');
    return inputs.systemPrompt;
  }

  // Single agent
  if (inputs.agent.length > 0) {
    core.info(`Loading agent: ${inputs.agent}`);
    const [ag] = await resolveAgents([inputs.agent], inputs.agentRef);
    if (ag == null) throw new Error(`Agent "${inputs.agent}" not found`);
    core.info(`Loaded agent: ${ag.name} (${ag.category})`);
    return ag.systemPrompt;
  }

  // Swarm — named agents
  if (inputs.agents.length > 0) {
    const slugs = inputs.agents.split(',').map((s) => s.trim()).filter(Boolean);
    core.info(`Loading swarm agents: ${slugs.join(', ')}`);
    const agentList = await resolveAgents(slugs, inputs.agentRef);
    core.info(`Loaded ${agentList.length} agents`);
    return buildSwarmPrompt(agentList, 'Agency Swarm');
  }

  // Swarm — by category
  if (inputs.agentCategory.length > 0) {
    core.info(`Loading all agents in category: ${inputs.agentCategory}`);
    const agentList = await resolveCategory(inputs.agentCategory, inputs.agentRef);
    core.info(`Loaded ${agentList.length} agents from category "${inputs.agentCategory}"`);
    return buildSwarmPrompt(agentList, inputs.agentCategory);
  }

  return '';
}

// ── Instruction assembly ───────────────────────────────────────────────────

function buildInstruction(inputs: ActionInputs, systemPrompt: string): string {
  let instruction = inputs.instruction;
  if (inputs.taskFile.length > 0) {
    if (!fs.existsSync(inputs.taskFile)) {
      throw new Error(`task_file not found: ${inputs.taskFile}`);
    }
    instruction = fs.readFileSync(inputs.taskFile, 'utf-8').trim();
  }
  if (instruction.length === 0) {
    throw new Error(
      'Either the "instruction" or "task_file" input must be provided.'
    );
  }
  if (systemPrompt.length > 0) {
    return `<system>\n${systemPrompt}\n</system>\n\n${instruction}`;
  }
  return instruction;
}

// ── Main ───────────────────────────────────────────────────────────────────

async function run(): Promise<void> {
  const inputs = getInputs();
  let daemonHandle: { stop(): void; logFile: string } | null = null;
  const tmpDir = os.tmpdir();

  try {
    // 1. Optionally start daemon
    let daemonUrl = inputs.daemonUrl;
    if (inputs.startDaemon) {
      const binPath = await installDaemon(inputs.nexusVersion);
      daemonHandle = await startDaemon({
        binPath,
        listenAddr: '127.0.0.1:63987',
        mcpAddr: '127.0.0.1:63988',
        dbPath: path.join(tmpDir, 'nexus-action.db'),
        logFile: path.join(tmpDir, 'nexus-daemon.log'),
        openaiApiKey: inputs.openaiApiKey,
        openaiModel: inputs.openaiModel,
        anthropicApiKey: inputs.anthropicApiKey,
        anthropicModel: inputs.anthropicModel,
        githubCopilotToken: inputs.githubCopilotToken,
        githubCopilotModel: inputs.githubCopilotModel,
      });
      daemonUrl = 'http://127.0.0.1:63987';
    }

    // 2. Resolve agent identity / system prompt
    const systemPrompt = await resolveSystemPrompt(inputs);

    // 3. Build instruction
    const instruction = buildInstruction(inputs, systemPrompt);

    // 4. Submit task
    const nexus = new NexusClient(daemonUrl);
    const taskReq: TaskRequest = {
      projectPath: inputs.projectPath,
      instruction,
      ...(inputs.targetFile.length > 0 && { targetFile: inputs.targetFile }),
      ...(inputs.contextFiles.length > 0 && { contextFiles: inputs.contextFiles }),
      ...(inputs.command.length > 0 && { command: inputs.command }),
      ...(inputs.model.length > 0 && { modelId: inputs.model }),
      ...(inputs.provider.length > 0 && { providerHint: inputs.provider }),
    };

    const submitted = await nexus.submitTask(taskReq);
    core.info(`Task submitted: ${submitted.task_id}`);
    core.info(`Dashboard: ${daemonUrl}/ui`);
    core.setOutput('task_id', submitted.task_id);

    // 5. Wait for completion
    const finalTask = await nexus.waitForTask(
      submitted.task_id,
      inputs.timeoutSeconds * 1_000
    );

    core.setOutput('status', finalTask.status);
    core.setOutput('logs', finalTask.logs);

    // 6. Handle terminal status
    switch (finalTask.status) {
      case 'COMPLETED':
        core.info(`Task ${submitted.task_id} completed successfully.`);
        break;
      case 'FAILED':
        core.setFailed(`Task ${submitted.task_id} failed.\n${finalTask.logs}`);
        break;
      case 'TOO_LARGE':
        core.setFailed(
          `Task ${submitted.task_id} rejected — instruction exceeds the model context window.`
        );
        break;
      case 'CANCELLED':
        core.warning(`Task ${submitted.task_id} was cancelled.`);
        break;
      default:
        core.setFailed(`Unexpected terminal status: ${finalTask.status}`);
    }
  } catch (err: unknown) {
    core.setOutput('status', 'FAILED');
    core.setOutput('logs', '');
    core.setFailed(err instanceof Error ? err.message : String(err));
  } finally {
    // Always stop daemon and print its log
    if (daemonHandle != null) {
      daemonHandle.stop();
      printDaemonLog(daemonHandle.logFile);
    }
  }
}

run();
