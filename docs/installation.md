# Installation

[Home](README.md)

---

## npm (recommended)

Best for frontend projects. Installs a platform-specific binary as a dev dependency:

```sh
npm install -D @ridakaddir/mockr
```

Use it via `npx` or in `package.json` scripts:

```sh
npx mockr --init
```

```json
{
  "scripts": {
    "mock": "mockr --config ./mocks --target https://api.example.com"
  }
}
```

---

## Binary download

Download a pre-built binary from the [latest release](https://github.com/ridakaddir/mockr/releases):

**macOS Apple Silicon:**

```sh
curl -L https://github.com/ridakaddir/mockr/releases/latest/download/mockr_darwin_arm64.tar.gz | tar xz
sudo mv mockr /usr/local/bin/
```

**macOS Intel:**

```sh
curl -L https://github.com/ridakaddir/mockr/releases/latest/download/mockr_darwin_amd64.tar.gz | tar xz
sudo mv mockr /usr/local/bin/
```

**Linux x86-64:**

```sh
curl -L https://github.com/ridakaddir/mockr/releases/latest/download/mockr_linux_amd64.tar.gz | tar xz
sudo mv mockr /usr/local/bin/
```

---

## Go install

Requires Go 1.25+:

```sh
go install github.com/ridakaddir/mockr@latest
```

---

## Build from source

```sh
git clone https://github.com/ridakaddir/mockr.git
cd mockr
task build          # requires Task (https://taskfile.dev)
# or
go build -o mockr .
```

---

## Verify installation

```sh
mockr --version
mockr --help
```

---

**Next:** [Quick Start](quick-start.md)
