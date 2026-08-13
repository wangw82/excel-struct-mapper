# Contributing

Thank you for contributing to Excel Struct Mapper. The project welcomes defect reports, use cases, documentation improvements, tests, reviews, and code changes.

## Before You Start

- Read the [README](README.md) and [architecture guide](docs/architecture.md).
- Search existing issues and pull requests to avoid duplicate work.
- Open a feature request before a large change to discuss scope, alternatives, and compatibility.
- Report security issues privately according to [SECURITY.md](SECURITY.md).

## Development

Use Go 1.25 or later. Run the required local checks before requesting review:

```text
gofmt -w .
go vet ./...
go test -race -coverprofile=coverage.out ./...
Get-ChildItem examples -Directory | ForEach-Object { go run "./examples/$($_.Name)" }
```

Generated example workbooks must be written to a temporary directory and removed by the example.

## Issue Requirements

A defect report should include reproduction steps, expected behavior, actual behavior, and a minimal sanitized example. A feature request should describe the user scenario and acceptance criteria rather than only prescribing an implementation.

Do not submit real business data, personal information, credentials, private addresses, customer names, or other non-public identifiers.

## Pull Request Requirements

- Keep each pull request focused on one reviewable objective.
- Explain motivation, scope, validation, and compatibility impact.
- Add tests and documentation for changed public behavior.
- Explain the purpose, license, maintenance status, and alternatives for new dependencies.
- Request review only after all automated checks pass.

## Review and Licensing

Reviews consider correctness, security, maintainability, compatibility, and documentation. By submitting a contribution, you agree to license it under the [Apache License 2.0](LICENSE) and confirm that you have the right to do so. Project decisions follow [GOVERNANCE.md](GOVERNANCE.md).
