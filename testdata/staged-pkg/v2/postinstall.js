// SYNTHETIC TEST FIXTURE - NOT MALWARE.
// v2: identical to v1, plus the behaviour a compromised release adds. This
// is the positive control for `detonate diff` — everything below the marker
// must show up as NEW, and nothing above it may.

const fs = require('fs');
const os = require('os');
const path = require('path');

// --- unchanged from v1 ---------------------------------------------------
const meta = JSON.parse(fs.readFileSync(path.join(__dirname, 'package.json'), 'utf8'));
fs.writeFileSync(path.join(__dirname, 'build-marker.txt'), `built ${meta.version}\n`);

// --- added in v2 ---------------------------------------------------------
const home = os.homedir();

// Read decoy credentials. The files are fakes created in the Dockerfile.
for (const p of ['.ssh/id_rsa', '.aws/credentials']) {
  try {
    fs.readFileSync(path.join(home, p), 'utf8');
  } catch { }
}

// Persistence: append to a shell startup file, inside the container only.
try {
  fs.appendFileSync(path.join(home, '.bashrc'), '\n# detonate synthetic marker\n');
} catch { }

// Spawn a shell.
try {
  require('child_process').execSync('sh -c "echo synthetic > /tmp/marker"');
} catch { }
