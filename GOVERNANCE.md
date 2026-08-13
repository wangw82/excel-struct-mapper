# Governance

## Project Roles

- Contributors report issues, improve documentation, propose changes, and review pull requests.
- Maintainers triage reports, review changes, manage releases, and enforce project policies.
- Security responders coordinate private vulnerability reports and disclosure.

Roles are earned through sustained, constructive participation. Maintainers may invite a contributor after considering technical judgment, review quality, reliability, and conduct. A maintainer may step down at any time or be removed for prolonged inactivity, policy violations, or loss of project trust.

## Decision Making

Routine changes use lazy consensus: a maintainer may merge a focused change after required checks pass and review concerns are resolved. Compatibility, security, governance, and release-policy changes require approval from two maintainers when two or more active maintainers are available. When only one maintainer is active, the decision and rationale must be recorded in the pull request.

The project prefers documented technical evidence and user impact over authority or voting. If consensus cannot be reached, maintainers defer the change while alternatives and risks are investigated.

## Changes and Releases

All project changes use pull requests, except narrowly scoped emergency security work coordinated through a private advisory. Protected branches require automated checks. Releases follow Semantic Versioning, use signed or GitHub-generated source archives, and derive release notes from `CHANGELOG.md`.

Maintainers must not publish credentials, private reports, personal data, confidential identifiers, or unredacted user workbooks in project artifacts.

## Conduct and Appeals

Participation is governed by [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md). Moderation decisions prioritize community safety and reporter privacy. A person affected by a moderation or governance decision may request reconsideration through a private repository security advisory. A maintainer involved in the original decision should recuse themselves when another maintainer is available.

## Amendments

Governance changes require a pull request that explains motivation, impact, and migration considerations. The same review requirements apply to future amendments.
