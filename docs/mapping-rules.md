# Mapping and Validation Rules

## Header processing

- Surrounding header whitespace is trimmed by default; internal characters are unchanged.
- Header matching ignores case by default.
- Use `WithTrimTitle(false)` or `WithCaseInsensitiveTitle(false)` before compilation to change either policy.
- Headers duplicated after normalization are rejected.
- A missing `required=true` column fails before data rows are processed.
- Unbound input columns and missing optional columns are ignored.

## Empty values

- Whitespace-only input is considered empty for required-value validation.
- A required empty value returns a location-aware validation error.
- Optional empty values leave the Go field at its zero value.
- Non-empty values are passed to the resolved `ValueCodec`; conversion failures are never hidden.

## Type conversion

- Numeric and boolean conversion use `strconv` and require valid complete values.
- `time.Time` accepts Excel serial values, RFC 3339, `2006-01-02 15:04:05`, and `2006-01-02`.
- String slices use multiple cells; other slices, maps, and non-time structs use JSON text.
- Standard text marshal/unmarshal implementations take precedence and preserve their underlying errors.
- Custom field conversion is registered as a `ValueCodec` and resolved during `Compile`.

## Failure behavior

The mapper returns the first configuration, split, conversion, or validation failure. Decode builds a temporary model and updates the destination only after every block succeeds.

## Repeated block boundaries

- `repeat_title` selects all matching title occurrences in source order.
- With `blank_line=true`, a repeated block ends at its following blank line and consumes that delimiter.
- Without a blank-line boundary, a repeated block ends at the next occurrence of the same title.
- `end_title` explicitly stops repeated selection before the next sibling block. It must match that block's title and is required to exist in the input when configured.
- An optional block may be absent or contain only its title; malformed content and missing configured end boundaries remain errors.
- Multiple selected blocks assigned to a regular slice are concatenated in source order.
- Multiple selected group blocks require a slice field; each block produces one group item.
- Workflow-selected blocks must be non-overlapping and ordered by source row.
- Custom workflows select logical blocks as `[]Block` and receive the corresponding encoded blocks as `[][]Line` during placement.
- Child blocks must consume every non-empty row inside a group unless unconsumed rows are explicitly ignored.

## Layout orientation

- `struct` and `slice` read fields from columns under a header row.
- `form` reads fields from rows identified by labels.
- `transpose` reads one slice item from each non-empty value column.
- `single` can validate a row label and read one or multiple cells at configured coordinates.
- A multi-cell field with `separator` stops before that marker; the marker is structural and is reproduced during encoding.

## Writing

Block placement is delegated to the compiled `BlockWorkflow`. Conflicting row layouts return `ErrLayoutConflict`; data is never silently overwritten. XLSX writing stores text and protects formula-like values by default.
