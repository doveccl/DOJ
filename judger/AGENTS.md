# Judger Rules

`judger/` contains host-side judger worker and container orchestration. `judger/runner` contains the container-side runner. The real judger is Linux-only.

## Boundaries

- Judger talks only to the server API.
- PostgreSQL, Redis/Valkey, and object storage are server-only dependencies.
- The outer Judger container is only a deployment wrapper, not a security boundary. It intentionally runs privileged with the host PID/cgroup namespaces and Docker socket; do not trade orchestration correctness for capability-based hardening there.
- Runner language containers and per-case cgroups are the user-code isolation boundary. Keep their network, capability, filesystem, identity, process, and resource restrictions intact.
- Privileged mode with the host cgroup namespace exposes the writable host cgroup hierarchy; do not add an explicit `/sys/fs/cgroup` bind unless a supported Docker engine proves it necessary.
- Language config comes from the server as a source file name, fixed image, compile command, and run command. User source is written into an isolated source dir; host-side judger compiles in a short-lived container that sees only source and an empty output dir, then runner runs cases in the fixed language container.
- Server sends resource limits to judger in KB. Judger reports memory in KB.

## Execution Model

- A submission may compile once in a short-lived language container, then runs cases in one reused language container.
- Multiple cases reuse the container.
- Each case starts fresh JudgeProgram and UserProgram processes.
- Builtin judge follows the testlib checker model: run UserProgram with the case input, write user output to a private file, then invoke the builtin checker as `doj runner judge [--mode=default|strict] <input> <output> <answer> <result>`.
- Custom judge follows the testlib interactor model and is always invoked as `<judge> <input> <transcript> <answer> <result>` with this pipe model:
  - JudgeProgram stdout -> UserProgram stdin
  - UserProgram stdout -> JudgeProgram stdin
- Judge results are read from testlib-style exit codes and the plain result file. Do not keep legacy JSON/env protocols.
- Control protocol uses Unix domain socket + Go gob, not stdout/stderr.

## Isolation

- cgroup v2 is the source of truth for per-case memory.
- UserProgram must enter the case cgroup before exec.
- JudgeProgram and UserProgram use different child processes, UIDs/GIDs, and process groups.
- Runner internal fds, answer files, result files, judge binaries, and control sockets must not be inherited by UserProgram.
- Every case must kill process groups, reap children, close pipes/fds, clean temp files, and clean remaining cgroup processes.

## Tests

- Keep tests for normal judging, custom interactor judging, Quine-style interaction, output flood, timeout, compile limits, case isolation, fd inheritance, Docker, and cgroup behavior.
- Docker and cgroup tests may skip when prerequisites are missing, but Linux validation must run them on a real Linux host before treating the judger as accepted.
- Non-Linux stubs are only for compiling server/API development on non-Linux hosts; they are not judger support.
