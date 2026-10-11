/**
 * Static content for the MCP Integration page: setup snippets, the tool
 * catalogue, usage examples and GitHub Action samples.
 */

export const toc = [
  { id: 'what-is-mcp', label: 'What is MCP?' },
  { id: 'claude-desktop', label: 'Claude Desktop Setup' },
  { id: 'vscode-setup', label: 'VS Code Setup' },
  { id: 'available-tools', label: 'Available Tools' },
  { id: 'examples', label: 'Usage Examples' },
  { id: 'brain-examples', label: 'Brain Tools' },
  { id: 'protocol', label: 'Protocol Details' },
  { id: 'troubleshooting', label: 'Troubleshooting' },
  { id: 'github-actions', label: 'GitHub Actions' },
];

export const claudeConfig = `{
  "mcpServers": {
    "nexus-orchestrator": {
      "url": "http://localhost:63988/mcp"
    }
  }
}`;

export const examples = [
  {
    title: 'Submit a Task',
    request: `{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "submit_task",
    "arguments": {
      "projectPath": "/path/to/project",
      "targetFile": "handler.go",
      "instruction": "Add error handling to the HTTP handler",
      "command": "execute"
    }
  }
}`,
    response: `{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\\"id\\":\\"abc-123\\",\\"status\\":\\"QUEUED\\"}"
      }
    ]
  }
}`,
  },
  {
    title: 'Get Task Status',
    request: `{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "get_task",
    "arguments": {
      "id": "abc-123"
    }
  }
}`,
    response: '',
  },
  {
    title: 'Check Available Providers',
    request: `{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "get_providers"
  }
}`,
    response: `{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "[{\\"name\\":\\"LM Studio\\",\\"active\\":true,\\"activeModel\\":\\"codellama\\"},{\\"name\\":\\"Ollama\\",\\"active\\":false}]"
      }
    ]
  }
}`,
  },
  {
    title: 'Health Check',
    request: `{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "health"
  }
}`,
    response: '',
  },
];

export const vsCodeConfig = `{
  "servers": {
    "Nexus Orchestrator": {
      "type": "http",
      "url": "http://127.0.0.1:63988/mcp"
    }
  }
}`;

export const brainExamples = [
  {
    title: 'Check Brain Status',
    request: `{
  "jsonrpc": "2.0",
  "id": 10,
  "method": "tools/call",
  "params": {
    "name": "get_brain_status",
    "arguments": {
      "projectPath": "/your/project"
    }
  }
}`,
    response: `{
  "jsonrpc": "2.0",
  "id": 10,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\\"projectPath\\":\\"/your/project\\",\\"entryCount\\":42,\\"lastIngested\\":\\"2025-01-01T00:00:00Z\\"}"
      }
    ]
  }
}`,
  },
  {
    title: 'Ingest a Knowledge File',
    request: `{
  "jsonrpc": "2.0",
  "id": 11,
  "method": "tools/call",
  "params": {
    "name": "ingest_knowledge",
    "arguments": {
      "projectPath": "/your/project",
      "filePath": "/your/project/CLAUDE.md"
    }
  }
}`,
    response: `{
  "jsonrpc": "2.0",
  "id": 11,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\\"ingested\\":12,\\"projectPath\\":\\"/your/project\\"}"
      }
    ]
  }
}`,
  },
  {
    title: 'Get Project Context',
    request: `{
  "jsonrpc": "2.0",
  "id": 12,
  "method": "tools/call",
  "params": {
    "name": "get_project_context",
    "arguments": {
      "projectPath": "/your/project",
      "maxTokens": 800
    }
  }
}`,
    response: '',
  },
  {
    title: 'Search Knowledge Base',
    request: `{
  "jsonrpc": "2.0",
  "id": 13,
  "method": "tools/call",
  "params": {
    "name": "search_knowledge",
    "arguments": {
      "projectPath": "/your/project",
      "query": "architecture",
      "limit": 5
    }
  }
}`,
    response: '',
  },
];

export const protocolDetails = [
  { label: 'Protocol', value: 'JSON-RPC 2.0' },
  { label: 'Version', value: '2024-11-05' },
  { label: 'Endpoint', value: 'POST /mcp' },
  { label: 'Default Port', value: '63988' },
];

export const troubleshootingRows = [
  {
    issue: 'Connection refused',
    solution: 'Start nexus-daemon first: <code class="text-slate-300">./nexus-daemon</code>',
  },
  {
    issue: 'Port conflict',
    solution: 'Use <code class="text-slate-300">NEXUS_MCP_ADDR=:9090</code> to change the MCP port',
  },
  {
    issue: 'No tools in Claude',
    solution:
      'Check URL ends with <code class="text-slate-300">/mcp</code>, restart Claude Desktop',
  },
  {
    issue: 'Task stuck in QUEUED',
    solution:
      'Check <code class="text-slate-300">GET /api/providers</code> — ensure at least one LLM provider is active',
  },
];

export const ghActionBasic = `# .github/workflows/ai-task.yml
name: AI Code Generation

on:
  workflow_dispatch:
    inputs:
      instruction:
        description: 'What should the LLM implement?'
        required: true

jobs:
  generate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Submit AI task
        id: nexus
        uses: el-j/nexus-orchestrator@v1
        with:
          instruction: \${{ github.event.inputs.instruction }}
          target_file: 'output.go'
          command: execute
          openai_api_key: \${{ secrets.OPENAI_API_KEY }}

      - name: Show result
        run: echo "\${{ steps.nexus.outputs.logs }}"`;

export const ghActionAgents = `# Single agent — identity loaded automatically from el-j/agency-agents
jobs:
  generate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Submit AI task as Backend Architect
        id: nexus
        uses: el-j/nexus-orchestrator@v1
        with:
          agent: engineering-backend-architect   # fetched from el-j/agency-agents@main
          instruction: 'Add JWT authentication middleware to the HTTP handler'
          target_file: 'internal/middleware/auth.go'
          command: execute
          anthropic_api_key: \${{ secrets.ANTHROPIC_API_KEY }}

      - name: Commit generated code
        if: steps.nexus.outputs.status == 'COMPLETED'
        run: |
          git config user.name "nexus-bot"
          git config user.email "nexus-bot@users.noreply.github.com"
          git add -A
          git commit -m "feat: AI-generated auth middleware" || echo "nothing to commit"
          git push`;

export const ghActionSwarm = `# Swarm — load all engineering agents and orchestrate as a team
jobs:
  swarm:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Submit task to engineering swarm
        id: nexus
        uses: el-j/nexus-orchestrator@v1
        with:
          agent_category: engineering        # loads all agents in the category
          instruction: 'Design and implement a rate-limiter for the API'
          command: plan
          openai_api_key: \${{ secrets.OPENAI_API_KEY }}
          openai_model: gpt-4o`;

export const actionInputs = [
  {
    name: 'instruction',
    default: '—',
    desc: 'Task text sent to the LLM (required unless task_file is set)',
  },
  {
    name: 'task_file',
    default: '—',
    desc: 'Path to a .md/.txt file whose content becomes the instruction',
  },
  {
    name: 'agent',
    default: '""',
    desc: 'Single agent slug from el-j/agency-agents (e.g. "engineering-backend-architect")',
  },
  {
    name: 'agents',
    default: '""',
    desc: 'Comma-separated agent slugs — generates a combined swarm prompt',
  },
  {
    name: 'agent_category',
    default: '""',
    desc: 'Load ALL agents in a category (e.g. "engineering", "design", "testing")',
  },
  {
    name: 'agent_ref',
    default: 'main',
    desc: 'Git ref of el-j/agency-agents to fetch from (branch, tag, SHA)',
  },
  {
    name: 'system_prompt',
    default: '""',
    desc: 'Raw system prompt override — takes precedence over agent/agents/agent_category',
  },
  {
    name: 'target_file',
    default: '""',
    desc: 'Relative path the LLM should write (e.g. src/utils.go)',
  },
  {
    name: 'command',
    default: 'execute',
    desc: '"plan" (orchestration) or "execute" (code generation)',
  },
  {
    name: 'openai_api_key',
    default: '""',
    desc: 'OpenAI API key — enables GPT models on the local daemon',
  },
  {
    name: 'anthropic_api_key',
    default: '""',
    desc: 'Anthropic API key — enables Claude models on the local daemon',
  },
  {
    name: 'github_copilot_token',
    default: '""',
    desc: 'GitHub Copilot token — enables GPT-4o via Copilot',
  },
  {
    name: 'timeout_seconds',
    default: '300',
    desc: 'Seconds to wait for task completion before failing',
  },
  { name: 'nexus_version', default: 'latest', desc: 'Release version to install (e.g. v0.2.0)' },
  {
    name: 'start_daemon',
    default: 'true',
    desc: 'Download + start a local daemon; set false to use daemon_url',
  },
  {
    name: 'daemon_url',
    default: 'http://127.0.0.1:63987',
    desc: 'URL of existing daemon (only when start_daemon=false)',
  },
];
