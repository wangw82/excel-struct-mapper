# Architecture

## Design Principles

- **Explicit behavior:** Schema and data problems are not hidden by inference.
- **Layered isolation:** Workbook adaptation, metadata, conversion, and diagnostics remain separate.
- **Clear boundaries:** Two-dimensional text mapping is independent from XLSX file handling.
- **Actionable errors:** Errors support both human diagnosis and programmatic classification.
- **Safe defaults:** The library does not execute macros, evaluate untrusted formulas, or include cell values in diagnostics.

## Modules

1. The **core mapper** converts between `[][]string` and struct slices without a spreadsheet dependency.
2. **Metadata parsing** reads struct tags, flattens anonymous structs, and detects mapping conflicts.
3. The **codec layer** handles scalars, times, JSON compound values, and standard text codecs.
4. The **diagnostic layer** provides unwrap-compatible row, column, field, and cause context.
5. The **XLSX adapter** uses Excelize to open, select, create, and save worksheets.

The repository intentionally keeps the small format-neutral implementation in one root package. Splitting reflection plans, codecs, and execution into public subpackages would expose internal types and increase the import surface without creating an independent reuse boundary. Files are instead grouped by responsibility; `xlsx` is a separate package because it owns a real external dependency boundary.

## Dependency direction

```text
application model
      |
Registry -> immutable compiled dependencies
      |                  |
      +---- Compile -----+
               |
          immutable Plan
          /            |             \
   BlockWorkflow    ValueCodec /    TagHandler
          \         BlockCodec       /
           \           |            /
             Decode / Encode / RunTag
               |
             Sheet
               |
          xlsx adapter
```

Execution depends only on interfaces resolved into `Plan`. It never looks up implementations by string and never depends on a mutable registry. Plans form a tree when `format=group` recursively compiles a nested struct; leaf nodes remain ordinary blocks using the same workflow and value or block codec interfaces. Registry names are resolved during compilation.

Application tag plans form a separate metadata tree. `TagHandler` outputs are application-owned values, so the core only preserves declaration order, field metadata, raw parameters, and parent-child relationships. This supports independent capabilities without coupling them to worksheet decoding or allowing arbitrary options in the mapper-owned tag.

## Codec boundaries

`ValueCodec` and `BlockCodec` deliberately remain separate: a value codec converts one field and a block codec converts an entire model field. Combining their inputs would require unrelated state and runtime mode checks. The two repeated method pairs are smaller and clearer than an extra generic abstraction.

`BlockWorkflow` owns both read selection and write placement. Selection returns one or more blocks so repeated layouts do not require execution to look up workflow names or concrete implementations. This keeps layout-specific rules out of the encoder and makes custom layouts symmetric.

## Read Path

```text
workbook -> worksheet -> rows -> metadata -> header binding -> conversion -> validation -> structs
```

Configuration and header errors return before data conversion. Execution returns the first operational error, and the destination changes only when every block succeeds. The XLSX adapter currently uses Excelize `GetRows` and does not provide a strict bounded-memory guarantee.

## Data Flow

1. `Compile[T]` reads and validates the unified `excel` tags once.
2. Compilation resolves workflows, block formats, value codecs, field paths, and row indexes into an immutable `Plan`.
3. `Decode` selects blocks from a `Sheet`, maps cells to fields, validates values, and checks for unconsumed non-empty rows.
4. `Encode` traverses the same plan in declaration order and builds a new `Sheet`.

## Concurrency

A built registry and a compiled plan are immutable. They can be shared by concurrent operations. Mutable registry builders must not be shared while registrations are in progress.

## Errors

Operational failures use structured errors with a kind, sheet, block, field, one-based row and column, A1 cell name, and wrapped cause. Callers can use `errors.Is` and `errors.As` without parsing messages.

## Dependency Boundary

The core package does not depend on application models, logging systems, network services, or file-format libraries. Excelize appears only in the `xlsx` adapter. Every new dependency requires a documented purpose, license, maintenance status, and alternatives.

## Security Boundary

Input workbooks are untrusted. Applications must enforce file size, worksheet count, row, column, and runtime limits appropriate to their environment. Bounded diagnostics can limit mapper error accumulation. Error messages contain location and causes but do not echo complete cell values.
