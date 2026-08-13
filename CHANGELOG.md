# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project intends to follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html) after the first public release.

## [Unreleased]

### Added

- Go struct mapper with configurable header matching, tags, defaults, and validation.
- Scalar, time, pointer, JSON collection, and text marshaler conversions.
- XLSX reader and writer adapter based on Excelize.
- Immutable plans shared by decoding and encoding.
- Standard and custom block workflows, including repeated titles and title ranges.
- Conventional tables, scalar values, vertical forms, transposed records, and recursive groups.
- Explicit `ValueCodec`, `BlockCodec`, `BlockBinding`, and application-owned tag extensions.
- Field, mapped-value, and whole-model validation with structured location errors.
- Unit, integration, round-trip, race, fuzz, and boundary tests.
- Runnable examples, bilingual project overview, and public maintainer documentation.
- Cross-platform CI, coverage enforcement, CodeQL, dependency review, and sensitive-content scanning.
