# Excel Struct Mapper

简体中文 | [English](README.md)

Excel Struct Mapper 是一个 Go 语言库，用于将电子表格行映射为强类型结构，并将结构数据导出回电子表格。项目提供与文件格式无关的行映射器，以及基于 Excelize 的 XLSX 适配层。

项目地址：[github.com/wangw82/excel-struct-mapper](https://github.com/wangw82/excel-struct-mapper)

当前 API 尚未达到 1.0，首个稳定版本发布前仍可能调整。

## 项目目标

- 通过显式元数据将工作表列映射到结构字段。
- 在数据进入业务逻辑前校验表头、必填值和类型转换。
- 提供包含工作表、行、列及字段上下文的错误信息。
- 支持标准文本编解码接口，同时完整暴露转换失败。
- 保持核心能力独立于具体业务模型与基础设施。

## 非目标

- 替代完整的电子表格编辑器。
- 执行不可信的公式或宏。
- 从单元格样式中推断业务规则。
- 静默转换有歧义或无效的数据。

## 功能特性

- 编译一次不可变计划，同时用于解码和编码。
- 支持字符串、布尔值、数字、时间、指针、JSON 值及文本编解码接口。
- 通过值编解码器、块工作流和业务自有结构标签扩展，无需包初始化副作用。
- 支持重复标题区块、纵向表单、按列记录和由多个子表组成的递归结构组。
- 提供兼容 `errors.Is` 和 `errors.As` 的定位错误。
- 核心映射与 Excelize XLSX 适配层隔离。

## 快速示例

```go
type Person struct {
	ID   int    `excel:"header=ID;required=true"`
	Name string `excel:"header=Name;required=true"`
}

type Workbook struct {
	People []Person `excel:"key=people;workflow=all;format=slice"`
}

plan, err := mapper.Compile[Workbook]()
if err != nil {
	return err
}

sheet := mapper.NewSheet("People", [][]string{
    {"ID", "Name"},
    {"1", "Ada"},
})
var workbook Workbook
if err := plan.Decode(context.Background(), sheet, &workbook); err != nil {
	return err
}
```

处理 XLSX 工作簿时，可使用 `xlsx.Read`、`xlsx.ReadFile`、`xlsx.Write` 和 `xlsx.WriteFile`。完整用法见[快速开始](docs/getting-started.md)。

普通布局、递归结构组、扩展接口、校验及 XLSX 文件的可运行示例见 [`examples`](examples/README.md)。

## 文档

- [文档索引](docs/README.md)
- [快速开始](docs/getting-started.md)
- [核心概念](docs/concepts.md)
- [标签参考](docs/tags.md)
- [映射与校验规则](docs/mapping-rules.md)
- [架构设计](docs/architecture.md)
- [常见问题](docs/faq.md)

## 环境要求

- Go 1.21 或更高版本。
- `xlsx` 包通过 `github.com/xuri/excelize/v2` 提供 XLSX 支持。

## 参与贡献

提交 Issue 或 Pull Request 前，请阅读 [CONTRIBUTING.md](CONTRIBUTING.md)。社区协作遵循 [行为准则](CODE_OF_CONDUCT.md)。安全漏洞请按 [SECURITY.md](SECURITY.md) 私下报告，不要创建公开 Issue。

## 许可证

本项目采用 [Apache License 2.0](LICENSE) 许可。
