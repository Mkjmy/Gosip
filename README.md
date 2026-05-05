# GOSIP OS // SYSTEM_V3.0

A custom package manager for Linux built primarily because standard tools were too efficient and didn't look enough like a 90s hacker movie.

## Technical Details

1. Hybrid UI: An inline TUI menu that lives in your terminal scrollback. During installation, it switches to an Alternate Screen Buffer because watching progress bars is a full-screen experience.
2. Logic Audit: A pattern scanner that looks for "unfortunate" commands like `rm -rf /` or `sudo` hidden in source code. It also flags binary blobs because real developers build from source (or so I tell myself).
3. Commit Pinning: Since trust is a rare commodity, this locks apps to specific Git hashes. Even if the author decides to "improve" their code with malware later, Gosip will stay on the audited version.
4. Trust System: A local whitelist for GitHub authors. Verified developers get a [★] badge, mostly to make their apps feel more expensive.
5. Snapshots: Export the entire system state to JSON. Useful for when you break your OS and need to pretend it never happened on a fresh install.
6. System Dashboard: A status report that calculates disk usage and checks if I've actually remembered to add the bin directory to $PATH.

## Installation

```bash
go build -o gosip cmd/gosip/*.go
./gosip init
```

The init command handles the shell configuration. Read the code before running it if you actually care about your .bashrc.

## Usage

- `gosip`: Enter the interactive menu and scroll through things you probably don't need.
- `gosip search <name>`: Find tools in the void.
- `gosip install <name>`: Deployment with mandatory auditing.
- `gosip install <name> --auto`: For the brave and the lazy. Bypasses all confirmation prompts.
- `gosip dump <file>`: Capture the current state of your system's clutter.
- `gosip restore <file>`: Recreate the same clutter on a new machine.

## Contributing

If you have built a tool that is actually useful (or just looks cool), you can submit it to the community registry.

### Submission Process

1. Open a new issue in the `gosip-registry` repository.
2. Apply the label `app-submission`.
3. Paste the following JSON template into the issue body:

```json
{
  "apps": [
    {
      "name": "your-app-name",
      "repo": "username/repo-name",
      "version": "v1.0.0",
      "description": "Short but descriptive text.",
      "type": "git-config",
      "target_path": "~/.your-app-path",
      "dependencies": ["git", "any-other-tool"],
      "post_install": "any-build-commands-here"
    }
  ]
}
```

### The Reality of Your Submission

- **Community Tier**: Your app will be added to the `community` registry. 
- **User Trust**: Most users will likely ignore your tool because it is unverified and potentially dangerous.
- **Verification**: If you want your app to be moved to the "Official" registry, I must manually audit your code.
- **The Catch**: I am exceptionally lazy. Unless your tool is revolutionary or I happen to be bored, it will likely remain in the community graveyard indefinitely.

## Philosophy

Gosip is a personal layer for managing specific workflows. It doesn't aim to compete with system package managers; it just aims to look better while doing less.

Disclaimer: Provided as-is. If it breaks, you own both pieces.

---
*Developed by Mkjmy.*
