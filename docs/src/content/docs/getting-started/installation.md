---
title: Installation
description: Install and run the Ork CLI.
---

Install the latest release:

```sh
curl -fsSL https://tryork.dev/install | sh
```

The installer detects your operating system and architecture, downloads the matching release asset from GitHub, verifies the checksum when the release publishes one, and installs `ork` into `~/.local/bin` by default.

## Requirements

- macOS or Linux
- `curl`
- `tar`
- Any tools required by the adapters you use, such as Docker, Terraform, or AWS CLI

## Options

Install a specific release:

```sh
curl -fsSL https://tryork.dev/install | ORK_VERSION=v0.1.0 sh
```

Install into a custom directory:

```sh
curl -fsSL https://tryork.dev/install | ORK_INSTALL_DIR=/usr/local/bin sh
```

If `ork` is installed into `~/.local/bin`, make sure that directory is on your `PATH`.

## Build

You can also build from source.

From the repository root:

```sh
go build -o bin/ork ./cmd/ork
```

Check the CLI:

```sh
bin/ork version
```

## Documentation Site

The docs site is a separate Starlight app:

```sh
cd docs
npm install
npm run dev
```
