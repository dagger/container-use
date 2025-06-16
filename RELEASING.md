# Releasing

## Steps

1. **Fetch the latest main branch**
   ```sh
   git fetch origin main
   git checkout origin/main
   ```
   NOTE: this puts you on a detached head, which is fine for tagging and pushing the tag.

2. **Tag the release**
   ```sh
   git tag v1.2.3
   ```

3. **Push the tag**
   ```sh
   git push origin v1.2.3
   ```

4. **Check the draft release**
   - Monitor the [release workflow](https://github.com/dagger/container-use/actions/workflows/release.yml) for progress and errors
   - Go to [GitHub Releases](https://github.com/dagger/container-use/releases)
   - Review the auto-generated draft release
   - Verify binaries and checksums are attached

5. **Publish the release**
   - Edit the draft release if needed
   - Click "Publish release"

6. **Merge the homebrew tap PR**
   - After publishing the release, a PR will be automatically created in [dagger/homebrew-tap](https://github.com/dagger/homebrew-tap)
   - Review and merge the PR to make the release available via Homebrew

The Dagger CI automatically handles building binaries and creating the draft release when tags are pushed.
