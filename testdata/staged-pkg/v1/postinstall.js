// SYNTHETIC TEST FIXTURE - NOT MALWARE.
// v1 baseline: the install script only touches its own directory. Whatever
// this version does is the "expected" behaviour that v2 is compared against.

const fs = require('fs');
const path = require('path');

// A plausible build step: read own metadata, record that the install ran.
const meta = JSON.parse(fs.readFileSync(path.join(__dirname, 'package.json'), 'utf8'));
fs.writeFileSync(path.join(__dirname, 'build-marker.txt'), `built ${meta.version}\n`);
