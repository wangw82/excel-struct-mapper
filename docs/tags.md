# Unified Excel Tags

Tags contain escaped, semicolon-separated `key=value` options.

```go
type Catalog struct {
    Products []Product `excel:"key=products;workflow=title;title=Products;min_rows=3;format=slice"`
}
```

Use `\;`, `\=`, and `\\` for a literal semicolon, equals sign, and backslash. Duplicate keys, unknown keys, empty required values, and invalid combinations fail during compilation.

## Block options

`key`, `workflow`, `format`, `start_row`, `end_row`, `title`, `end_title`, `min_rows`, `blank_line`, `optional`, `data_row`, `label_col`, `value_col`, `label`, `multi_cell`, `separator`, `allow_empty`, `value_codec`, `block_codec`, `codec_options`, and `include_end_block` configure a block. Supported workflows are `all`, `index`, `start`, `title`, `repeat_title`, and `title_range`. Supported formats are `struct`, `slice`, `single`, `group`, `form`, and `transpose`.

`repeat_title` selects every block beginning with the configured title. With `blank_line=true`, each occurrence ends at the next blank line; otherwise it ends at the next matching title. Set `end_title` when repeated blocks are followed by another top-level block; it must match the title of the next sibling block, and the boundary title is not consumed. A regular `slice` merges records from every selected block.

`optional=true` allows a block to be absent or to contain only its title. Missing optional blocks leave the destination field at its zero value and empty optional values are omitted during encoding. Selection, conversion, and validation errors in a present block are still returned.

`title_range` accepts either `title`, `end_title`, or both. An omitted `title` starts at the beginning of the worksheet; an omitted `end_title` extends to the end.

`group` recursively executes the tagged block fields of a nested struct. A group slice requires `workflow=repeat_title`, and its title must match the first child block title. Each selected outer block becomes one group item, preserving the relationship between its child blocks. A non-slice group uses `all`, `index`, `start`, `title`, or `title_range` and produces one nested struct. A range group's start and end titles must match its first and last child block titles, and the last child uses `blank_line=true` as the range delimiter. The final child may end at the end of the worksheet when no trailing blank row exists. Empty group slices are rejected during encoding so encoded output remains decodable.

```go
type Group struct {
    Entries  []Entry  `excel:"key=entries;workflow=title;title=ITEMS;format=slice;blank_line=true"`
    Policies []Policy `excel:"key=policies;workflow=title;title=POLICIES;format=slice;blank_line=true"`
}

type Workbook struct {
    Groups []Group `excel:"key=groups;workflow=repeat_title;title=ITEMS;format=group"`
}
```

`form` maps a vertical label/value block to one struct. `transpose` maps each value column to one slice item. `data_row`, `label_col`, and `value_col` are one-based when supplied. Their defaults are the first content row, column 1, and column 2.

```go
type Metadata struct {
    Name   string   `excel:"header=Name;required=true"`
    Limits []string `excel:"header=Limits;multi_cell=true;separator=/"`
}

type Target struct {
    Metadata Metadata `excel:"key=metadata;workflow=title;title=META;format=form;blank_line=true"`
}
```

For a scalar block, `label`, `label_col`, `value_col`, `data_row`, `multi_cell`, `separator`, `allow_empty`, and `value_codec` describe the selected row. A configured label is validated during decoding and emitted during encoding. Registered value codecs can therefore be reused by table fields and standalone values.

Set `block_codec=name` to resolve a whole-block codec registered with `Registry.RegisterBlockCodec`. It cannot be combined with `format=group`. `codec_options` is passed unchanged through `ValueCodecContext` or `BlockCodecContext`, allowing one registered implementation to support parameterized fields. A `title_range` block codec can set `include_end_block=true` to receive and emit the ending titled section; in that mode the codec owns the complete framing.

## Field options

`header`, `required`, `allow_empty`, `skip_decode`, `skip_encode`, `multi_cell`, `separator`, `value_codec`, `codec_options`, and `validate` configure fields. `separator` requires `multi_cell=true`, terminates that field's cells, and is emitted during encoding. User-visible row and column numbers are one-based and are normalized during compilation.

## Application-owned tags

The `excel` tag is reserved and rejects unknown mapping options. Independent capabilities can use their own struct tag after registering named implementations with `Registry.RegisterTagHandler`. Values use `name=raw parameters`; only the first equals sign is structural, and the remaining text is passed unchanged. A value containing only `name` has no parameters, while `-` skips the field and its descendants.

Execute a compiled application tag with `Plan.RunTag`. Results preserve field declaration order and nesting. Parent handlers receive child results, allowing recursive struct and slice metadata without adding application-specific behavior to the mapper.
