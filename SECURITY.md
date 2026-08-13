# Security Policy

## Supported Versions

The project has not published a tagged release. Until then, security fixes target the current default branch only. This section will list supported release lines after the first release.

## Reporting a Vulnerability

Use **Security advisories → Report a vulnerability** in the GitHub repository. Do not open a public issue or include exploit details, real workbooks, personal information, or credentials in public examples.

Include the affected version or revision, minimal reproduction steps with sanitized input, likely impact and prerequisites, attempted mitigations, and a contact method for coordinated disclosure.

Maintainers will acknowledge the report privately, assess impact, and coordinate remediation and disclosure. Response and remediation time depend on complexity. Avoid public disclosure until a fix and advisory are available.

## Security Boundary

Treat every spreadsheet as untrusted input. The project does not execute macros, guarantee safe evaluation of external formulas, remove every form of active content, or replace application-level file size and resource limits. Security guarantees are defined by the documentation for each supported release.

Values beginning with `=`, `+`, `-`, or `@` may be interpreted as formulas by spreadsheet applications. The XLSX writer stores values as text by default. Enable formula output only for trusted data when formula behavior is explicitly required.

Applications must enforce upload-size and workbook-complexity limits before reading input and should use context cancellation around mapping operations. Avoid logging complete rows, workbooks, or raw cell values; structured mapper errors provide locations and field names without requiring sensitive input in logs.
