Release process
===============

Building, signing, notarizing, and publishing is handled by
[`.github/workflows/release.yml`](.github/workflows/release.yml),
triggered by a tag push.

## Steps

1. Update version in the following files and commit on `master`:
    - `CHANGELOG.md`
    - `main.go`
    - `install`
    - `install.ps1`
    - `man/man1/fzf.1`
    - `man/man1/fzf-tmux.1`

2. Sign and push the tag.

    ```sh
    export V=v0.73.0
    git tag -s $V -m $V

    # Push the tag only. master on origin still points to the old version,
    # so /master/install keeps resolving against existing binaries during
    # the publish window.
    git push origin $V
    ```

3. The workflow fires on the tag push and pauses on the `release`
   environment gate. Approve it in the Actions tab to release.

4. After the GitHub release is published, fast-forward `master`:

    ```sh
    git push origin master
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

Use this to validate the workflow YAML, version-extraction logic,
the macOS runner setup, and the signing/notarization credentials.
