---
name: package-upload-prod
description: Package the current new-api repository into a production release bundle and upload it to `root@38.76.215.133:/root/code/new-api/release`. Use when asked to build a release tarball, package the Docker image, sync release artifacts, or prepare a production upload for the 133 server.
---

# Package Upload Prod

## Overview

Use this skill to turn the current working tree into a release bundle and copy it to the production server. Keep the scope to packaging and upload only; do not restart services or modify live deployment state unless the user explicitly asks.

## Workflow

1. Run `scripts/package_and_upload_prod.sh` from the repo root.
2. Build both frontends with Bun, then build `bin/new-api-prod`.
3. Reuse the local `new-api-prod-10011-new-api:latest` image if it exists; otherwise build a fresh image tar.
4. Write a release directory under `release/new-api-prod-10011-<timestamp>` with the compose file, nginx config, `Dockerfile.prod`, `INSTALL.md`, `prod.env.example`, `SHA256SUMS`, and `new-api-prod-10011-image.tar`.
5. Compress the release directory into `release/new-api-prod-10011-<timestamp>.tar.gz`.
6. Upload both the directory and the tarball to `/root/code/new-api/release` on `38.76.215.133`.
7. Verify the remote file sizes and SHA256 values after upload.

## Notes

- Keep the remote target fixed to server 133 unless the user overrides `REMOTE_HOST` or `REMOTE_DIR`.
- Expect `bun`, `go`, `docker`, `ssh`, and `rsync` to be available locally.
- If the base image is missing, the script falls back to a Docker build and may require registry access.
- Use `SKIP_UPLOAD=1` when you only need to validate the local packaging path.
