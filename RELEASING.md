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

The Dagger CI automatically handles building binaries and creating the draft release when tags are pushed.

## Docs Hotfix

Publishing a release also publishes the docs branch (via the
[publish-docs workflow](https://github.com/dagger/container-use/actions/workflows/publish-docs.yml)).
For documentation fixes that need to be published without waiting for a full release:

1. **Merge your documentation PR to main**

2. **Run the publish-docs workflow manually**
   ```sh
   gh workflow run publish-docs.yml -R dagger/container-use
   ```

3. **Verify publication**
   - Check the [docs branch commits](https://github.com/dagger/container-use/commits/docs/) to verify the docs were published successfully
