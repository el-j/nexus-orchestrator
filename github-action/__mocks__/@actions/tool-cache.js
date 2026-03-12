'use strict';

// Manual CJS mock for @actions/tool-cache v2 (pure-ESM package in newer CI setups)
// Jest runs in CommonJS mode and cannot require() the ESM dist directly.

const downloadTool = jest.fn().mockResolvedValue('/tmp/archive.tar.gz');
const extractTar  = jest.fn().mockResolvedValue('/tmp/extracted');
const extractZip  = jest.fn().mockResolvedValue('/tmp/extracted');
const find        = jest.fn().mockReturnValue('');
const cacheFile   = jest.fn().mockResolvedValue('/tmp/cached');

module.exports = {
  downloadTool,
  extractTar,
  extractZip,
  find,
  cacheFile,
};
