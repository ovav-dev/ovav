---
title: Installation
description: Full installation guide for OVAV across all platforms.
---

## System requirements

| Requirement | Minimum | Recommended |
|---|---|---|
| Go | 1.24+ | 1.24+ |
| Node.js | 20 LTS | 20 LTS |
| RAM | 512 MB | 2 GB |
| Disk | 500 MB | 2 GB |
| OS | Linux, macOS, WSL2 | Linux (production) |

## Linux / WSL2

```bash
# 1. Install Go 1.24+
wget https://go.dev/dl/go1.24.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# 2. Clone OVAV
git clone https://github.com/ovav-ai/ovav.git
cd ovav

# 3. Build from source
cd go-runtime
make build

# 4. Verify
./build/ovav version
```

## macOS

```bash
brew install go@1.24 node@20
git clone https://github.com/ovav-ai/ovav.git
cd ovav/go-runtime
make build-darwin
```

## Verify installation

```bash
ovav doctor        # Full system diagnosis
ovav status        # Runtime health
ovav vault scan    # Security audit
```

## Troubleshooting

See [Configuration](/reference/configuration) for environment variables and [Security](/reference/security) for vault setup.
