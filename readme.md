# Captr

**Captr** is a lightweight screen capturing and recording tool written in Go. It's designed for speed, simplicity, and minimal resource usage.
(Windows OS Only)

> **Status:** Release  
> Currently supports full-screen, window-specific screenshots and full-screen recording. Supports streaming to youtube and twitch. Window specific recording is partially implemented.

## Features

- 📸 Capture full-screen screenshots  
- 🖼️ Capture specific window screenshots  
- 🎥 Screen recording
- 🎥 Window specific recording (partially implemented)
- 🎥 Stream display to twitch or youtube

## Installation
**It's a portable executable file. Doesn't need any installation.**<br><br>
**Something you should know**: On first run, it auto creates an appdata directory `captr` and puts the config file inside.

## Usage
Download the exe from the releases or [build yourself](#build-yourself).<br>
Run the exe to use it.<br>
For recording and streaming functionaility, you must have `ffmpeg` installed and added to path. If you don't have it, the app has the prebuilt prompt to download ffmpeg to the `%appdata%\captr\bin` folder in appdata to use.<br>
**For audio, please  manually configure the config file or enable "Stereo Mix" device in windows settings.**

### Flags
- `--config`: Opens the config file in the notepad for manual edits.
- `--hotkey`: Change the hotkey for stopping the recording.
- `--reset`: Deletes the `%appdata%/captr` folder.

## Roadmap
- [x] Full-Screen Screenshots
- [x] Window-Specific Screenshots
- [x] Screen Recording
- [x] Stream to Youtube and Twitch
- [x] Window Recording [Partial]

## Build Yourself
### Prerequisites
- Golang (>=1.24)
- Mingw-w64

Run `go build` to build the standalone exe file.

## Contributing
Contributions are welcome. Open an issue or submit a pr.

## License
Apache License, Version 2.0