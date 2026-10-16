# Third-party notices
sub-sync bundles FFmpeg. This file records what it is, how sub-sync uses it, where it came from, and where to find its source.

## How FFmpeg is used
- The executable name is `ffmpeg` on Linux and `ffmpeg.exe` on Windows.
- FFmpeg is included as an unmodified standalone executable.
- sub-sync starts FFmpeg as a separate program to decode the input video's audio track.

## Build
- FFmpeg version: `n9.0.2-27-g7e93014c5b-20261012`
- Variant: LGPL, static, release line 9.0
- Archive names:
    - Linux: `ffmpeg-n9.0.2-27-g7e93014c5b-linux64-lgpl-9.0.tar.xz`
    - Windows: `ffmpeg-n9.0.2-27-g7e93014c5b-win64-lgpl-9.0.zip`

## File origin
- Repository: https://github.com/BtbN/FFmpeg-Builds
- Release tag: `autobuild-2026-10-12-13-12`
- Mirror release: https://github.com/osmonduu/sub-sync/releases/tag/deps-ffmpeg-n9.0.2-27-g7e93014c5b
- Mirror release tag: `deps-ffmpeg-n9.0.2-27-g7e93014c5b`

## Source code
The following links identify the exact source of the binaries that sub-sync ships.
- FFmpeg source: https://github.com/FFmpeg/FFmpeg/commit/7e93014c5b
- FFmpeg build script: https://github.com/BtbN/FFmpeg-Builds/commit/63ac3265b0789ea16c7b1f8351cd2ad195a28722

## Replacing FFmpeg
Users can substitute their own `ffmpeg` build by placing it next to the sub-sync executable. The file must keep the name `ffmpeg` (`ffmpeg.exe` on Windows). The app looks next to the executable first and falls back to `PATH` if it finds nothing there.

## License
- FFmpeg is licensed under the LGPL version 3 or later as described via `ffmpeg -L`. To read the full license text, refer to `ffmpeg-LICENSE.txt` included in this package.
- sub-sync is licensed under the GPLv3. To read the full license text, refer to `LICENSE` included in this package.
