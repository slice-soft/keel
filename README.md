<img src="https://cdn.slicesoft.dev/boat.svg" width="400" />

# Keel

Keel is a Go framework for building REST APIs with modular 
architecture, automatic OpenAPI, and built-in validation.

[![CI](https://github.com/slice-soft/ss-keel-core/actions/workflows/ci.yml/badge.svg)](https://github.com/slice-soft/ss-keel-core/actions)
![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)
[![Go Report Card](https://goreportcard.com/badge/github.com/slice-soft/ss-keel-core)](https://goreportcard.com/report/github.com/slice-soft/ss-keel-core)
[![Go Reference](https://pkg.go.dev/badge/github.com/slice-soft/ss-keel-core.svg)](https://pkg.go.dev/github.com/slice-soft/ss-keel-core)
![License](https://img.shields.io/badge/License-MIT-green)
![Made in Colombia](https://img.shields.io/badge/Made%20in-Colombia-FCD116?labelColor=003893)


## Features

- 🚀 **Quick Project Scaffolding** - Create new projects with best practices built-in
- 🧩 **Module Generation** - Generate controllers, services, repositories, and more
- 🔄 **Live Reload** - Built-in development server with hot reload
- ⚡ **Fast & Lightweight** - Written in Go, optimized for performance
- 📦 **Auto-updates** - Stay up-to-date with the latest features

## Installation

```bash
# Using Go install
go install github.com/slice-soft/keel@latest

# Or download from releases
curl -sSL https://github.com/slice-soft/keel/releases/latest/download/keel-$(uname -s)-$(uname -m) -o keel
chmod +x keel
sudo mv keel /usr/local/bin/
```

## Quick Start

### Create a new project

```bash
keel new my-app
cd my-app
```

### Generate a module

```bash
keel generate module users
```

This creates:
- `users/module.go` - Module definition
- `users/controller.go` - HTTP handlers
- `users/service.go` - Business logic
- `users/repository.go` - Data access layer

### Run development server

```bash
keel run
```

## Commands

### `keel new <project-name>`

Create a new Keel project with the complete structure:

```bash
keel new my-api
```

### `keel generate [type] [name]`

Generate different components:

```bash
# Generate a complete module
keel generate module posts

# Generate individual components
keel generate controller posts
keel generate service posts
keel generate repository posts
keel generate middleware auth
keel generate guard admin
keel generate dto create-user
keel generate crud products
```

### `keel run`

Start the development server with live reload:

```bash
keel run
```

### `keel upgrade`

Upgrade Keel CLI to the latest version:

```bash
keel upgrade
```

### `keel version`

Show the installed version:

```bash
keel version
```

## Project Structure

```
my-app/
├── main.go           # Application entry point
├── go.mod            # Go module definition
├── .env              # Environment variables
├── air.toml          # Live reload configuration
└── modules/
    └── users/
        ├── module.go      # Module registration
        ├── controller.go  # HTTP handlers
        ├── service.go     # Business logic
        └── repository.go  # Data layer
```

## Configuration

Keel projects use a `keel.toml` file for configuration:

```toml
[app]
name = "my-app"
version = "0.1.0"

[server]
port = 8080
host = "localhost"
```

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for setup and repository-specific rules.
The base workflow, commit conventions, and community standards live in [ss-community](https://github.com/slice-soft/ss-community/blob/main/CONTRIBUTING.md).

## Community

| Document | |
|---|---|
| [CONTRIBUTING.md](https://github.com/slice-soft/ss-community/blob/main/CONTRIBUTING.md) | Workflow, commit conventions, and PR guidelines |
| [GOVERNANCE.md](https://github.com/slice-soft/ss-community/blob/main/GOVERNANCE.md) | Decision-making, roles, and release process |
| [CODE_OF_CONDUCT.md](https://github.com/slice-soft/ss-community/blob/main/CODE_OF_CONDUCT.md) | Community standards |
| [VERSIONING.md](https://github.com/slice-soft/ss-community/blob/main/VERSIONING.md) | SemVer policy and breaking changes |
| [SECURITY.md](https://github.com/slice-soft/ss-community/blob/main/SECURITY.md) | How to report vulnerabilities |
| [MAINTAINERS.md](https://github.com/slice-soft/ss-community/blob/main/MAINTAINERS.md) | Active maintainers |

## License

MIT License - see [LICENSE](LICENSE) for details.

## Links

- Website: [keel-go.dev](https://keel-go.dev)
- GitHub: [github.com/slice-soft/ss-keel-cli](https://github.com/slice-soft/ss-keel-cli)
- Documentation: [docs.keel-go.dev](https://docs.keel-go.dev)

---

Made by [SliceSoft](https://slicesoft.dev) — Colombia 💙
