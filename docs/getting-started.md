# Getting Started

## Installation

Until the first tagged release, install the current main branch explicitly:

```text
go get github.com/wangw82/excel-struct-mapper@main
```

Import the core as `github.com/wangw82/excel-struct-mapper` and the XLSX adapter as `github.com/wangw82/excel-struct-mapper/xlsx`.

## Model and plan

Mapping has three steps: define a model, compile a plan, and reuse that plan.

```go
type Product struct {
    ID   int    `excel:"header=ID;required=true"`
    Name string `excel:"header=Name;required=true"`
}

type Catalog struct {
    Products []Product `excel:"key=products;workflow=all;format=slice"`
}

plan, err := mapper.Compile[Catalog]()
if err != nil {
    return err
}
```

`Plan` is immutable and safe to reuse concurrently. Compilation resolves every workflow and value codec, so execution does not consult mutable global state.

## Decode and encode

```go
sheet := mapper.NewSheet("Catalog", [][]string{
    {"ID", "Name"},
    {"7", "Notebook"},
})

var catalog Catalog
if err := plan.Decode(context.Background(), sheet, &catalog); err != nil {
    return err
}

output, err := plan.Encode(context.Background(), catalog)
```

Header matching trims surrounding whitespace and ignores case by default. Pass `WithTrimTitle(false)` or `WithCaseInsensitiveTitle(false)` to `Compile` to change those policies.

## XLSX adapter

The `xlsx` package only translates between XLSX workbooks and `mapper.Sheet`.

```go
sheet, err := xlsx.ReadFile("input.xlsx", xlsx.WithSheetName("Catalog"))
if err != nil {
    return err
}
var catalog Catalog
if err := plan.Decode(context.Background(), sheet, &catalog); err != nil {
    return err
}

output, err := plan.Encode(context.Background(), catalog)
if err == nil {
    err = xlsx.WriteFile("output.xlsx", output, xlsx.WithSheetName("Catalog"))
}
```

## Extension points

- `ValueCodec` converts one field between cells and a Go value. Register it with `Registry.RegisterValueCodec`.
- `BlockCodec` owns conversion of an entire selected block. Register it with `Registry.RegisterBlockCodec` and reference it with `block_codec`, or supply it through `BlockBinding` for exceptional fields that cannot use tags.
- `BlockWorkflow` owns both input block selection and output block placement. Its `Select` method returns ordered, non-overlapping `[]Block`; `Place` receives the corresponding `[][]Line` logical blocks. Register it with `Registry.RegisterBlockWorkflow`.
- `TagHandler` implements one named behavior in an application-owned struct tag. Register it with `Registry.RegisterTagHandler`; the compiled plan can then execute that tag with `Plan.RunTag`.

Pass registry and policy options directly to `Compile`. For exceptional model fields, call `CompileWithBindings` with explicit `BlockBinding` values.

Implement `MappingValidator` on mapped values for local cross-field rules. Use `WithModelValidation` for rules that require the complete decoded or encoded model.

## Application-owned tags

Applications can add an independent metadata capability without adding options to the mapper-owned `excel` tag. A tag value has the form `handler=parameters`; everything after the first equals sign is passed to the handler unchanged. A handler can return any application-owned result.

```go
registry := mapper.NewRegistry()
err := registry.RegisterTagHandler("ui", "text", mapper.TagHandlerFunc(
    func(_ context.Context, tag mapper.TagContext) (any, error) {
        return map[string]any{
            "field": tag.Field.Name,
            "label": tag.Params,
        }, nil
    },
))

type Model struct {
    Rows  []Row  `excel:"key=rows;workflow=all;format=slice"`
    Title string `json:"title" ui:"text=Heading,Heading"`
}

plan, err := mapper.Compile[Model](mapper.WithRegistry(registry))
items, err := plan.RunTag(context.Background(), "ui")
```

Handlers receive the field path, `reflect.StructField`, raw parameters, and recursively produced child results. A parent handler can therefore compose a tagged struct or slice from its children. Use `ui:"-"` to skip a field and its descendants. Unknown handlers, invalid tags, duplicate registrations, and attempts to register the reserved `excel` tag fail before execution. A compiled plan owns its resolved handlers and is safe to reuse concurrently; registered handlers must also support concurrent calls when the plan is shared.

Application tags are intended for deterministic metadata generation such as UI schemas, documentation, or policy descriptions. Operations that depend on live model values or perform external side effects belong in the application service layer.

## Errors

Operational failures return `*mapper.Error` with sheet, block, field, row, column, cell name, and an unwrap-compatible cause. Configuration failures are returned by `Compile` whenever possible.
