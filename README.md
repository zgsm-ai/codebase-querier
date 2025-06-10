# codebase-indexer

<div align="center">

[English](./README.md) | [简体中文](./README_zh.md)

A powerful code indexing and context retrieval service for AI programming assistants.

[![Go Report Card](https://goreportcard.com/badge/github.com/zgsm-ai/codebase-indexer)](https://goreportcard.com/report/github.com/zgsm-ai/codebase-indexer)
[![Go Reference](https://pkg.go.dev/badge/github.com/zgsm-ai/codebase-indexer.svg)](https://pkg.go.dev/github.com/zgsm-ai/codebase-indexer)
[![License](https://img.shields.io/github/license/zgsm-ai/codebase-indexer)](LICENSE)

</div>

## Overview

codebase-indexer is the context module of [ZGSM (ZhuGe Smart Mind) AI Programming Assistant](https://github.com/zgsm-ai/zgsm). It provides powerful codebase indexing capabilities to support semantic search and code call graph relationship retrieval for RAG (Retrieval-Augmented Generation) systems.

### Key Features

- 🔍 Semantic code search with embeddings
- 📊 Code call graph analysis and retrieval
- 🌐 Multi-language support

## Requirements

- Go 1.24.3 or higher
- Docker
- PostgreSQL
- Redis

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/zgsm-ai/codebase-indexer.git
cd codebase-indexer

# Install dependencies
go mod tidy
```

### Configuration

1. Set up PostgreSQL 、 Redis、vector, etc.
```bash
vim etc/config.yaml
```
3. Update the configuration with your database and Redis credentials

### Running

```bash
# Build the project
make build
```

## Architecture

The system consists of several key components:

- **Parser**: Code parsing and AST generation
- **Embedding**: Code semantic vector generation
- **CodeGraph**: Code relationship graph construction
- **Store**: Data storage and indexing
- **API**: RESTful service interface

## License

This project is licensed under the [Apache 2.0 License](LICENSE).

## Acknowledgments

This project builds upon the excellent work of:

- [Sourcegraph](https://github.com/sourcegraph) - For their pioneering work in code intelligence
- [Tree-sitter](https://github.com/tree-sitter) - For providing robust parsing capabilities