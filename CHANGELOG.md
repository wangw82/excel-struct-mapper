# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-08-13

### Added

- Go struct mapper with configurable header matching, tags, defaults, and validation.
- Scalar, time, pointer, JSON collection, and text marshaler conversions.
- XLSX reader and writer adapter based on Excelize.
- Excelize 2.11 with patched spreadsheet parsing and transitive dependencies.
- Immutable plans shared by decoding and encoding.
- Standard and custom block workflows, including repeated titles and title ranges.
- Conventional tables, scalar values, vertical forms, transposed records, and recursive groups.
- Explicit `ValueCodec`, `BlockCodec`, `BlockBinding`, and application-owned tag extensions.
- Field, mapped-value, and whole-model validation with structured location errors.
- Unit, integration, round-trip, race, fuzz, and boundary tests.
- Runnable examples, bilingual project overview, and public maintainer documentation.
- Cross-platform CI, coverage enforcement, CodeQL, dependency review, and sensitive-content scanning.

### Requirements

- Go 1.25 or later.

[Unreleased]: https://github.com/wangw82/excel-struct-mapper/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/wangw82/excel-struct-mapper/releases/tag/v0.1.0
