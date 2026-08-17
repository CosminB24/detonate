<h1 align="center">detonate</h1>

<p align="center">
  <strong>See what an npm package actually does when you install it.</strong>
</p>

<p align="center">
  Runs untrusted code in a disposable sandbox, watches every move it makes,<br>
  and hands you evidence instead of a guess.
</p>

---

## The problem

`npm install` executes arbitrary code on your machine. That has always been true,
and it is how most supply chain attacks land — a compromised maintainer publishes
a new version, an install hook fires, and credentials leave your laptop or your
CI runner before anyone notices.

Reading the code rarely helps. Install scripts are minified, obfuscated, or just
downloaders that fetch the real payload at runtime. Static scanners tell you what
a package *looks* like. They cannot tell you what it *does*.

So run it somewhere it cannot hurt you, and watch.

## What detonate does

```console
$ detonate npm some-package@1.2.3

  VERDICT   malicious                                        score 87/100

  FINDINGS
  ● HIGH      Read SSH private key                          T1552.004
              /root/.ssh/id_rsa   ← postinstall.js:12
  ● HIGH      Install script contacted external host        T1041
              203.0.113.7:443     ← blocked, no egress
  ● MEDIUM    Modified shell startup file                   T1546.004
              /root/.bashrc

  COVERAGE  install script ✓   module import ✓   exports ✗   network sinkholed

  4.1s · 218 events · report: ./detonate-report.json
```

Every finding points at a real recorded event. Nothing is inferred from the
source, and nothing is guessed.


## Status

Early, and honest about it: there is no CLI to install yet. The work right now is
validating the collection layer — proving the sandbox produces enough signal to
separate a clean install from a malicious one before any of the above gets built
on top of it.

## Running the current experiment

```bash
cd spike
docker build -t detonate-spike .
./run.sh
```

Produces two traces in `out/` — one from a known-good package, one from a
synthetic fixture that simulates credential theft. Both run fully offline.

> **A note on isolation.** Tracing requires `SYS_PTRACE` and a relaxed seccomp
> profile, which weakens container isolation. Treat the current sandbox as a
> research tool, not a containment boundary. Every test fixture in this repo is
> hand-written and harmless — no real malware samples, ever.

## License

Apache-2.0.
