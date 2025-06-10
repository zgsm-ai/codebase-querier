# codebase-indexer

<div align="center">

[English](./README.md) | [简体中文](./README_zh.md)

强大的 AI 编程助手代码索引和上下文检索服务

[![Go Report Card](https://goreportcard.com/badge/github.com/zgsm-ai/codebase-indexer)](https://goreportcard.com/report/github.com/zgsm-ai/codebase-indexer)
[![Go Reference](https://pkg.go.dev/badge/github.com/zgsm-ai/codebase-indexer.svg)](https://pkg.go.dev/github.com/zgsm-ai/codebase-indexer)
[![License](https://img.shields.io/github/license/zgsm-ai/codebase-indexer)](LICENSE)

</div>

## 项目概述

codebase-indexer 是诸葛神码 AI 编程助手的服务端上下文模块，提供代码库索引功能，支持 RAG 的语义检索和代码调用链图关系检索。

### 主要特性

- 🔍 基于向量的语义代码搜索
- 📊 代码调用关系图分析与检索
- 🌐 多编程语言支持

## 环境要求

- Go 1.24.3 或更高版本
- Docker
- PostgreSQL
- Redis

## 快速开始

### 安装

```bash
# 克隆仓库
git clone https://github.com/zgsm-ai/codebase-indexer.git
cd codebase-indexer

# 安装依赖
go mod tidy
```

### 配置

 设置 PostgreSQL 、 Redis、向量数据库（weaviate）等等。
```bash
vim etc/config.yaml
```


### 运行

```bash
# 构建项目
make build

```

## 架构

系统由以下几个关键组件组成：

- **Parser**: 代码解析和 AST 生成
- **Embedding**: 代码语义向量生成
- **CodeGraph**: 代码关系图构建
- **Store**: 数据存储和索引
- **API**: RESTful 服务接口

## 许可证

本项目采用 [Apache 2.0 许可证](LICENSE)。

## 致谢

本项目基于以下优秀项目的工作：

- [Sourcegraph](https://github.com/sourcegraph) - 在代码智能方面的开创性工作
- [Tree-sitter](https://github.com/tree-sitter) - 提供强大的解析功能