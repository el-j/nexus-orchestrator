'use strict';

// Manual CJS mock for @actions/core v3 (pure-ESM package)

const outputs = {};
const failures = [];
const warnings = [];
const infos = [];

function getInput(name) { return process.env[`INPUT_${name.toUpperCase().replace(/ /g, '_')}`] || ''; }
function setOutput(name, value) { outputs[name] = value; }
function setFailed(message) { failures.push(message); }
function warning(message) { warnings.push(message); }
function info(message) { infos.push(message); }
function startGroup(name) { void name; }
function endGroup() {}
function debug(message) { void message; }
function error(message) { void message; }
function exportVariable(name, val) { process.env[name] = val; }
function addPath(p) { void p; }
function saveState(name, value) { void name; void value; }
function getState(name) { void name; return ''; }
function isDebug() { return false; }

module.exports = {
  getInput,
  setOutput,
  setFailed,
  warning,
  info,
  startGroup,
  endGroup,
  debug,
  error,
  exportVariable,
  addPath,
  saveState,
  getState,
  isDebug,
  // expose test helpers
  _outputs: outputs,
  _failures: failures,
  _warnings: warnings,
  _infos: infos,
};
