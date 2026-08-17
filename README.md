# detonate

**See what an npm package actually does when you install it.**

Runs an untrusted package in a disposable sandbox, records what it touches -
processes, files, network, credentials - and reports evidence instead of a guess.

---

## Status: work in progress

No CLI, no release, nothing usable yet. This is a spike answering one question
before any real code gets written:

> **Does `strace` over `npm install` produce enough signal to tell a clean
> package from a malicious one, without drowning in noise?**

Yes, and Phase 1 builds the CLI in Go. No, and the collector changes first.

## Right now: sessions 1-2

**Session 1 - produce the traces.**
Build a container with Node and `strace`, run a known-good package
(`lodash`, zero dependencies) and a synthetic malicious fixture, capture both.
Done when `out/clean.log` and `out/evil.log` exist and aren't empty.

**Session 2 - measure the noise floor.**
Read `out/clean.log` by hand. No parser, no code. How many processes? Which
`execve` calls, and why? How many writes land outside the working directory?
How much is pure `stat` noise over `node_modules`? Answers go in `spike/NOTES.md`.

That second session is the actual deliverable. Everything downstream - which
syscalls to trace, which rules are worth writing - depends on knowing how loud
a clean install already is.

## Run it

```bash
cd spike
docker build -t detonate-spike .
./run.sh
```

Both runs are fully offline (`--network=none`); the baseline package is fetched
at image build time.

## Safety

`--cap-add=SYS_PTRACE` and `seccomp=unconfined` are needed for tracing and both
weaken container isolation. **Not a security boundary.** Everything in
`spike/testdata/` is synthetic and harmless by construction. Never commit real
malware samples here. See [`SECURITY.md`](SECURITY.md).

## More

Plan: [`docs/PHASE-0.md`](docs/PHASE-0.md) · Current state: [`NEXT.md`](NEXT.md) ·
Long-term intent: [`VISION.md`](VISION.md) · Event model: [`schema/event-v1.json`](schema/event-v1.json)

## License

Apache-2.0.