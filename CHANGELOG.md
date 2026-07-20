# Changelog

All notable changes to this project will be documented in this file.

## [v1.2.0-rc] - 2026-07-10

This is the v1.2 release candidate for local and NAS acceptance. It is not yet
marked as the final stable release.

### Added

- CSV import preview for WeChat Pay, Alipay, and generic fixtures without
  writing preview data into formal transactions.
- Row-level import adjustment, duplicate detection, suspicious-row
  confirmation, transactional commit, rollback feedback, and import audit
  records.
- Import rule creation, editing, priority, amount range, archive/restore,
  category/account/tag suggestions, and rule-hit explanations.
- Ledger-scoped RBAC, protected attachment access, metadata archive/restore,
  system diagnostics, and stronger production configuration validation.
- Faster transaction entry, save-and-continue, transaction copy, reusable
  templates, recurring-bill confirmation, and explainable settlement details.

### Changed

- Import and import-rule operations are Owner-only in v1.2.
- Archived rules, or rules referring to archived categories, accounts, or
  tags, no longer provide import suggestions.
- Manual row selections always take precedence over rule suggestions.
- Mobile high-frequency paths were tightened for 375px, 390px, and 430px
  viewports.
- The runtime database schema is now version `18`.

### Compatibility

- Existing `/api/transactions/import/*` routes and
  `DELETE /api/import-rules/{ruleID}` remain transitional compatibility
  entries. New clients should use `/api/imports/*` and archive/restore APIs.
- OCR, bank synchronization, automatic suspicious-row submission, direct
  payment notifications, and import-batch undo are not part of v1.2.
- Upgrade and rollback instructions are documented in
  `docs/releases/v1.2.0-rc-升级与回滚指南.md`.

## [v1.0.0] - 2026-06-16

### 🎉 Initial Public Release

Welcome to the first stable release of LedgerTwo!
LedgerTwo is a localized, privacy-first, shared accounting Web tool built specifically for two users, ideal for couples, roommates, or partners.

#### ✨ Features
- **Core Ledger**: Fixed two-user shared ledger architecture (Creator & Partner).
- **Transaction Management**: Record income and expenses with support for private, partner-readable, and fully shared visibility.
- **Shared Expenses & Splits**: Advanced multi-person splitting methods including:
  - Equal Split (平分)
  - Exact Amount (按金额)
  - Ratio/Percentage (按比例)
  - Shares (按份额)
- **Settlement Center**: Automated real-time net-balance calculations. Generates smart transfer suggestions to minimize transactions between participants.
- **Data Safety & Backup**:
  - Secure active SQLite backups (`VACUUM INTO`) without locking the database.
  - Three-step safety confirmation for restoring backups, preventing accidental data loss.
  - Export capabilities for CSV (Excel friendly) and JSON (full anonymized database dump).
- **Cross-Platform PWA**: Optimized responsive UI for Desktop, Tablet, and Mobile devices (375px+). Installable as a Progressive Web App (PWA) with offline draft support.
- **Offline Drafts**: Continue recording expenses even when the network drops. Drafts are safely cached in the browser's LocalStorage and wait for manual synchronization once online.

#### 🔧 Deployment
- Fully supported Docker Compose deployment tailored for Synology NAS environments.
- SQLite backend ensuring zero-configuration lightweight deployment.
- Seamless database migration workflow using `goose`.

#### 📚 Documentation
- Comprehensive documentation covering Product Requirements (PRD), UI Specs, Technical Implementations, and NAS Deployment.
- Upgrading and Rollback procedures outlined in `docs/tech/08-NAS部署方案.md`.

---
*Note: This version is restricted to a maximum of 2 users per ledger as per the MVP scope.*
