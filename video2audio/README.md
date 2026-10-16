# VideoToAudio · 视频转音频

把视频文件里的声音提取/转换为音频文件的 Windows 小工具。Go + Win32 原生开发,单文件、无依赖、内嵌 ffmpeg。

## 功能

- 🎬 **批量转换**:拖入 / 多选添加视频文件(mp4、mkv、avi、mov、flv、wmv、webm、ts、rmvb 等)
- 🎵 **多种输出格式**
  - MP3(128/192/256/320 kbps)—— 兼容性最好
  - M4A / AAC(128/192/256 kbps)—— 体积更小音质更好
  - WAV(无损未压缩)/ FLAC(无损压缩)
  - **原样提取**:不转码、无损、速度极快(按源音频编码输出 m4a/mp3/opus/ogg/flac/wav 等)
- 📂 **输出位置**:源文件夹 / 源文件夹下 `audio` 子文件夹 / 自定义文件夹
- 📊 **实时进度**:每个文件的转换百分比与整体进度条
- ⏹ **随时取消**:中途取消转换队列

## 使用

1. 下载 `VideoToAudio.exe`(单文件,已内嵌 ffmpeg,约 10 MB)
2. 双击运行,把视频拖进窗口(或点「添加文件」)
3. 选择输出格式与音质,点「开始转换」

也可以把视频文件直接拖到 `VideoToAudio.exe` 图标上启动,文件会自动加入列表。

> 若把 `ffmpeg.exe` 放在本程序同目录,将优先使用它(可自行替换为完整版 ffmpeg 以支持更多格式);
> 内嵌的 ffmpeg 会释放到 `%LOCALAPPDATA%`(缓存目录)供转换使用。

## 从源码构建

```sh
cd video2audio
sh build.sh            # 开发版(使用同目录或 PATH 里的 ffmpeg)
```

单文件正式版(内嵌 ffmpeg.exe)由 GitHub Actions 的 `VideoToAudio` 工作流构建并提交到 `dist/`;
本地交叉编译:`ci/build-ffmpeg.sh`(需 mingw-w64)后执行 `ci/build-exe.sh`。

## 技术说明

- Go + 原生 Win32 API,零第三方 GUI 依赖(与 ClipGlass 同风格)
- 转换核心调用内嵌的最小化 ffmpeg(只编入音频相关组件,~5 MB,GPLv3)
- MP3 编码使用 LAME(libmp3lame)
- 支持 Windows 10/11 x64

## 许可

本工具源码与 ClipGlass 同仓库发布;内嵌的 FFmpeg 与 LAME 遵循其各自的许可证(FFmpeg GPLv3 / LAME LGPL)。
