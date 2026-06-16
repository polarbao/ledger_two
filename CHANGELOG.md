# Changelog

All notable changes to this project will be documented in this file.

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
- Upgrading and Rollback procedures outlined in `docs/tech/08-nas-deployment.md`.

---
*Note: This version is restricted to a maximum of 2 users per ledger as per the MVP scope.*
