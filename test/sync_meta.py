#!/usr/bin/env python3
import os
import sys
import argparse
import json
import hashlib
from datetime import datetime

def is_hidden(path):
    """判断文件或目录是否为隐藏的"""
    return os.path.basename(path).startswith('.')

def traverse_directory(root_dir):
    """递归遍历目录，跳过隐藏文件和目录"""
    if not os.path.exists(root_dir):
        raise Exception(f"root dir {root_dir} not exists")
    script_file = sys.argv[0]

    file_list = {}
    for dirpath, dirnames, filenames in os.walk(root_dir):
        # 跳过隐藏目录
        dirnames[:] = [d for d in dirnames if not is_hidden(d)]

        for filename in filenames:
            # 跳过隐藏文件
            if is_hidden(filename):
                continue
            # 跳过本脚本
            if filename == script_file:
                continue
            # 获取文件的相对路径（相对于 root_dir）
            rel_path = os.path.relpath(os.path.join(dirpath, filename), root_dir)
            # 统一使用正斜杠作为路径分隔符
            rel_path = rel_path.replace(os.path.sep, '/')
            file_list[rel_path] = "add"

    return file_list

def main():
    parser = argparse.ArgumentParser(description='生成代码库文件列表')
    parser.add_argument('--clientId', required=True, help='客户端ID')
    parser.add_argument('--codebaseName', required=True, help='代码库名称')
    parser.add_argument('--codebasePath', required=True, help='代码库路径')
    parser.add_argument('--root', default='.', help='要遍历的根目录（默认为当前目录）')

    args = parser.parse_args()

    # 规范化代码库路径
    codebase_path = os.path.abspath(args.codebasePath)

    # 遍历目录获取文件列表
    file_list = traverse_directory(args.root)

    # 生成时间戳（秒级）
    timestamp = int(datetime.now().timestamp())

    # 构建结果数据结构
    result = {
        "clientId": args.clientId,
        "codebaseName": args.codebaseName,
        "codebasePath": codebase_path,
        "fileList": file_list,
        "timestamp": timestamp
    }

    # 创建 .shenma_sync 目录（如果不存在）
    sync_dir = os.path.join(os.getcwd(), '.shenma_sync')
    os.makedirs(sync_dir, exist_ok=True)

    # 构建输出文件名（时间戳.json）
    output_file = os.path.join(sync_dir, f"{timestamp}")

    # 写入 JSON 文件
    with open(output_file, 'w', encoding='utf-8') as f:
        json.dump(result, f, indent=2)

    print(f"已成功输出到: {output_file}")

if __name__ == "__main__":
    main()