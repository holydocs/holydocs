# HolyDOCs

[![Go Report Card](https://goreportcard.com/badge/github.com/holydocs/holydocs)](https://goreportcard.com/report/github.com/holydocs/holydocs)
[![GoDoc](https://godoc.org/github.com/holydocs/holydocs?status.svg)](https://godoc.org/github.com/holydocs/holydocs)

HolyDOCs is a comprehensive documentation generation tool for microservices architectures. It serves as an umbrella project that combines the power of [MessageFlow](https://github.com/holydocs/messageflow) for AsyncAPI specifications and [ServiceFile](https://github.com/holydocs/servicefile) for service definitions to create unified, interactive documentation for complex service ecosystems.

## Key Features

- **Unified Documentation**: Combines service definitions and message flows into cohesive documentation
- **Interactive Diagrams**: Generates D2-based diagrams for service relationships, system overviews, and message flows
- **System Architecture Views**: Provides both high-level system overviews and detailed service relationships
- **Message Flow Visualization**: Integrates with MessageFlow for comprehensive message flow documentation

## Quickstart

[Here](internal/docs/testdata/expected/README.md) you can see at generated markdown documentation based on [example specifications](pkg/schema/testdata). 

The resulting overview diagram looks like this:
![Overview Diagram](internal/docs/testdata/expected//diagrams/overview.svg)

### Prerequisites

You'll need:
- **ServiceFile specifications** (`.servicefile.yaml`) defining your services, their relationships, and metadata
- **AsyncAPI specifications** (`.asyncapi.yaml`) describing message flows and channels

### Generate Documentation

1. **Install HolyDOCs**:
   ```bash
   go install github.com/holydocs/holydocs/cmd/holydocs@latest
   ```

2. **Prepare your specifications**:
   Create a directory with your ServiceFile and AsyncAPI specifications:
   ```
   specs/
   ├── analytics.servicefile.yml
   ├── analytics.asyncapi.yaml
   ├── campaign.servicefile.yaml
   ├── campaign.asyncapi.yaml
   ├── mailer.servicefile.yml
   ├── mailer.asyncapi.yaml
   ├── notification.servicefile.yaml
   ├── notification.asyncapi.yaml
   ├── reports.servicefile.yml
   ├── reports.asyncapi.yaml
   ├── user.servicefile.yaml
   └── user.asyncapi.yaml
   ```

3. **Generate documentation**:
   ```bash
   holydocs gen-docs --dir ./specs --output ./docs --title "My Service Architecture"
   ```

4. **View the results**:
   Open `./docs/README.md` in your browser or markdown viewer to see your generated documentation.


## Installation

### Using Go Binary

Install the binary directly:

```bash
go install github.com/holydocs/holydocs/cmd/holydocs@latest
```

### Using Docker

Pull and run the latest version:

```bash
# Pull the image
docker pull ghcr.io/holydocs/holydocs:latest

# Generate documentation
docker run --rm -v $(pwd):/work -w /work ghcr.io/holydocs/holydocs:latest gen-docs --dir ./specs --output ./docs
```

## Usage

### Generate Documentation

The `gen-docs` command generates comprehensive markdown documentation from ServiceFile and AsyncAPI specifications:

```bash
# Generate documentation from a directory containing spec files
holydocs gen-docs --dir ./specs --output ./docs

# Generate with custom title and global name
holydocs gen-docs --dir ./specs --output ./docs --title "My Service Architecture" --global-name "Internal Services"

# Specify individual files
holydocs gen-docs --service-files "service1.servicefile.yaml,service2.servicefile.yaml" --asyncapi-files "service1.asyncapi.yaml,service2.asyncapi.yaml" --output ./docs
```

### Command Options

- `--config`: Path to YAML configuration file

### Configuration

HolyDOCs supports flexible configuration through multiple sources with the following priority order:

1. **Environment variables**
2. **YAML configuration file** (lowest priority)

#### Environment

All environment variables use the `HOLYDOCS_` prefix:

```bash
# Output configuration
export HOLYDOCS_OUTPUT_TITLE="My Service Architecture Documentation"
export HOLYDOCS_OUTPUT_DIRECTORY="./docs"
export HOLYDOCS_OUTPUT_GLOBAL_NAME="Internal Services"

# Input configuration
export HOLYDOCS_INPUT_DIRECTORY="./specs"
```

#### Configuration File

Create a `holydocs.yaml` file in your project root or specify a custom path:

```yaml
# Output configuration
output:
  title: "My Service Architecture Documentation"
  directory: "./docs"
  global_name: "Internal Services"

# Input configuration
input:
  dir: "./specs"  # Directory to scan for specifications
  # asyncapi_files: ["specs/analytics.asyncapi.yaml"]
  # service_files: ["specs/analytics.servicefile.yml"]
```

Full example can be found [here](holydocs.example.yaml).

## Roadmap

HolyDOCs is actively developed with the following features planned:

### Documentation Generation
- [ ] **Single Page / Multi Page Documentation**: Support for both single-page applications and multi-page documentation sites
- [x] **Changelog Integration**: Automatic changelog generation similar to MessageFlow, tracking changes in service specifications over time

### Extensibility
- [x] **Manual Extensibility**: Configuration file support for customizing documentation generation
- [] **Markdown Integration**: Support for custom markdown content and templates

### Deployment & Hosting
- [ ] **Static HTML Generation**: Generate static HTML files for easy deployment
- [ ] **On-the-fly Serving**: Built-in web server for serving documentation dynamically

### Service Artifacts Support
- [ ] **Multiple Specification Formats**: Support for various service artifacts including AsyncAPI, OpenAPI/Swagger, gRPC, etc.

### Architecture Documentation
- [ ] **Architecture Decision Records (ADRs)**: Support for documenting architectural decisions
- [ ] **Use Cases Documentation**: Structured documentation of system use cases and scenarios
