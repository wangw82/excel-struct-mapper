# Runnable Examples

Every directory is an independent program. Run one with `go run ./examples/<name>` or run the complete set with the repository CI command.

| Example | Covered scenarios |
| --- | --- |
| `basic` | Minimal in-memory decode using a conventional row table. |
| `layouts` | All six block formats, pointers, optional and repeated blocks, separated multi-cell fields, both open-ended title ranges, indexed rows, and the unconsumed-row policy. |
| `groups` | Repeated compound groups, bounded range groups, and nested recursive groups. |
| `extensions` | Registered value and block codecs, codec options, an inclusive title range, a custom workflow, an application-owned tag, and an explicit block binding. |
| `types` | Built-in scalar and time conversion, pointers, JSON compound values, and standard text marshaling interfaces. |
| `validation` | Field validation, `MappingValidator`, whole-model validation, and structured location errors. |
| `xlsx` | Temporary XLSX file write/read round trip through the Excelize adapter. |

Examples use neutral data and panic when an expected round trip or error contract does not hold. Detailed failure and edge-case coverage remains in the test suite.
