// Mocha entry point loaded inside the VS Code extension host.
const path = require('node:path');
const fs = require('node:fs');
const Mocha = require('mocha');

exports.run = function run() {
  const mocha = new Mocha({ ui: 'tdd', color: true, timeout: 30000 });
  for (const f of fs.readdirSync(__dirname).filter((n) => n.endsWith('.test.cjs'))) {
    mocha.addFile(path.join(__dirname, f));
  }
  return new Promise((resolve, reject) => {
    mocha.run((failures) => (failures ? reject(new Error(`${failures} tests failed`)) : resolve()));
  });
};
