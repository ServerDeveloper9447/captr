# Captr

**Captr** is a lightweight screen capturing and recording tool written in Go. It's designed for speed, simplicity, and minimal resource usage.
(Windows OS Only)

> **Status:** Beta  
> Currently supports full-screen and window-specific screenshots. Recording functionality is in development.

## Features

- 📸 Capture full-screen screenshots  
- 🖼️ Capture specific window screenshots  
- 🎥 Screen recording (coming soon)

## Installation
On first run, it auto creates an appdata directory `captr` and puts the config file inside. No separate installation required.

## Usage
Download the exe from the releases or [build yourself]()<br>
Run the exe to use it or pass the `--config` flag to open the config file.<br>
For recording functionaility, you must have `ffmpeg` installed and added to path. If you don't have it, the app has the prebuilt prompt to download ffmpeg to the `%appdata%\captr\bin` folder in appdata to use.

## Roadmap
- [x] Full-Screen Screenshots
- [x] Window-Specific Screenshots
- [ ] Screen Recording
- [ ] Window Recording

## Build Yourself
### Prerequisites
- Golang (>=1.22)
- Mingw-w64

Run `go build` to build the standalone exe file.

## Contributing
Contributions are welcome. Open an issue or submit a pr.

## License
Apache License, Version 2.0