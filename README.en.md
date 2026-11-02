<p align="center"><img src="assets/wordmark-en.svg" width="560" alt="Paperwork — room to think, structure to remember"></p>
<p align="center"><a href="README.md">Português (Brasil)</a> · <strong>English</strong></p>

Paperwork connects handwritten notes with your knowledge space in **SilverBullet 2.11.1**. Its workflow preserves the original document, prepares a transcription for review, and incorporates the approved content into existing notes. You can transcribe manually or configure an AI provider.

The project brings daily notes, Zotero references, and scanned documents into a workflow guided by the user. Digital writing and reviewed handwriting share the same `Daily/YYYY-MM-DD` pages. Stable identifiers, explicit review, and source records help maintain continuity across your notes.

The implementation uses **Go** on the server, **TypeScript** as the interface source, and native Space Lua integration. Paperwork can run independently or optionally share connections with Relay.

**Current status:** experimental MVP. Automated validation uses synthetic documents and simulated services. Recognition quality for personal handwriting and compatibility across all platforms have not been established. Incorporation always requires review.

## Working with Paperwork

1. Open **Paperwork: Configurações** to select manual transcription or configure an AI provider.
2. Open **Paperwork: Inbox** to import a multipage PDF, PNG, JPEG, or WebP, or choose **Fotografar página** to capture a page with the camera. The photo preview enables **Usar esta foto**, which opens the review step.
3. Transcribe the document and review the text, destination date or page, structure, and unresolved entries.
4. Confirm the review and incorporate the approved content into your notes.

Original files remain preserved. Repeating a completed incorporation does not duplicate its blocks. Changing the structure of an already incorporated document requires manual reconciliation in this MVP.

Camera access depends on HTTPS or localhost, browser support, and permission. A file picker provides a fallback. Each captured photo is a separate original; use a multipage PDF when several pages need shared context. The current workflow does not provide automatic photo-to-PDF assembly, perspective correction, or a multipage scanner. Maximum original size: **18 MB**.

AI integrations use provider APIs and server-side credentials. Browser subscriptions do not supply API access. External processing must be enabled in settings and authorized for the document. Importing a file or choosing a provider does not send it for transcription. Handwriting recognition remains a task to review, not a verified transcription.

## Daily notes and Zotero

Paperwork owns daily note reconciliation and Zotero search, import, citations, and synchronization. Existing command names are retained:

- **Daily: Abrir hoje** — open today's daily note.
- **Vikunja: Sincronizar Daily atual** — reconcile checkboxes with Vikunja.
- **Zotero: Buscar biblioteca**, or `/cite` — search and work with references.

Standalone configuration supports timezone, SilverBullet URL/token, Vikunja URL/token/Inbox, and a Zotero user or group ID/key. Enable background jobs explicitly with `enabled.daily` and `enabled.zotero`. Examples keep them disabled. Intervals are set in seconds through `daily_interval` and `zotero_interval`.

With Relay, connections may be shared through `PAPERWORK_RELAY_CONFIG` and `PAPERWORK_RELAY_SETTINGS`. Relay forwards legacy Daily/Zotero routes and does not run their jobs.

## Build and install

Use Go 1.26+ and Node.js for the TypeScript build; the test harnesses require a Node.js release with native TypeScript execution. JavaScript files distributed under `Library/Scripts` and `zotero` are generated runtime artifacts, marked as generated for GitHub. There is no Python backend or Python runtime dependency in the package.

```sh
npm ci
npm run build
npm run test:zotero
go test -p 1 ./...
go build -o bin/paperwork ./cmd/paperwork
go build -o bin/paperwork-state ./cmd/paperwork-state
go build -o bin/sb-tools ./cmd/sb-tools
bin/sb-tools package --kind paperwork --source . --binary bin/paperwork --destination /path/to/new-installation
bin/sb-tools bootstrap --package /path/to/new-installation
```

The generated installation contains a Dockerfile, Compose file, native libraries, and Go binary. Configure credentials and Space access before enabling jobs. The server listens on loopback at `127.0.0.1:8089`; `PAPERWORK_HOME`, `PAPERWORK_CONFIG`, and `PAPERWORK_LISTEN` customize its paths and address. Credentials remain outside Markdown and are not returned to the browser.

`npm run build:zotero` builds the Zotero extension with the Go archive tool. `npm run test:camera` uses Chromium with simulated camera input; set `CHROMIUM_PATH` when needed. These checks do not send personal documents to AI services.

## State and backups

The backup action captures originals, SQLite, and notes with a hash manifest. Keep active SQLite state on persistent local storage rather than a concurrently synchronized folder.

For migration, stop both old and new workers and create a consistent SQLite snapshot. `paperwork-state --source SNAPSHOT` reports aggregate counts. Adding `--destination EXISTING_LIBRARY --workers-stopped` transfers legacy rows into separate `relay_*` tables, preserving IDs, hashes, and raw values. The operator must verify workers are stopped: the flag does not inspect external processes.

Rollback requires reconciling any newer state. Do not overwrite an active library with a stale snapshot.

Generated packages include the native `Std` library aligned with SilverBullet 2.11.1. It provides the Space Lua APIs used by commands and widgets; installing only the backend does not register the interface. Reload the client after updating the libraries.
