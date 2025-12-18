# Gangaji Example Project

A simple C++ Bazel project to demonstrate profile generation for Gangaji.

## Project Structure

```
example/
├── MODULE.bazel
├── BUILD.bazel
├── lib/
│   ├── BUILD.bazel
│   ├── math.{h,cc}
│   ├── strings.{h,cc}
│   └── utils.{h,cc}
└── src/
    ├── BUILD.bazel
    ├── main.cc
    ├── math_test.cc
    └── strings_test.cc
```

## Generate Build Profile

```bash
cd example

# Build everything with profiling enabled
bazel build //... --profile=profile.json

# Generate compressed profile
bazel build //... --profile=profile.json.gz
```

## Run Tests with Profile

```bash
cd example

# Run all tests with profiling
bazel test //... --profile=profile.json

# Run specific test
bazel test //src:math_test --profile=profile.json

# Run tests with both profiles
bazel test //... --profile=profile.json --starlark_cpu_profile=starlark.json
```

## Generate Starlark CPU Profile

```bash
cd example

# Build with Starlark CPU profiling
bazel build //... --starlark_cpu_profile=starlark.json
```

## Generate Both Profiles

```bash
cd example

bazel build //... --profile=profile.json --starlark_cpu_profile=starlark.json
```

## View with Gangaji

From the gangaji repo root (`cd ..`):

**Build profile only:**
```bash
./gangaji --profile=example/profile.json
```

**Starlark CPU profile only:**
```bash
./gangaji --starlark_cpu_profile=example/starlark.json
```

**Both profiles combined:**
```bash
./gangaji --profile=example/profile.json --starlark_cpu_profile=example/starlark.json
```

**Additional options:**
```bash
./gangaji --profile=example/profile.json --port=3000  # custom port
```

## Targets

- `//lib:math` - Math utility library
- `//lib:strings` - String utility library
- `//lib:utils` - Combined utilities (depends on math and strings)
- `//src:app` - Main application binary
- `//src:math_test` - Math library tests
- `//src:strings_test` - String library tests
- `//src:generate_config` - Genrule to generate config.h
- `//src:generate_data` - Genrule to generate data.txt
