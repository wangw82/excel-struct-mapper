# Excel Struct Mapper

[简体中文](README.zh-CN.md) | English

Excel Struct Mapper is a Go library for mapping spreadsheet rows to typed structures and exporting structures back to spreadsheets. It provides a format-independent row mapper and an XLSX adapter based on Excelize.

Repository: [github.com/wangw82/excel-struct-mapper](https://github.com/wangw82/excel-struct-mapper)

The API is pre-1.0 and may change before the first stable release.

## Goals

- Map worksheet columns to structure fields through explicit metadata.
- Validate headers, required values, and type conversions before data reaches application logic.
- Report errors with worksheet, row, column, and field context.
- Support standard text encoding interfaces without hiding conversion failures.
- Keep the core independent from application-specific models and infrastructure.

## Non-goals

- Replacing a full spreadsheet editor.
- Evaluating untrusted formulas or macros.
- Inferring business rules from cell formatting.
- Silently coercing ambiguous or invalid values.

## Features

- One compiled, immutable plan for both decoding and encoding.
- Strings, booleans, numbers, times, pointers, JSON values, and text marshalers.
- Extensible value codecs, block workflows, and application-owned struct tags without package initialization hooks.
- Repeated title blocks, vertical forms, transposed records, and recursive groups composed of multiple child tables.
- Location-aware errors compatible with `errors.Is` and `errors.As`.
- A format-neutral core plus an isolated Excelize-based XLSX adapter.

## Quick example

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

Use `xlsx.Read`, `xlsx.ReadFile`, `xlsx.Write`, and `xlsx.WriteFile` to adapt XLSX files to and from `mapper.Sheet`. See [Getting started](docs/getting-started.md) for complete usage and current module-path guidance.

Runnable examples for layouts, recursive groups, extensions, validation, and XLSX files are indexed in [`examples`](examples/README.md).

## Documentation

- [Documentation index](docs/README.md)
- [Getting started](docs/getting-started.md)
- [Core concepts](docs/concepts.md)
- [Tag reference](docs/tags.md)
- [Mapping and validation rules](docs/mapping-rules.md)
- [Architecture](docs/architecture.md)
- [FAQ](docs/faq.md)

## Requirements

- Go 1.25 or later.
- XLSX support requires `github.com/xuri/excelize/v2` through the `xlsx` package.

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening an issue or pull request. Participation is governed by the [Code of Conduct](CODE_OF_CONDUCT.md). Please report vulnerabilities according to [SECURITY.md](SECURITY.md), not through public issues.

Project roles and decisions are described in [GOVERNANCE.md](GOVERNANCE.md), and support boundaries are described in [SUPPORT.md](SUPPORT.md).

## License

Licensed under the [Apache License 2.0](LICENSE).
