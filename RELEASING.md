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

## Docs Hotfix

For documentation fixes that need to be published without waiting for a full release:

1. **Squash-merge your documentation PR to main via Github**

2. **Get the SHA of the merged commit**
   ```sh
   # Copy the SHA of the merged documentation commit
   git fetch origin main && git log origin/main
   ```

3. **Cherry-pick the commit onto the docs branch**

   ```sh
   git checkout docs --
   git cherry-pick <commit-hash>
   ```

4. **Push to origin**
   ```sh
   git push origin docs
   ```

5. **Verify publication**
   - Check [GitHub Commits](https://github.com/dagger/container-use/commits/docs/) to verify the docs were published successfully
   - The docs site should update automatically once the workflow completes
