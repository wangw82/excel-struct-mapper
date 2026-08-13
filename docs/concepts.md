# Core Concepts

## Workbooks and Worksheets

A workbook contains input or output worksheets. A worksheet is the scope of one mapping operation. The `xlsx` package can select a worksheet explicitly; otherwise it reads the first worksheet. A missing requested worksheet returns `xlsx.ErrSheetNotFound`.

## Row Structs

A row struct is an application-defined typed record. Exported fields participate when they have an `excel` tag with an explicit `header`; `excel:"-"` excludes a field.

## Metadata

Struct metadata describes block workflows, block representations, column names, required fields, value codecs, and validation rules. It is compiled before execution so invalid tags and missing extensions fail early.

The `excel` tag is reserved for mapping behavior and remains strictly validated. Applications may register separate struct-tag namespaces with `RegisterTagHandler`. These handlers receive field metadata and recursively compiled child results without weakening `excel` tag validation or adding application concepts to the core.

## Repeated Groups

A group is a nested struct composed of child blocks. `format=group` compiles those child blocks into a plan tree instead of treating the struct fields as worksheet columns. When a group slice uses `workflow=repeat_title`, every matching outer region is decoded into one slice item. This preserves relationships between different child tables that repeat together.

Group recursion supports struct values, pointers, slices, pointer slice elements, and nested groups. `title` and `title_range` can bound one group; `repeat_title` creates a group slice and can use `end_title` to stop before the next sibling block with that title. Recursive Go types are rejected during compilation.

## Layout Formats

`struct` and `slice` map conventional header rows followed by records. `form` maps vertical label/value rows to one struct. `transpose` maps vertical labels with one record per value column. `single` maps one selected value, while `group` recursively composes independently selected child blocks.

Variable-width fields use `multi_cell=true`. A `separator` can terminate one field before the next variable-width field. Layouts that cannot be described by these formats use a registered `BlockCodec`, while selection and placement remain owned by a `BlockWorkflow`.

## Binding

Binding connects parsed headers to fields. By default, matching trims surrounding whitespace and ignores case. Options can change both policies. Binding uses names rather than positions, so input columns may be reordered.

## Conversion

Conversion maps cell text to a field type. Built-in support includes scalars, `time.Time`, pointers, and JSON-encoded slices, maps, and structs. Types implementing `encoding.TextUnmarshaler` or `encoding.TextMarshaler` control their text representation.

## Diagnostics

Diagnostics describe configuration, split, conversion, and validation failures. `*mapper.Error` contains a kind, sheet, block, field, one-based row and column, A1 cell name, and an unwrap-compatible cause.

Values can implement `MappingValidator` for row or nested-value rules. `WithModelValidation` runs once against the complete model after decoding and before encoding. These hooks receive the operation context and complement field-level `validate` rules.
