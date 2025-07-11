# codebase-querier

<div align="center">

[English](./README.md) | [简体中文](./README_zh.md)

A proxy connect to codebase-indexer which running on client.

[![Go Report Card](https://goreportcard.com/badge/github.com/zgsm-ai/codebase-querier)](https://goreportcard.com/report/github.com/zgsm-ai/codebase-querier)
[![Go Reference](https://pkg.go.dev/badge/github.com/zgsm-ai/codebase-querier.svg)](https://pkg.go.dev/github.com/zgsm-ai/codebase-querier)
[![License](https://img.shields.io/github/license/zgsm-ai/codebase-querier)](LICENSE)

</div>

## Overview

codebase-querier is one module of [ZGSM (ZhuGe Smart Mind) AI Programming Assistant](https://github.com/zgsm-ai/zgsm) which running on backend. It is a proxy connect to codebase-indexer which running on client to search code call graph for AI programming systems.


## Requirements

- Go 1.24.3 or higher
- Docker

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/zgsm-ai/codebase-querier.git
cd codebase-querier

# Install dependencies
go mod tidy
```

### Configuration

```bash
vim etc/config.yaml
```
3. Update the configuration with your database and Redis credentials

### Running

```bash
# Build the project
make build
```

## License

This project is licensed under the [Apache 2.0 License](LICENSE).
