# Gangaji

A Bazel build profiler with flamegraph visualization.

## Installation

```bash
go install github.com/thesayyn/gangaji/cmd/gangaji@latest
```

Or build from source:

```bash
go build -o gangaji ./cmd/gangaji/
```

## Usage

### Generate a Build Profile

```bash
# Build with profiling
bazel build //... --profile=profile.json

# Test with profiling
bazel test //... --profile=profile.json

# Include target labels for better visibility
bazel build //... --profile=profile.json --experimental_profile_include_target_label
```

### Generate a Starlark CPU Profile

```bash
bazel build //... --starlark_cpu_profile=starlark.json
```

### Generate Both Profiles

```bash
bazel build //... --profile=profile.json --starlark_cpu_profile=starlark.json
```

### View with Gangaji

**Build profile only:**
```bash
gangaji --profile=profile.json
gangaji --profile=profile.json.gz  # compressed
```

**Starlark CPU profile only:**
```bash
gangaji --starlark_cpu_profile=starlark.json
```

**Both profiles combined:**
```bash
gangaji --profile=profile.json --starlark_cpu_profile=starlark.json
```

**Additional options:**
```bash
gangaji --profile=profile.json --port=3000      # custom port
gangaji --profile=profile.json --open=false     # don't auto-open browser
```

## Features

- Interactive flamegraph visualization
- Dark/light theme support
- Search and filter actions
- Zoom and drill-down navigation
- Build optimization suggestions
- Statistics by category and mnemonic

## Example Project

See the `example/` directory for a sample Bazel project to test with.

```bash
# Generate profiles
cd example
bazel build //... --profile=profile.json --starlark_cpu_profile=starlark.json
cd ..

# View build profile only
./gangaji --profile=example/profile.json

# View starlark profile only
./gangaji --starlark_cpu_profile=example/starlark.json

# View both profiles combined
./gangaji --profile=example/profile.json --starlark_cpu_profile=example/starlark.json
```

## License

MIT
