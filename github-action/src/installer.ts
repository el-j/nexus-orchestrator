import * as core from '@actions/core';
import * as tc from '@actions/tool-cache';
import * as path from 'path';
import * as fs from 'fs';

const GITHUB_RELEASES = 'https://github.com/el-j/nexus-orchestrator/releases';

/** Detect current OS + arch → archive name segment */
function platformSegment(): { os: string; arch: string; ext: string } {
  const raw = process.platform;
  const os =
    raw === 'darwin' ? 'darwin' : raw === 'win32' ? 'windows' : 'linux';
  const ext = os === 'windows' ? 'zip' : 'tar.gz';

  const rawArch = process.arch;
  const arch = rawArch === 'arm64' ? 'arm64' : 'amd64';

  return { os, arch, ext };
}

/** Resolve the download URL for a nexus-orchestrator release */
export function resolveDownloadUrl(version: string): string {
  const { os, arch, ext } = platformSegment();
  const archive = `nexus-orchestrator-${os}-${arch}.${ext}`;
  const base =
    version === 'latest'
      ? `${GITHUB_RELEASES}/latest/download`
      : `${GITHUB_RELEASES}/download/${version}`;
  return `${base}/${archive}`;
}

/** Download, extract, and return the path to the nexus-daemon binary */
export async function installDaemon(version: string): Promise<string> {
  const url = resolveDownloadUrl(version);
  core.info(`Downloading nexus-daemon from ${url}`);

  const archivePath = await tc.downloadTool(url);
  const { ext } = platformSegment();

  let extractDir: string;
  if (ext === 'zip') {
    extractDir = await tc.extractZip(archivePath);
  } else {
    extractDir = await tc.extractTar(archivePath);
  }

  const binName = process.platform === 'win32' ? 'nexus-daemon.exe' : 'nexus-daemon';
  const binPath = path.join(extractDir, binName);

  if (!fs.existsSync(binPath)) {
    throw new Error(
      `nexus-daemon binary not found in extracted archive at ${binPath}`
    );
  }

  if (process.platform !== 'win32') {
    fs.chmodSync(binPath, 0o755);
  }

  core.info(`nexus-daemon installed at ${binPath}`);
  return binPath;
}
