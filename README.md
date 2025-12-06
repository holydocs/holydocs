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

[Here](internal/adapters/secondary/docs/testdata/expected/README.md) you can see at generated markdown documentation based on [example specifications](internal/adapters/secondary/schema/testdata). 

The resulting overview diagram looks like this:
![Overview Diagram](internal/adapters/secondary/docs/testdata/expected/diagrams/overview.svg)

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
   holydocs gen-docs
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
# Generate documentation from a directory containing spec files to docs dir
holydocs gen-docs
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
export HOLYDOCS_OUTPUT_DIR="./docs"
export HOLYDOCS_OUTPUT_GLOBAL_NAME="Internal Services"

# Input configuration
export HOLYDOCS_INPUT_DIR="./specs"
export HOLYDOCS_INPUT_ASYNCAPI_FILES="specs/analytics.asyncapi.yaml,specs/campaign.asyncapi.yaml"
export HOLYDOCS_INPUT_SERVICE_FILES="specs/analytics.servicefile.yml,specs/campaign.servicefile.yaml"

# Diagram configuration (D2)
export HOLYDOCS_DIAGRAM_D2_PAD="64"
export HOLYDOCS_DIAGRAM_D2_THEME="0"
export HOLYDOCS_DIAGRAM_D2_SKETCH="false"
export HOLYDOCS_DIAGRAM_D2_FONT="SourceSansPro"
export HOLYDOCS_DIAGRAM_D2_LAYOUT="elk"
```

#### Configuration File

Create a `holydocs.yaml` file in your project root or specify a custom path:

```yaml
# Output configuration
output:
  title: "My Service Architecture Documentation"
  dir: "./docs"
  global_name: "Internal Services"

# Input configuration
input:
  dir: "./specs"  # Directory to scan for specifications
  asyncapi_files: ["specs/analytics.asyncapi.yaml", "specs/campaign.asyncapi.yaml"]
  service_files: ["specs/analytics.servicefile.yml", "specs/campaign.servicefile.yaml"]

# Diagram configuration
diagram:
  d2:
    # Render settings
    pad: 64                    # Padding around diagrams in pixels
    theme: 0                   # Theme ID (0 for default, -1 for dark)
    sketch: false              # Enable sketch mode for hand-drawn appearance
    
    # Font and layout settings
    font: "SourceSansPro"      # Font family (SourceSansPro, SourceCodePro, HandDrawn)
    layout: "elk"              # Layout engine (dagre, elk)

# Documentation configuration
documentation:
  overview:
    description:
      content: "# Custom Overview\nThis is custom content for the overview section."
      # file_path: "./docs/overview.md"  # Alternative: load from file
  
  services:
    analytics:
      summary:
        content: "Analytics service handles data processing and insights."
      description:
        content: "Detailed description of the analytics service..."
        # file_path: "./docs/analytics-service.md"  # Alternative: load from file
  
  systems:
    notification-system:
      summary:
        content: "Notification system manages user communications."
      description:
        file_path: "./docs/notification-system.md"
```

#### Configuration Options

**Input Configuration:**
- `input.dir`: Directory to scan for AsyncAPI and ServiceFile specifications
- `input.asyncapi_files`: Explicit list of AsyncAPI specification files
- `input.service_files`: Explicit list of ServiceFile specification files

**Output Configuration:**
- `output.dir`: Directory where generated documentation will be saved
- `output.title`: Title for the generated documentation
- `output.global_name`: Name used for grouping internal services in diagrams

**Diagram Configuration (D2):**
- `diagram.d2.pad`: Padding around diagrams in pixels (default: 64)
- `diagram.d2.theme`: Theme ID for diagrams (0 for default, -1 for dark)
- `diagram.d2.sketch`: Enable sketch mode for hand-drawn appearance
- `diagram.d2.font`: Font family for diagram text (SourceSansPro, SourceCodePro, HandDrawn)
- `diagram.d2.layout`: Layout engine for diagram arrangement (dagre, elk)

**Documentation Configuration:**
- `documentation.overview.description`: Custom markdown content for the overview section
- `documentation.services.{service_name}.summary`: Summary text for specific services
- `documentation.services.{service_name}.description`: Detailed description for specific services
- `documentation.systems.{system_name}.summary`: Summary text for specific systems
- `documentation.systems.{system_name}.description`: Detailed description for specific systems

**Markdown Content:**
Each markdown field supports two formats:
- `content`: Raw markdown content as a string
- `file_path`: Path to a markdown file to load content from

You cannot specify both `content` and `file_path` for the same field.

Full example can be found [here](holydocs.example.yaml).

## Roadmap

HolyDOCs is actively developed with the following features planned:

### Documentation Generation
- [ ] **Single Page / Multi Page Documentation**: Support for both single-page applications and multi-page documentation sites
- [x] **Changelog Integration**: Automatic changelog generation similar to MessageFlow, tracking changes in service specifications over time

### Extensibility
- [x] **Manual Extensibility**: Configuration file support for customizing documentation generation
- [x] **Markdown Integration**: Support for custom markdown content and templates

### Deployment & Hosting
- [ ] **Static HTML Generation**: Generate static HTML files for easy deployment
- [ ] **On-the-fly Serving**: Built-in web server for serving documentation dynamically

### Service Artifacts Support
- [ ] **Multiple Specification Formats**: Support for various service artifacts including AsyncAPI, OpenAPI/Swagger, gRPC, etc.

### Architecture Documentation
- [ ] **Architecture Decision Records (ADRs)**: Support for documenting architectural decisions
- [ ] **Use Cases Documentation**: Structured documentation of system use cases and scenarios
