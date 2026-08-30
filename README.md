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
and it is how most supply chain attacks land, a compromised maintainer publishes
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


## Diffing two versions

Most supply chain attacks are not a malicious package. They are a *new version*
of a package you already trust. So the more useful question is not "is this
dangerous" but "what does this release do that the last one didn't":

```console
$ detonate diff npm some-package@1.2.3 some-package@1.2.4

  === some-package@1.2.3 → some-package@1.2.4

  from:        inconclusive   23 behaviours
  to:          malicious      28 behaviours

  NEW (5)
    file.read        /root/.aws/credentials
    file.read        /root/.ssh/id_rsa
    file.write       /root/.bashrc
    file.write       /tmp/marker
    process.exec     /bin/sh
  REMOVED (0)
  unchanged:   23

  NEW FINDING [high] Read credential file (2 events)
  NEW FINDING [high] Modified shell startup file (1 events)
```

Each version is detonated more than once, and only the behaviours every run
agreed on are compared, anything that varies between runs of the same version
is noise, and would otherwise be reported as a change between versions.

Static version diffing exists. Dynamic behavioural diffing, as far as we can
tell, does not.

## Status

Early, and honest about it. The output above is real, but the coverage behind it
is narrow: install scripts only, one detonation per run, exported functions never
invoked. That is why there is no `benign` verdict, with this much coverage, no
findings is not evidence of safety, so the tool says `inconclusive` instead.

## Running it

```bash
docker build -t detonate-spike ./spike

go run ./cmd/detonate npm express@5.2.1
go run ./cmd/detonate npm ./spike/testdata/evil-pkg
go run ./cmd/detonate diff npm ./testdata/staged-pkg/v1 ./testdata/staged-pkg/v2
```

Writes a trace and a JSON report to `out/`. The package is downloaded with the
network on and no code running, then executed with the network off, so the only
thing that ever touches the internet is the download.

> **A note on isolation.** Tracing requires `SYS_PTRACE` and a relaxed seccomp
> profile, which weakens container isolation. Treat the current sandbox as a
> research tool, not a containment boundary. Every test fixture in this repo is
> hand-written and harmless, no real malware samples, ever.

## License

Apache-2.0.
