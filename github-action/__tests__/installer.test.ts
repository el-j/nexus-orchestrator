import * as tc from '@actions/tool-cache';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';

import { resolveDownloadUrl, installDaemon } from '../src/installer.js';

const mockDownloadTool = tc.downloadTool as jest.MockedFunction<typeof tc.downloadTool>;
const mockExtractTar   = tc.extractTar   as jest.MockedFunction<typeof tc.extractTar>;
const mockExtractZip   = tc.extractZip   as jest.MockedFunction<typeof tc.extractZip>;

// ── resolveDownloadUrl ─────────────────────────────────────────────────────

describe('resolveDownloadUrl', () => {
  const GITHUB_RELEASES = 'https://github.com/el-j/nexus-orchestrator/releases';

  it('uses latest/download path for version "latest"', () => {
    const url = resolveDownloadUrl('latest');
    expect(url).toContain(`${GITHUB_RELEASES}/latest/download/`);
  });

  it('uses versioned download path for a specific version', () => {
    const url = resolveDownloadUrl('v1.2.3');
    expect(url).toContain(`${GITHUB_RELEASES}/download/v1.2.3/`);
  });

  it('includes the correct archive name with OS and arch', () => {
    const url = resolveDownloadUrl('latest');
    const rawArch = process.arch === 'arm64' ? 'arm64' : 'amd64';
    const rawPlatform = process.platform;
    const osName =
      rawPlatform === 'darwin' ? 'darwin' : rawPlatform === 'win32' ? 'windows' : 'linux';
    expect(url).toContain(`nexus-orchestrator-${osName}-${rawArch}`);
  });

  it('uses .zip extension on Windows', () => {
    const origPlatform = Object.getOwnPropertyDescriptor(process, 'platform');
    Object.defineProperty(process, 'platform', { value: 'win32', configurable: true });
    const url = resolveDownloadUrl('latest');
    expect(url).toMatch(/\.zip$/);
    if (origPlatform) Object.defineProperty(process, 'platform', origPlatform);
  });

  it('uses .tar.gz extension on Linux', () => {
    const origPlatform = Object.getOwnPropertyDescriptor(process, 'platform');
    Object.defineProperty(process, 'platform', { value: 'linux', configurable: true });
    const url = resolveDownloadUrl('latest');
    expect(url).toMatch(/\.tar\.gz$/);
    if (origPlatform) Object.defineProperty(process, 'platform', origPlatform);
  });

  it('uses .tar.gz extension on macOS', () => {
    const origPlatform = Object.getOwnPropertyDescriptor(process, 'platform');
    Object.defineProperty(process, 'platform', { value: 'darwin', configurable: true });
    const url = resolveDownloadUrl('latest');
    expect(url).toMatch(/\.tar\.gz$/);
    if (origPlatform) Object.defineProperty(process, 'platform', origPlatform);
  });
});

// ── installDaemon ──────────────────────────────────────────────────────────

describe('installDaemon', () => {
  let tmpDir: string;

  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'nexus-installer-test-'));
    mockDownloadTool.mockReset();
    mockExtractTar.mockReset();
    mockExtractZip.mockReset();
  });

  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true });
    jest.restoreAllMocks();
  });

  it('downloads and extracts a .tar.gz archive on non-Windows', async () => {
    // Only run this logic when not on Windows
    if (process.platform === 'win32') return;

    const extractDir = path.join(tmpDir, 'extracted');
    fs.mkdirSync(extractDir);
    const binName = 'nexus-daemon';
    const binPath = path.join(extractDir, binName);
    fs.writeFileSync(binPath, '#!/bin/sh\necho hello');

    mockDownloadTool.mockResolvedValue('/tmp/archive.tar.gz');
    mockExtractTar.mockResolvedValue(extractDir);

    const result = await installDaemon('latest');
    expect(mockDownloadTool).toHaveBeenCalledWith(expect.stringContaining('nexus-orchestrator'));
    expect(mockExtractTar).toHaveBeenCalledWith('/tmp/archive.tar.gz');
    expect(result).toBe(binPath);
  });

  it('downloads and extracts a .zip archive on Windows', async () => {
    const origPlatform = Object.getOwnPropertyDescriptor(process, 'platform');
    Object.defineProperty(process, 'platform', { value: 'win32', configurable: true });

    const extractDir = path.join(tmpDir, 'extracted-win');
    fs.mkdirSync(extractDir);
    const binPath = path.join(extractDir, 'nexus-daemon.exe');
    fs.writeFileSync(binPath, 'fake-binary');

    mockDownloadTool.mockResolvedValue('/tmp/archive.zip');
    mockExtractZip.mockResolvedValue(extractDir);

    const result = await installDaemon('v1.0.0');
    expect(mockDownloadTool).toHaveBeenCalledWith(expect.stringContaining('v1.0.0'));
    expect(mockExtractZip).toHaveBeenCalledWith('/tmp/archive.zip');
    expect(result).toBe(binPath);

    if (origPlatform) Object.defineProperty(process, 'platform', origPlatform);
  });

  it('throws when binary is not found in extracted directory', async () => {
    if (process.platform === 'win32') return;

    const extractDir = path.join(tmpDir, 'empty-extracted');
    fs.mkdirSync(extractDir);
    // Do NOT create the binary — simulate missing binary

    mockDownloadTool.mockResolvedValue('/tmp/archive.tar.gz');
    mockExtractTar.mockResolvedValue(extractDir);

    await expect(installDaemon('latest')).rejects.toThrow('nexus-daemon binary not found');
  });

  it('propagates download errors', async () => {
    mockDownloadTool.mockRejectedValue(new Error('HTTP 404'));
    await expect(installDaemon('v9.9.9')).rejects.toThrow('HTTP 404');
  });
});
