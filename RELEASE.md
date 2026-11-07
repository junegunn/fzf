Release process
===============

Building, signing, notarizing, and publishing is handled by
[`.github/workflows/release.yml`](.github/workflows/release.yml),
triggered by a tag push.

## Steps

1. Sign in at https://developer.apple.com/account and accept any pending
   agreements (e.g. an updated Apple Developer Program License Agreement).
   Otherwise notarization fails.

2. Update version in the following files and commit on `master`:
    - `CHANGELOG.md`
    - `main.go`
    - `install`
    - `install.ps1`
    - `man/man1/fzf.1`
    - `man/man1/fzf-tmux.1`

3. Verify file consistency, sign the tag, and push the tag.

    ```sh
    make tag VERSION=0.75.0
    ```

    `make tag` runs `prerelease` first (checks that the version
    appears in CHANGELOG.md, both man pages, install, and install.ps1)
    and only signs + pushes the tag if the checks pass.

    Only the tag is pushed; `master` on origin still points to the
    old version, so `/master/install` keeps resolving against existing
    binaries during the publish window.

4. The workflow fires on the tag push and pauses on the `release`
   environment gate. Approve it in the Actions tab to release.
    - goreleaser creates a draft release
    - The `msi` job builds the Windows MSIs, uploads them, and publishes
      the release
    - The `winget` job submits the release to winget

5. After the `msi` job publishes the GitHub release, fast-forward `master`:

    ```sh
    git push origin master
    ```

### If the `msi` job fails

The release stays a draft. Do not push `master` until it is published.

- Fix the problem and re-run the failed jobs, or
- Publish without MSIs and submit to winget by hand:

    ```sh
    gh release edit v0.75.0 --draft=false
    gh workflow run winget.yml -f release-tag=v0.75.0
    ```

## Testing the workflow

To exercise the workflow without firing a real release:

1. Actions tab -> **Release** -> **Run workflow**.
2. Pick a branch and enter the version currently on that branch
   (the version-consistency check requires the input to match the
   files in the checked-out tree).
3. Approve the `release` environment gate when prompted.
4. Goreleaser runs with `--snapshot --skip=publish`. Signing and
   notarization run; only the GitHub release upload is skipped.
5. The `msi` job builds the MSIs and uploads them as the `windows-msi`
   artifact of the run. The release upload and the `winget` job are
   skipped.

Use this to validate the workflow YAML, version-extraction logic,
the macOS runner setup, and the signing/notarization credentials.
