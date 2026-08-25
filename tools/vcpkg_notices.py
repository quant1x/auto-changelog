#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
vcpkg 第三方许可证收集与合规声明生成工具
用于 C++ 静态链接项目，自动生成符合法务要求的 THIRD_PARTY_NOTICES.txt

以项目下的 vcpkg manifest（vcpkg.json）为唯一依赖清单，只检索其中声明的
port 的 license（copyright 文件）与 notice（NOTICE 文件）。

与 Go 版 tools/noticegen 对齐的能力：
  - 按许可证文本关键词识别 SPDX 标识，不依赖 copyright 文件中是否含 "License:" 行
  - 提取 Copyright 版权声明行（满足 MIT/BSD 的版权保留义务）
  - 处理 NOTICE 文件全文（满足 Apache-2.0 第 4(d) 条的 NOTICE 再分发要求）
  - 模块名之后保留空白行，格式与 third_party/notice.tmpl 一致

用法（示例）：
    python tools/vcpkg_notices.py --installed-dir vcpkg_installed/x64-windows --product-name "quant1x"
    python tools/vcpkg_notices.py --vcpkg-json vcpkg.json --ports-dir C:/vcpkg/ports
"""

from __future__ import annotations

import argparse
import json
import platform as _platform
import re
import sys
from pathlib import Path

# ---------------------------------------------------------------------------
# 1. 许可证识别：按文本关键词识别 SPDX 标识（对齐 noticegen 的 detectLicenseType）
# ---------------------------------------------------------------------------
def detect_license_type(text: str) -> str:
    """依据许可证全文关键词识别 SPDX 标识，支持混合许可（AND 连接）。"""
    kinds: list[str] = []
    if "Apache License" in text and "Version 2.0" in text:
        kinds.append("Apache-2.0")
    if "MIT License" in text or (
        "Permission is hereby granted, free of charge" in text
        and 'THE SOFTWARE IS PROVIDED "AS IS"' in text
    ):
        kinds.append("MIT")
    if "Redistribution and use in source and binary forms" in text:
        kinds.append("BSD-3-Clause" if "Neither the name" in text else "BSD-2-Clause")
    if "Mozilla Public License" in text:
        kinds.append("MPL-2.0")
    if "ISC License" in text or (
        "Permission to use, copy, modify" in text and "IN NO EVENT SHALL" in text
    ):
        kinds.append("ISC")
    if "GNU GENERAL PUBLIC LICENSE" in text:
        kinds.append("GPL")
    if "Boost Software License" in text:
        kinds.append("BSL-1.0")
    if "Microsoft Public License" in text:
        kinds.append("MS-PL")
    if "The Unlicense" in text or "This is free and unencumbered software" in text:
        kinds.append("Unlicense")
    # zlib：双关键词避免与其他许可文本误判
    if "This software is provided 'as-is'" in text and "Altered source versions" in text:
        kinds.append("Zlib")
    return " AND ".join(kinds) if kinds else "Unknown"


# 行首的 "License: xxx" / "SPDX-License-Identifier: xxx"
# 要求：行首开始 + 带冒号。避免把正文中 "license text here"、"License Version 2.0"
# 等中间出现 license 单词、以及跨行内容误判为许可证名
license_line_re = re.compile(
    r"^[ \t]*(?:SPDX-License-Identifier|License)[ \t]*:[ \t]*([A-Za-z0-9.\-+]+)",
    re.IGNORECASE | re.MULTILINE,
)


def detect_license_name(text: str) -> str:
    """识别许可证类型：优先按文本关键词，其次解析行首的 'License:'/'SPDX-License-Identifier:'。"""
    name = detect_license_type(text)
    if name != "Unknown":
        return name
    m = license_line_re.search(text)
    return m.group(1) if m else "Unknown"


# ---------------------------------------------------------------------------
# 2. 版权声明提取
# ---------------------------------------------------------------------------
# 匹配行首版权声明，兼容 C/C++ 常见写法：
#   "Copyright (c) 2015 Microsoft"、"Copyright (C) 1995 Jean-loup Gailly"、
#   "Copyright 2015 Google"、"(C) 1995 ..."、"© 2015 ..."
copyright_re = re.compile(
    r"(?im)^[ \t]*(?:copyright[ \t]*(?:\(c\)|©|\(C\))?|\(c\)|\(C\)|©)[ \t]*\d{4}.*$"
)


def extract_copyright(text: str) -> str:
    """从许可证文本中提取版权声明行（去重、取前 4 行）。"""
    seen: list[str] = []
    for m in copyright_re.findall(text):
        line = m.strip()
        if line not in seen:
            seen.append(line)
            if len(seen) == 4:
                break
    return "\n".join(seen)


# ---------------------------------------------------------------------------
# 3. 版本号提取
# ---------------------------------------------------------------------------
def first_line_version(text: str) -> str:
    """启发式：copyright 首行形如 'zlib 1.2.11' 时提取版本号。"""
    m = re.match(r"^[A-Za-z0-9_\-]+\s+(\d[\w.\-]*)", text.split("\n", 1)[0])
    return m.group(1) if m else ""


def read_version_from_metadata(port_dir: Path, heuristic: str) -> str:
    """版本号来源优先级：vcpkg.json > vcpkg.status > copyright 首行启发式。"""
    # 1. vcpkg.json（manifest 模式）
    vcpkg_json = port_dir / "vcpkg.json"
    if vcpkg_json.exists():
        try:
            data = json.loads(vcpkg_json.read_text(encoding="utf-8", errors="ignore"))
            version = data.get("version") or data.get("version-string")
            if version:
                return str(version)
        except Exception as e:
            print(f"  [警告] 解析 {vcpkg_json} 失败: {e}")
    # 2. vcpkg.status（经典模式）
    status = port_dir / "vcpkg.status"
    if status.exists():
        try:
            for line in status.read_text(encoding="utf-8", errors="ignore").splitlines():
                m = re.match(r"^Version:\s*(.+)$", line.strip())
                if m:
                    return m.group(1).strip()
        except Exception as e:
            print(f"  [警告] 解析 {status} 失败: {e}")
    # 3. copyright 首行启发式
    return heuristic or "Unknown"


# ---------------------------------------------------------------------------
# 4. 解析 vcpkg 的 copyright 文件
# ---------------------------------------------------------------------------
def parse_vcpkg_copyright(copyright_path: Path) -> dict | None:
    """
    解析 vcpkg 生成的 copyright 文件。
    注意：vcpkg 的 copyright 文件没有严格的机器可读标准，此函数做了最大程度的兼容和容错。
    """
    try:
        text = copyright_path.read_text(encoding="utf-8", errors="ignore").strip()
    except Exception as e:
        print(f"  [警告] 无法读取 {copyright_path}: {e}")
        return None

    if not text:
        return None

    return {
        "license_name": detect_license_name(text),
        "copyright": extract_copyright(text),
        "version": read_version_from_metadata(copyright_path.parent, first_line_version(text)),
        "license_text": text,
    }


# ---------------------------------------------------------------------------
# 5. 解析 vcpkg manifest（依赖清单）
# ---------------------------------------------------------------------------
def current_platform_tags() -> set[str]:
    """返回当前平台的 vcpkg 标识集合（如 windows/x64/linux/arm64）。"""
    tags: set[str] = set()
    sys_name = _platform.system().lower()
    if "windows" in sys_name:
        tags.add("windows")
    elif "darwin" in sys_name:
        tags.add("osx")
    elif "linux" in sys_name:
        tags.add("linux")
    elif "freebsd" in sys_name:
        tags.add("freebsd")
    arch = _platform.machine().lower()
    if arch in ("amd64", "x86_64"):
        tags.add("x64")
    elif arch in ("x86", "i386", "i686"):
        tags.add("x86")
    elif arch in ("arm64", "aarch64"):
        tags.add("arm64")
    elif arch.startswith("arm"):
        tags.add("arm")
    return tags


def evaluate_platform(expr: str) -> bool:
    """简单评估 vcpkg platform 表达式（如 '!windows'、'linux & x64'、'windows | osx'）。"""
    current = current_platform_tags()
    for alt in expr.split("|"):
        alt = alt.strip()
        ok = True
        for cond in alt.split("&"):
            cond = cond.strip()
            neg = cond.startswith("!")
            tag = cond[1:].strip() if neg else cond
            matched = tag in current
            if neg:
                matched = not matched
            if not matched:
                ok = False
                break
        if ok:
            return True
    return False


def parse_manifest_dependencies(vcpkg_json: Path) -> list[str]:
    """
    解析 vcpkg manifest 的 dependencies，返回 port 名列表。
    支持字符串与对象（含 platform 限定）两种声明形式，按声明顺序去重。
    """
    try:
        data = json.loads(vcpkg_json.read_text(encoding="utf-8", errors="ignore"))
    except Exception as e:
        print(f"错误: 解析 {vcpkg_json} 失败: {e}")
        return []

    deps: list[str] = []
    for dep in data.get("dependencies", []):
        if isinstance(dep, str):
            name, platform = dep, ""
        elif isinstance(dep, dict):
            name, platform = dep.get("name"), dep.get("platform", "")
        else:
            continue
        if not name or name in deps:
            continue
        if platform and not evaluate_platform(platform):
            print(f"  [-] 跳过 {name}（平台限定 {platform} 不匹配当前环境）")
            continue
        deps.append(name)
    return deps


# ---------------------------------------------------------------------------
# 6. 检索依赖的 license / notice
# ---------------------------------------------------------------------------
def find_share_dir(installed_dir: Path) -> Path | None:
    """兼容 vcpkg_installed/share 与 vcpkg_installed/<triplet>/share 两种布局。"""
    direct = installed_dir / "share"
    if direct.is_dir():
        return direct
    for sub in sorted(installed_dir.iterdir()):
        if sub.is_dir():
            cand = sub / "share"
            if cand.is_dir():
                return cand
    return None


def load_port_notice(port_dir: Path) -> str:
    """读取 port 目录下的 NOTICE 文件（Apache-2.0 等协议需要）；Windows 上 glob 大小写不敏感，set 去重。"""
    notice_files = sorted(
        {p for p in port_dir.glob("NOTICE*")} | {p for p in port_dir.glob("notice*")}
    )
    if not notice_files:
        return ""
    try:
        return notice_files[0].read_text(encoding="utf-8", errors="ignore").strip()
    except Exception:
        return ""


def collect_vcpkg_licenses(
    dep_names: list[str],
    installed_dir: Path | None = None,
    ports_dir: Path | None = None,
) -> list[dict]:
    """只对 vcpkg manifest 中声明的依赖检索 license（copyright）与 notice。"""
    licenses: list[dict] = []

    for name in dep_names:
        info = None

        # 1. 优先从 vcpkg 安装目录检索（share/<port>/copyright）
        if installed_dir:
            share = find_share_dir(installed_dir)
            if share:
                port_dir = share / name
                copyright_file = port_dir / "copyright"
                if copyright_file.exists():
                    info = parse_vcpkg_copyright(copyright_file)
                    if info:
                        info["name"] = name
                        info["notice_text"] = load_port_notice(port_dir)

        # 2. 备选：vcpkg 源码 ports 目录（<port>/copyright）
        if info is None and ports_dir:
            port_dir = ports_dir / name
            copyright_file = port_dir / "copyright"
            if copyright_file.exists():
                info = parse_vcpkg_copyright(copyright_file)
                if info:
                    info["name"] = name
                    info["notice_text"] = load_port_notice(port_dir)

        if info is None:
            print(f"  [-] 未找到 {name} 的 copyright 文件，跳过")
            continue

        licenses.append(info)
        print(f"  [+] 已收集: {info['name']} ({info['version']}) - {info['license_name']}")

    return licenses


# ---------------------------------------------------------------------------
# 7. 生成最终的 THIRD_PARTY_NOTICES 文件
# ---------------------------------------------------------------------------
def generate_notices_file(licenses: list[dict], output_path: Path, product_name: str) -> None:
    """根据收集到的信息，生成合规的纯文本声明文件（格式与 third_party/notice.tmpl 一致）。"""
    header = (
        "This product includes software developed by third parties.\n"
        "\n"
        f"The {product_name} software itself is distributed under its respective license.\n"
        "\n"
        "The following third-party software is included in this product:\n"
    )

    with open(output_path, "w", encoding="utf-8") as f:
        f.write(header)

        for lib in licenses:
            f.write("\n" + "-" * 80 + "\n")
            f.write(f"{lib['name']} {lib['version']}\n\n")  # 名称之后留空白行

            if lib["copyright"]:
                f.write(f"Copyright: {lib['copyright']}\n")
            f.write(f"License: {lib['license_name']}\n\n")

            # 写入完整的 License 文本
            f.write(f"{lib['license_text']}\n")

            # 如果有 NOTICE 文件，追加输出（满足 Apache 2.0 等协议要求）
            if lib["notice_text"]:
                f.write("\nNOTICE FILE CONTENTS:\n")
                f.write(f"{lib['notice_text']}\n")

    print(f"已生成第三方声明文件: {output_path}")
    print(f"   共包含 {len(licenses)} 个第三方库。")

    # 摘要警告（对齐 noticegen 的输出风格）
    unknown = [lib["name"] for lib in licenses if lib["license_name"] == "Unknown"]
    no_copyright = [lib["name"] for lib in licenses if not lib["copyright"]]
    with_notice = [lib["name"] for lib in licenses if lib["notice_text"]]
    if unknown:
        print(f"WARN 未识别到许可证类型: {', '.join(unknown)}")
    if no_copyright:
        print(f"WARN 未提取到版权声明: {', '.join(no_copyright)}")
    if with_notice:
        print(f"含 NOTICE 文件: {', '.join(with_notice)}")


# ---------------------------------------------------------------------------
# 8. 命令行入口
# ---------------------------------------------------------------------------
def main() -> None:
    parser = argparse.ArgumentParser(description="vcpkg 第三方许可证收集与合规声明生成工具")
    parser.add_argument(
        "--vcpkg-json",
        type=str,
        default="vcpkg.json",
        help="项目下的 vcpkg manifest 文件路径 (默认: 当前目录 vcpkg.json)",
    )
    parser.add_argument(
        "--installed-dir",
        type=str,
        default=None,
        help="vcpkg 安装目录 (例如: vcpkg_installed/x64-windows)，从 share/<port>/copyright 检索",
    )
    parser.add_argument(
        "--ports-dir",
        type=str,
        default=None,
        help="vcpkg 源码 ports 目录 (例如: C:/vcpkg/ports)，未找到 installed 时的备选检索源",
    )
    parser.add_argument(
        "--output",
        type=str,
        default="THIRD_PARTY_NOTICES.txt",
        help="输出的声明文件路径 (默认: THIRD_PARTY_NOTICES.txt)",
    )
    parser.add_argument(
        "--product-name",
        type=str,
        default="My C++ Application",
        help="你自己的产品名称",
    )

    args = parser.parse_args()

    vcpkg_json = Path(args.vcpkg_json)
    if not vcpkg_json.exists():
        print(f"错误: 找不到 vcpkg manifest: {vcpkg_json}")
        return

    installed_path = Path(args.installed_dir) if args.installed_dir else None
    ports_path = Path(args.ports_dir) if args.ports_dir else None
    if not installed_path and not ports_path:
        # 未显式指定检索源时，尝试当前目录下的 vcpkg_installed
        candidate = Path("vcpkg_installed")
        if candidate.exists():
            installed_path = candidate
    if not installed_path and not ports_path:
        print("错误: 请指定 --installed-dir 或 --ports-dir 作为版权检索源")
        return

    dep_names = parse_manifest_dependencies(vcpkg_json)
    print(f"vcpkg manifest: {vcpkg_json}，声明依赖 {len(dep_names)} 个: {', '.join(dep_names)}")
    if not dep_names:
        print("警告: manifest 中没有声明任何依赖")
        return

    licenses = collect_vcpkg_licenses(dep_names, installed_path, ports_path)
    if not licenses:
        print("警告: 没有检索到任何第三方库的许可证信息。请检查检索源目录是否正确。")
        return

    generate_notices_file(licenses, Path(args.output), args.product_name)


if __name__ == "__main__":
    main()
