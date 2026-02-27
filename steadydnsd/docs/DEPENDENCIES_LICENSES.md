# 依赖项许可证清单

**检查日期**: 2026-02-27  
**项目许可证**: GNU Affero General Public License v3.0 (AGPLv3)

---

## 一、许可证头部检查报告

### 检查统计

| 指标 | 数量 |
|------|------|
| Go 源文件总数 | 80 |
| 已包含 AGPLv3 许可证头部 | 32 |
| 缺少许可证头部 | 48 |
| 合规率 | 40% |

### 标准许可证头部格式

```go
/*
SteadyDNS - DNS服务器实现

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
```

---

## 二、依赖项许可证检查报告

### 直接依赖项

| 依赖项 | 版本 | 许可证 | 与 AGPLv3 兼容性 |
|-------|------|-------|-----------------|
| github.com/gin-gonic/gin | v1.9.1 | MIT | ✅ 兼容 |
| github.com/golang-jwt/jwt/v4 | v4.5.2 | MIT | ✅ 兼容 |
| github.com/miekg/dns | v1.1.69 | BSD 3-Clause | ✅ 兼容 |
| golang.org/x/crypto | v0.46.0 | BSD 3-Clause | ✅ 兼容 |
| gorm.io/driver/sqlite | v1.6.0 | MIT | ✅ 兼容 |
| gorm.io/gorm | v1.31.1 | MIT | ✅ 兼容 |

### 间接依赖项

| 依赖项 | 版本 | 许可证 | 与 AGPLv3 兼容性 |
|-------|------|-------|-----------------|
| github.com/bytedance/sonic | v1.9.1 | MIT | ✅ 兼容 |
| github.com/chenzhuoyu/base64x | v0.0.0-20221115062448-fe3a3abad311 | MIT | ✅ 兼容 |
| github.com/gabriel-vasile/mimetype | v1.4.2 | MIT | ✅ 兼容 |
| github.com/gin-contrib/sse | v0.1.0 | MIT | ✅ 兼容 |
| github.com/go-playground/locales | v0.14.1 | MIT | ✅ 兼容 |
| github.com/go-playground/universal-translator | v0.18.1 | MIT | ✅ 兼容 |
| github.com/go-playground/validator/v10 | v10.14.0 | MIT | ✅ 兼容 |
| github.com/goccy/go-json | v0.10.2 | MIT | ✅ 兼容 |
| github.com/jinzhu/inflection | v1.0.0 | MIT | ✅ 兼容 |
| github.com/jinzhu/now | v1.1.5 | MIT | ✅ 兼容 |
| github.com/json-iterator/go | v1.1.12 | MIT | ✅ 兼容 |
| github.com/klauspost/cpuid/v2 | v2.2.4 | MIT | ✅ 兼容 |
| github.com/leodido/go-urn | v1.2.4 | MIT | ✅ 兼容 |
| github.com/mattn/go-isatty | v0.0.19 | MIT | ✅ 兼容 |
| github.com/mattn/go-sqlite3 | v1.14.22 | MIT | ✅ 兼容 |
| github.com/modern-go/concurrent | v0.0.0-20180306012644-bacd9c7ef1dd | MIT | ✅ 兼容 |
| github.com/modern-go/reflect2 | v1.0.2 | MIT | ✅ 兼容 |
| github.com/pelletier/go-toml/v2 | v2.0.8 | MIT | ✅ 兼容 |
| github.com/twitchyliquid64/golang-asm | v0.15.1 | MIT | ✅ 兼容 |
| github.com/ugorji/go/codec | v1.2.11 | MIT | ✅ 兼容 |
| golang.org/x/arch | v0.3.0 | BSD 3-Clause | ✅ 兼容 |
| golang.org/x/mod | v0.30.0 | BSD 3-Clause | ✅ 兼容 |
| golang.org/x/net | v0.47.0 | BSD 3-Clause | ✅ 兼容 |
| golang.org/x/sync | v0.19.0 | BSD 3-Clause | ✅ 兼容 |
| golang.org/x/sys | v0.39.0 | BSD 3-Clause | ✅ 兼容 |
| golang.org/x/text | v0.32.0 | BSD 3-Clause | ✅ 兼容 |
| golang.org/x/tools | v0.39.0 | BSD 3-Clause | ✅ 兼容 |
| google.golang.org/protobuf | v1.30.0 | BSD 3-Clause | ✅ 兼容 |
| gopkg.in/yaml.v3 | v3.0.1 | MIT | ✅ 兼容 |

---

## 三、兼容性说明

### 与 AGPLv3 兼容的许可证

以下许可证与 AGPLv3 完全兼容，可以安全地集成到项目中：

- **MIT License** - 宽松许可证，允许商业使用和修改
- **BSD 3-Clause License** - 宽松许可证，与 MIT 类似
- **Apache License 2.0** - 宽松许可证，提供专利授权保护
- **ISC License** - 功能上等同于 MIT 的简洁许可证

### 与 AGPLv3 不兼容的许可证

以下许可证与 AGPLv3 不兼容，**不应**集成到项目中：

- **GPLv2** - 与 GPLv3/AGPLv3 不兼容
- **LGPLv2.1** - 与 AGPLv3 存在兼容性问题
- **专有许可证** - 任何限制源代码公开的许可证

---

## 四、检查结论

### 依赖项许可证 ✅ 通过

所有依赖项均使用 MIT 或 BSD 3-Clause 许可证，与 AGPLv3 完全兼容。

### 源代码许可证头部 ⚠️ 需要修复

**48 个源文件缺少 AGPLv3 许可证头部**，需要在发布前补充。

### 建议操作

1. **优先级高**: 为所有缺少许可证头部的文件添加标准 AGPLv3 许可证头部
2. **优先级中**: 在 CI/CD 流程中添加许可证头部检查
3. **优先级低**: 定期更新此文档，保持依赖项信息最新

---

## 五、修复命令

可以使用以下脚本批量为缺少许可证头部的文件添加许可证：

```bash
#!/bin/bash
# add-license-header.sh

LICENSE_HEADER='/*
SteadyDNS - DNS服务器实现

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
'

for file in $(find . -name "*.go" -type f); do
    if ! grep -q "GNU Affero General Public License" "$file" 2>/dev/null; then
        echo "Adding license header to: $file"
        # 备份原文件
        cp "$file" "$file.bak"
        # 添加许可证头部
        echo "$LICENSE_HEADER" | cat - "$file.bak" > "$file"
        # 删除备份
        rm "$file.bak"
    fi
done
```

---

**文档维护**: 每次更新依赖项或添加新源文件后，应重新执行许可证检查并更新此文档。
