# Releasing

## Steps

1. **Tag the release**
   ```sh
   git tag v1.2.3
   ```

2. **Push the tag**
   ```sh
   git push origin v1.2.3
   ```

3. **Check the draft release**
   - Go to [GitHub Releases](https://github.com/dagger/container-use/releases)
   - Review the auto-generated draft release
   - Verify binaries and checksums are attached

4. **Publish the release**
   - Edit the draft release if needed
   - Click "Publish release"

The Dagger CI automatically handles building binaries and creating the draft release when tags are pushed.