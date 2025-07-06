# Subagent Sandboxing Research

## Overview

This document researches sandboxing options for securing Claude subagents that run with `--dangerously-skip-permissions`. The goal is to add OS-level security boundaries while maintaining the autonomous capabilities needed for task completion.

## Current Security Challenge

The task dashboard automatically launches Claude agents when tasks move from "todo" to "doing":

```bash
claude "Review plan.md and task.json. Begin task #XX: [title]. Update task.json status to 'done' when complete, commit to branch task_XX, then exit." --dangerously-skip-permissions
```

The `--dangerously-skip-permissions` flag bypasses Claude's built-in safety checks, creating potential security risks:
- Unrestricted file system access
- Ability to execute arbitrary commands
- Network access for package installation
- System configuration modifications

## gVisor Research

### What is gVisor?

> "gVisor is an open-source Linux-compatible sandbox that runs anywhere existing container tooling does. It enables cloud-native container security and portability."

> "The core of gVisor is a kernel that runs as a normal, unprivileged process that supports most Linux system calls. This kernel is written in Go, which was chosen for its memory- and type-safety."

### How gVisor Works

> "gVisor implements the Linux API: by intercepting all sandboxed application system calls to the kernel, it protects the host from the application. In addition, gVisor also sandboxes itself from the host using Linux's isolation capabilities."

### Architecture Components

> "There are two main components of gVisor: Sentry and Gofer. Together, these components handle the processes that would normally be handled by the system kernel."

1. **Sentry**: Acts as the application kernel, implementing system calls and managing memory
2. **Gofer**: Handles file system operations, interacting with host files on behalf of the sandbox

### Security Benefits

> "Through these layers of defense, gVisor achieves true defense-in-depth while still providing VM-like performance and container-like resource efficiency."

> "Containers are not a sandbox. While containers have revolutionized how we develop, package, and deploy applications, using them to run untrusted or potentially malicious code without additional isolation is not a good idea."

### Critical Limitation: No macOS Support

> "gVisor running on MacOS could provide an environment for Mac users to develop and run Linux containers on a Mac machine without needing to install Linux via a VM or Docker for Mac etc. However, this is still an open issue (#1270 on GitHub)"

> "From the README, we now know gvisor is only meant for linux. The official documentation confirms that gVisor supports x86_64 and ARM64, and requires Linux 4.14.77+ (older Linux)."

**Conclusion**: gVisor is not viable for macOS-based task dashboard.

## macOS sandbox-exec Alternative

### Overview

> "sandbox-exec is a built-in macOS command-line utility that enables users to execute applications within a sandboxed environment."

> "In essence, it creates a secure, isolated space where applications can run with limited access to system resources – only accessing what you explicitly permit."

### How It Works

> "Sandbox profiles use a Scheme-like syntax (a LISP dialect) with parentheses grouping expressions."

Basic profile structure:
```scheme
(version 1)
(deny default)
(allow file-read-data (regex "^/usr/lib"))
(allow process-exec (literal "/usr/bin/python3"))
```

### System Usage

> "I stumbled upon the /usr/share/sandbox/ directory, which contains sandboxing profiles for a lot of services. This alone is very cool because it shows that the system actually uses this feature to harden a great deal of functionality."

### Benefits

> "The most powerful aspect of sandbox-exec is its flexibility – you can create custom security profiles tailored to specific applications and use cases, going far beyond the one-size-fits-all approach of most security tools."

### Limitations

> "The tool has virtually no official documentation so some hacker insight can come very handy."

> "This sandboxing functionality is exposed via the sandbox-exec(1) command-line utility, which unfortunately has been listed as deprecated for at least the last two major versions of macOS. However, it's used heavily by internal systems so it's unlikely go away anytime soon."

## Implementation Recommendations

### 1. Create Sandbox Profiles

Base restrictive profile for Claude agents:

```scheme
(version 1)
(deny default)

; Import basic system functionality
(import "/System/Library/Sandbox/Profiles/bsd.sb")

; Allow read/write only in designated worktree
(allow file-read* file-write*
  (subpath "/Users/aplucche/repos/cc_task_dash-subagent"))

; Allow git operations
(allow process-exec
  (literal "/usr/bin/git")
  (literal "/usr/local/bin/git"))

; Allow Claude binary
(allow process-exec
  (regex "^/Users/.*/.nvm/.*/bin/claude$"))

; Network access for Claude API only
(allow network-outbound
  (remote tcp "api.anthropic.com:443"))

; Essential system operations
(allow file-read-metadata)
(allow signal (target self))
```

### 2. Modify agent_spawn.sh

Wrap the Claude execution in sandbox-exec:

```bash
# Instead of:
claude "$PROMPT" --dangerously-skip-permissions

# Use:
sandbox-exec -f "$SANDBOX_PROFILE" claude "$PROMPT" --dangerously-skip-permissions
```

### 3. Task Security Levels

Add security classification to tasks:
- `readonly`: Analysis and research tasks
- `development`: Code modification tasks
- `system`: Tasks requiring broader system access

### 4. Monitoring

> "To see all the operations that were denied, open Applications → Utilities → Console and search for sandbox and the application name."

## Key Resources

1. **gVisor Documentation**: https://gvisor.dev/
2. **gVisor GitHub**: https://github.com/google/gvisor
3. **macOS sandbox-exec overview**: https://igorstechnoclub.com/sandbox-exec/
4. **Practical sandbox-exec guide**: https://www.karltarvas.com/macos-app-sandboxing-via-sandbox-exec/
5. **System sandbox profiles**: `/usr/share/sandbox/`

## Conclusion

While gVisor provides excellent sandboxing for Linux environments, macOS users must rely on the native `sandbox-exec` utility. Despite being marked as deprecated, sandbox-exec remains functional and is actively used by macOS system services. It provides sufficient security boundaries for constraining Claude subagents while allowing necessary operations within designated worktrees.

The combination of:
1. Git worktree isolation (already implemented)
2. macOS sandbox profiles (proposed)
3. Task-based security levels
4. Comprehensive logging

Creates a defense-in-depth approach that significantly reduces the risks of running autonomous agents with `--dangerously-skip-permissions`.

## Podman Container Alternative

### Overview

Podman is a daemonless, rootless container runtime that provides an alternative approach to sandboxing. Unlike Docker, Podman doesn't require a privileged daemon and supports rootless containers by default.

> "While 'containers are Linux,' Podman also runs on Mac and Windows, where it provides a native podman CLI and embeds a guest Linux system to launch your containers."

### macOS Architecture

> "On Mac, each Podman machine is backed by a virtual machine. Once installed, the podman command can be run directly from the Unix shell in Terminal, where it remotely communicates with the podman service running in the Machine VM."

Key points:
- Requires a Linux VM on macOS (similar to Docker Desktop)
- Managed via `podman machine` commands
- Communication overhead through SSH to VM

### Performance Overhead

Research shows:
- **GPU workloads**: "Between 74% and 80% of the Metal performance" (20-26% overhead)
- **Build operations**: Can be significantly slower than Docker (6x slower in some cases)
- **VM resource allocation** is critical for performance

Optimization tips:
```bash
podman machine init \
  --cpus 4 \
  --memory 2048 \
  --disk-size 100 \
  --now
```

### Security Features

#### 1. Rootless Containers

> "Rootless containers are containers that can be created, run, and managed by users without admin rights. They add a new security layer; even if the container engine, runtime, or orchestrator is compromised, the attacker won't gain root privilege."

> "Rootless Podman is not, and will never be, root; it's not a setuid binary, and gains no privileges when it runs."

#### 2. User Namespaces

> "Podman makes use of a user namespace to shift the UIDs and GIDs of a block of users it is given access to on the host (via the newuidmap and newgidmap executables) and your own user within the containers that Podman creates."

#### 3. Seccomp Profiles

> "The default seccomp profile provides a sane default for running containers with seccomp and disables around 44 system calls out of 300+. It is moderately protective while providing wide application compatibility."

Custom seccomp profiles can be applied:
```bash
podman run --security-opt seccomp=/path/to/profile.json image
```

#### 4. SELinux Support

Podman integrates with SELinux for mandatory access controls:
- Volume labeling with `:z` (shared) or `:Z` (private)
- `--security-opt label=disable` to disable SELinux
- `--security-opt label=nested` for SELinux modifications within container

### Claude Code Container Projects

Several community projects provide containerized Claude Code solutions:

#### 1. Claude Code Sandbox
> "Claude Code Sandbox allows you to run Claude Code in isolated Docker containers, providing a safe environment for AI-assisted development."

Features:
- Network restrictions to npm, GitHub, and Anthropic servers only
- Complete isolation from host SSH keys
- Enables `--dangerously-skip-permissions` safely

#### 2. ClaudeBox
> "Offers pre-configured language stacks (C/C++, Python, Rust, Go, etc.) with complete separation of images, settings, and data between projects."

#### 3. Security Benefits
> "The container's enhanced security measures (isolation and firewall rules) allow you to run claude --dangerously-skip-permissions to bypass permission prompts for unattended operation."

### Podman Implementation for Subagents

#### 1. Container Image Creation

Create a Dockerfile for Claude subagents:
```dockerfile
FROM fedora:latest

# Install essential tools
RUN dnf install -y git nodejs npm

# Install Claude Code CLI
RUN npm install -g @anthropic/claude-code

# Create work directory
WORKDIR /workspace

# Run as non-root user
USER 1000:1000
```

#### 2. Network Security

Restrict network access to Anthropic API only:
```bash
podman network create --internal claude-net
podman run --network claude-net \
  --add-host api.anthropic.com:$(dig +short api.anthropic.com | head -1) \
  claude-subagent
```

#### 3. Volume Mounting

Mount only the specific worktree:
```bash
podman run -v /path/to/worktree:/workspace:Z \
  --security-opt label=disable \
  claude-subagent
```

#### 4. Integration with agent_spawn.sh

Replace direct Claude execution with Podman:
```bash
podman run --rm \
  -v "$WORKTREE_DIR:/workspace:Z" \
  -e ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY" \
  --security-opt seccomp=claude-profile.json \
  claude-subagent \
  claude "$PROMPT" --dangerously-skip-permissions
```

### Pros and Cons

**Pros:**
- Strong, proven container isolation
- Rootless by default (no privilege escalation)
- Fine-grained security controls (seccomp, SELinux)
- Active community and enterprise support
- Portable solution (works across platforms)

**Cons:**
- VM overhead on macOS (20-26% performance hit)
- Additional complexity (container management)
- Resource usage (VM requires dedicated CPU/memory)
- Slower build operations compared to native
- Learning curve for container orchestration

### Comparison: Podman vs sandbox-exec

| Feature | Podman | sandbox-exec |
|---------|---------|--------------|
| Platform | Cross-platform (via VM on macOS) | macOS native |
| Performance | 20-26% overhead (VM) | Minimal overhead |
| Security Model | Linux namespaces + seccomp + SELinux | macOS sandbox profiles |
| Complexity | Higher (containers + VM) | Lower (single binary) |
| Community | Large, active | Limited, unofficial |
| Future Support | Actively developed | Deprecated but functional |
| Resource Usage | Higher (VM resources) | Minimal |

### Recommendation

For macOS users, the choice between Podman and sandbox-exec depends on priorities:

1. **Use Podman if:**
   - Maximum security isolation is required
   - Cross-platform compatibility is important
   - You're already familiar with containers
   - Performance overhead is acceptable

2. **Use sandbox-exec if:**
   - Minimal overhead is critical
   - Simplicity is preferred
   - Native macOS integration is important
   - Quick implementation is needed

Both solutions effectively mitigate the risks of `--dangerously-skip-permissions` while maintaining Claude's autonomous capabilities.