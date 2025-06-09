# Releasing

## Steps

1. **Fetch the latest main branch**
   ```sh
   git checkout main
   git pull origin main
   ```

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
