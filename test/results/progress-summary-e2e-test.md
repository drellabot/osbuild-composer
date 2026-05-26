# End-to-End Test: Progress Summary Feature

**Date:** 2026-05-26 (updated, originally 2026-05-21)
**Distribution:** fedora-43
**Architecture:** x86_64
**Image Type:** guest-image (qcow2)
**Upload Target:** local

## PRs Under Test

| Repository | PR | Title |
|---|---|---|
| drellabot/osbuild | [#1](https://github.com/drellabot/osbuild/pull/1) | monitor: emit stage summary in JSON-seq output |
| drellabot/images | [#1](https://github.com/drellabot/images/pull/1) | osbuild/monitor: parse and expose stage summaries from JSON-seq output |
| drellabot/osbuild-composer | [#1](https://github.com/drellabot/osbuild-composer/pull/1) | Add human-readable activity summaries to Cloud API v2 progress |

## Test Setup

1. Cloned all three repositories at their PR branches
2. Installed osbuild from the PR branch (`pip install` to overlay the stage summary changes on top of the Fedora 43 system package)
3. Built osbuild-composer and osbuild-worker from the PR branch
4. Installed as systemd services using socket activation
5. Started a compose via the Cloud API v2 `POST /compose` endpoint
6. Polled `GET /composes/{id}` every 30 seconds to capture progress with the new `summary` field

## Progress Polling Results (every 30 seconds)

### Run 2 (2026-05-26, with improved summary surfacing)

**Compose ID:** b13af893-e3d7-4e78-8ed3-589105d2b735

| Timestamp | Done | Total | Summary |
|-----------|------|-------|---------|
| 2026-05-26T09:39:37Z | N/A | N/A | status=pending |
| 2026-05-26T09:40:07Z | N/A | N/A | status=pending |
| 2026-05-26T09:40:38Z | 0 | 5 | Preparing sources |
| 2026-05-26T09:41:08Z | 0 | 5 | Preparing sources |
| 2026-05-26T09:41:38Z | 1 | 5 | Verify, and install RPM packages |
| 2026-05-26T09:42:08Z | 1 | 5 | Verify, and install RPM packages |
| 2026-05-26T09:42:38Z | 1 | 5 | Verify, and install RPM packages |
| 2026-05-26T09:43:09Z | 1 | 5 | Verify, and install RPM packages |
| 2026-05-26T09:43:39Z | 2 | 5 | Verify, and install RPM packages |
| 2026-05-26T09:44:09Z | 0 | 0 | Uploading to Worker Server |
| 2026-05-26T09:44:39Z | N/A | N/A | status=success |

### Run 1 (2026-05-21, original PR code)

**Compose ID:** a5af7729-27cc-4fdc-b014-149297745ab0

| Timestamp | Done | Total | Summary |
|-----------|------|-------|---------|
| 2026-05-21T21:30:35Z | 0 | 5 | Starting pipeline source org.osbuild.curl |
| 2026-05-21T21:31:06Z | 0 | 5 | Starting pipeline source org.osbuild.curl |
| 2026-05-21T21:31:36Z | 0 | 5 | Starting pipeline source org.osbuild.curl |
| 2026-05-21T21:32:06Z | 0 | 5 | Starting pipeline source org.osbuild.curl |
| 2026-05-21T21:32:36Z | 2 | 5 | Finished module org.osbuild.rpm [sub: Verify, and install RPM packages (2/12)] |
| 2026-05-21T21:33:06Z | 0 | 0 | Uploading to Worker Server |
| 2026-05-21T21:33:36Z | N/A | N/A | status=success |

## Code Changes

Two changes were made to `internal/osbuildexecutor/runner-common.go`:

1. **Source pipeline mapped to "Preparing sources"**: When the osbuild pipeline name
   starts with "source", the top-level summary is set to "Preparing sources" instead
   of surfacing the raw osbuild monitor message (e.g., "Starting pipeline source org.osbuild.curl").

2. **Sub-progress summary surfaced as top-level message**: When a human-readable
   sub-progress summary is available (from stage metadata), it replaces the pipeline-level
   message as the top-level summary. This means the API consumer always sees the
   human-readable description (e.g., "Verify, and install RPM packages") rather than
   the technical pipeline name.

## Final Compose Status

```json
{
    "href": "/api/image-builder-composer/v2/composes/b13af893-e3d7-4e78-8ed3-589105d2b735",
    "id": "b13af893-e3d7-4e78-8ed3-589105d2b735",
    "image_status": {
        "status": "success",
        "upload_status": {
            "options": {
                "artifact_path": "/var/lib/osbuild-composer/artifacts/b13af893-e3d7-4e78-8ed3-589105d2b735/disk.qcow2"
            },
            "status": "success",
            "type": "local"
        },
        "upload_statuses": [
            {
                "options": {
                    "artifact_path": "/var/lib/osbuild-composer/artifacts/b13af893-e3d7-4e78-8ed3-589105d2b735/disk.qcow2"
                },
                "status": "success",
                "type": "local"
            }
        ]
    },
    "kind": "ComposeStatus",
    "status": "success"
}
```

## Result: PASS

The progress summary feature works correctly end-to-end. All compose phases now show
human-readable summaries: "Preparing sources", "Verify, and install RPM packages",
"Uploading to Worker Server".
