# Frequently Asked Questions

## Can I install it now?

Yes. Until the first tagged release, install the main branch with `go get github.com/wangw82/excel-struct-mapper@main`. The API is pre-1.0 and may still change.

## Are the examples current?

The README, getting-started guide, and `examples` directory track the current source. The API may still change before the first tag, with notable changes recorded in the changelog.

## Which spreadsheet formats are supported?

The adapter supports XLSX through Excelize. The core mapper accepts `[][]string`, so applications can adapt CSV or other row sources independently.

## Does it execute formulas or macros?

It does not execute macros. XLSX output writes text and does not generate formulas. Treat formula-related input and cached workbook content as untrusted.

## How should I handle large files?

The core mapper and current XLSX adapter keep complete row collections in memory. Enforce file, worksheet, row, column, and runtime limits before reading untrusted workbooks. Streaming and resource benchmarks remain roadmap work.

## How do I report a security issue?

Follow the [security policy](../SECURITY.md) and use private vulnerability reporting. Do not disclose exploit details or sensitive examples publicly.
