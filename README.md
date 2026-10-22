# sub-sync
An application that syncs mistimed Advanced SubStation Alpha (`.ass`) subtitles to a video. The engine compares subtitle timing with detected speech, applying the best offset to each subtitle line. It can handle constant global shift and offset drift.

## Download and run
Download the corresponding archive from the latest [release](https://github.com/osmonduu/sub-sync/releases).
- **Linux**: extract the `.tar.xz` and run `sub-sync` from the folder.
- **Windows**: extract the `.zip` and run `sub-sync.exe` from the folder.

*Note: ffmpeg comes bundled with sub-sync so you don't need to install it separately. Keep sub-sync and ffmpeg in the same folder. The app falls back to the `PATH` if the bundled copy is missing.*

## Requirements
### Linux
The GTK3 and WebKitGTK 4.1 runtime libraries are required for sub-sync to run. If they are not installed, run the following command to install the packages:  
```bash
sudo apt install libgtk-3-0 libwebkit2gtk-4.1-0
```
### Windows
The WebView2 runtime is required for sub-sync to run. Most machines running Windows 11 will have WebView2 installed by default. If you don't have it installed, you can download it from Microsoft's [website](https://developer.microsoft.com/en-us/microsoft-edge/webview2/).

*Note: The first time you run `sub-sync.exe`, Windows will show a SmartScreen warning because the executable is not signed. Select **More info**, then **Run anyway** to start the application. The warning usually appears only the first time you launch sub-sync.*

## How to use
To sync subtitles:
1. Choose a video
2. Add at least one subtitle file (you can add multiple subtitle files to re-time based on the single video source)
3. Select **Sync all**

By default, the re-timed subtitles will be placed in a folder named `synced_subtitles` next to each input subtitle. You can change the destination of the output subtitles by specifying an output folder.

## How does sub-sync work?
sub-sync extracts the audio from the video source with a 16kHz sample rate. The engine creates a boolean timeline, with true representing active audio at that point in time above a certain threshold and false otherwise. The subtitles are also converted to a boolean timeline with true representing active subtitles at that point in time, and false otherwise.  

The audio and subtitle timelines are broken up into segments of specified length. An offset sweep is conducted to find an offset with the highest 'true' matches between the two timeline segments. If segments do not have the same offset (a sign of offset drift or inconsistent subtitle timings), interpolated offsets are calculated for each subtitle between the two segment centers. The offset is applied to each subtitle and a new `.ass` file is written out to the output folder with the suffix `_synced` appended to the original subtitle file name.

## Known limitations
- An offset larger than the search range isn't detected, and the engine may report a smaller, wrong offset. The default search range is ±5000 ms and is adjustable under the "Advanced options" dropdown menu.
- The engine matches subtitle timing against detected sound, not recognized speech, so it cannot tell when its result is wrong. A wrong offset can look as plausible as the right one.
- Only `.ass` files are supported, and one video per run.
- Drag-and-drop isn't supported.
- The releases cover Linux x86-64 and Windows x64 only. macOS and ARM are not supported.

## Build from source
The tools needed are:
- Go 1.25: https://go.dev/dl/
- Node 22: https://nodejs.org/en/about/previous-releases
- Wails CLI v2.12.0: https://github.com/wailsapp/wails/releases/tag/v2.12.0
- ffmpeg next to the binary or on `PATH`

Clone the repository. From the repository root go to the `gui/` directory. Build the application with:
### Linux
Make sure you have the dependencies installed first:
```bash
sudo apt-get install -y build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev
```
and build specifying WebKitGTK 4.1:
```bash
wails build -tags webkit2_41
```
### Windows
```bash
wails build
```
  
From the repository root, the executable is `gui/build/bin/sub-sync` on Linux and `gui/build/bin/sub-sync.exe` on Windows.

## Command-line tools
Releases ship only the GUI, so the `sub-sync-cli` and `sabotage` tools are for people who build from source. After building the executables, you can view the flag descriptions for each tool using the `-h` flag.
### sub-sync CLI
The sub-sync CLI syncs one subtitle file per run and doesn't have an output folder option.

From the repository root, build the executable:
```bash
go build -o sub-sync-cli ./cmd/cli/
```
To use the tool, run:
```bash
./sub-sync-cli -video <video_filepath> -sub <sub_filepath> -search <search_range_ms> -segment <segment_duration_min>
```
An example would be:
```bash
./sub-sync-cli -video video.mkv -sub test.ass -search 3000 -segment 3
```

The CLI accepts these flags:
| Flag | Default value | Description |
| :---: | :---: | :--- |
| `-video` | `""` | (Required) One video filepath of any format that ffmpeg can decode |
| `-sub` | `""` | (Required) One `.ass` subtitle filepath |
| `-search` | `5000` | Search range in milliseconds | 
| `-segment` | `5` | Segment length in minutes |
  
*Note:*
- *The CLI writes `<input_name>_synced.ass` next to the input file.*
- *The CLI finds `ffmpeg` through the same lookup as the GUI, so it needs a copy next to the binary or on the `PATH`.*

### sabotage
The sabotage tool is used as a test data generator that shifts a subtitle file by a known amount. You can use this tool to check that the engine is performing as developed. 

From the repository root, build the executable:
```bash
go build -o sabotage ./cmd/sabotage/
```
To use the tool, run:
```bash
./sabotage -input <input_filepath> -output <output_filepath> -mode <global/step/ramp> -offset <offset_ms> -checkpoints <timestamp_sec:offset_ms>
```
An example would be:
```bash
./sabotage -input test.ass -mode ramp -checkpoints 0:0,300:2000
```

The sabotage tool accepts these flags:
| Flag | Default value | Description |
| :---: | :---: | :--- |
| `-input` | `""` | (Required) Input `.ass` subtitle filepath |
| `-output` | `""` | Output `.ass` subtitle filepath. If no output path is provided, the tool will use the input path and add a suffix corresponding to the mode (e.g. `<input_name>_sabotaged_global_<offset>ms.ass`) as the output filename and place it next to the input file. |
| `-mode` | `global` | Three mode options: `global`, `step`, or `ramp` |
| `-offset` | `0` | Offset in milliseconds to apply. Can be negative. |
| `-checkpoints` | `""` | Comma-separated timestamp(seconds):offset(milliseconds) pairs, e.g. `0:0,300:2000,600:-1000` |

Mode options:
- `global` mode applies the same offset to every single subtitle.
- `step` applies an offset to every subtitle from its corresponding checkpoint timestamp to the next checkpoint timestamp. The last checkpoint applies its offset from its timestamp to the rest of the file. 
- `ramp` is similar to step but applies an interpolated offset based on a subtitle's position between the two nearest timestamps.  

*Note:*
- *If you are using `step` or `ramp` mode, it is required to pass in a `checkpoints` string.*
- *The offset that the tool applies is the shift that the engine must undo, so the engine's correction has the opposite sign. (e.g. a +5000ms file should sync back with a -5000ms offset applied).*
- *For instructions to download test youtube videos and subtitles, see [test_instructions.md](test_instructions.md)*

## License
sub-sync is licensed under the GNU General Public License, version 3 or later. See [LICENSE](LICENSE) for the full text. It bundles FFmpeg under the LGPL. See [NOTICE.md](third_party/ffmpeg/NOTICE.md).