# testdata

Fixtures the tool is verified against. Unlike `spike/`, this directory is not
disposable.

Everything here is **hand-written and synthetic**. No real malware, ever.
The fixtures read decoy credentials created in `spike/Dockerfile`, write only
inside the container, and never open a connection to anything but `127.0.0.1`.

## staged-pkg

Two versions of the same package, and the positive control for
`detonate diff`. Both declare the name `staged-pkg`, so they install to the
same path and their behaviour sets are directly comparable — different
directory names would have shown up as spurious differences.

| | postinstall does |
|---|---|
| `v1` (1.0.0) | reads its own `package.json`, writes `build-marker.txt` |
| `v2` (2.0.0) | the same, **plus** reads decoy credentials, appends to `.bashrc`, spawns a shell |

```bash
go run ./cmd/detonate diff npm ./testdata/staged-pkg/v1 ./testdata/staged-pkg/v2
```

Expected: exactly five `NEW` behaviours — both credential reads, the `.bashrc`
write, the `/tmp/marker` write and the shell spawn — nothing `REMOVED`, and the
`build-marker.txt` write that both versions do staying under `unchanged`.
Two new findings fire, and the verdict goes `suspicious` → `malicious`.

v1 is already `suspicious` rather than `inconclusive`, and that is not the
fixture's doing: npm runs *any* install script through `sh -c`, so the
`SHELL_SPAWN` rule fires on every package that has one. The diff is what makes
this bearable — the rule fires on both sides, so it is not reported as new.
