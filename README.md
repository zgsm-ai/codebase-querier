# codebase-indexer

<div align="center">

[English](./README.md) | [简体中文](./README.md)

A powerful code indexing and context retrieval service for AI programming assistants.

[![Go Report Card](https://goreportcard.com/badge/github.com/zgsm-ai/codebase-indexer)](https://goreportcard.com/report/github.com/zgsm-ai/codebase-indexer)
[![Go Reference](https://pkg.go.dev/badge/github.com/zgsm-ai/codebase-indexer.svg)](https://pkg.go.dev/github.com/zgsm-ai/codebase-indexer)
[![License](https://img.shields.io/github/license/zgsm-ai/codebase-indexer)](LICENSE)

</div>

## Overview | 项目概述

codebase-indexer is the context module of [ZGSM (ZhuGe Smart Mind) AI Programming Assistant](https://github.com/zgsm-ai/zgsm). It provides powerful codebase indexing capabilities to support semantic search and code call graph relationship retrieval for RAG (Retrieval-Augmented Generation) systems.

codebase-indexer 是诸葛神码 AI 编程助手的服务端上下文模块，提供代码库索引功能，支持 RAG 的语义检索和代码调用链图关系检索。

### Key Features | 主要特性

- 🚀 Fast and efficient codebase indexing | 快速高效的代码库索引
- 🔍 Semantic code search with embeddings | 基于向量的语义代码搜索
- 📊 Code call graph analysis and retrieval | 代码调用关系图分析与检索
- 🌐 Multi-language support | 多编程语言支持
- 🔄 Real-time index updates | 实时索引更新
- 🎯 High precision search results | 高精度搜索结果

## Requirements | 环境要求

- Go 1.24.3 or higher
- Docker
- PostgreSQL
- Redis

## Quick Start | 快速开始

### Installation | 安装

```bash
# Clone the repository
git clone https://github.com/zgsm-ai/codebase-indexer.git
cd codebase-indexer

# Install dependencies
go mod download
```

### Configuration | 配置

1. Set up PostgreSQL and Redis
2. Copy the example configuration file:
```bash
cp etc/config.example.yaml etc/config.yaml
```
3. Update the configuration with your database and Redis credentials

### Running | 运行

```bash
# Build the project
make build

# Run the service
make run
```

## Documentation | 文档

For detailed documentation, please visit our [Wiki](https://github.com/zgsm-ai/codebase-indexer/wiki).

## Architecture | 架构

The system consists of several key components:

- **Parser**: Code parsing and AST generation
- **Embedding**: Code semantic vector generation
- **CodeGraph**: Code relationship graph construction
- **Store**: Data storage and indexing
- **API**: RESTful service interface

## Contributing | 贡献指南

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

## License | 许可证

This project is licensed under the [MIT License](LICENSE).

## Acknowledgments | 致谢

This project builds upon the excellent work of:

- [Sourcegraph](https://github.com/sourcegraph) - For their pioneering work in code intelligence
- [Tree-sitter](https://github.com/tree-sitter) - For providing robust parsing capabilities

## Contact | 联系方式

- GitHub Issues: For bug reports and feature requests
- Email: [your-email@example.com]

---

⭐️ If you find this project helpful, please consider giving it a star!