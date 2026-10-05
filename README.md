# ClipGlass（本人的第一个项目，希望大家喜欢）

                                                       <img width="1024" height="1024" alt="ClipGlass_icon_hd" src="https://github.com/user-attachments/assets/daccd29d-3e09-4219-b69f-e34a23485f4f" />
            


<p align="center">
  <b>轻量 · 优雅 · 安全的 Windows 剪贴板管理工具</b><br>
  Go 原生开发 · 零依赖 · 单文件运行
</p>

---

✨ 简介

ClipGlass 是一款使用 **Go 语言 + Win32 API 原生开发**的 Windows 剪贴板增强工具。它不依赖任何第三方 GUI 框架，编译产物仅为一个可执行文件，内存占用极低。

它以一枚精致的**玻璃拟态（Glassmorphism）悬浮球**常驻桌面，点击悬浮球或按下自定义快捷键即可唤出主面板，快速浏览、搜索、复用历史剪贴板内容。无论是日常办公还是开发场景，都能让"复制粘贴"变得高效而优雅。

<img width="491" height="654" alt="1" src="https://github.com/user-attachments/assets/fe4ce6e7-1747-4368-b143-f99f42ab839b" />


🚀 核心特性

📋 智能剪贴板历史
- 自动记录文本、图片等剪贴板内容，支持条数上限与保留时长设置
- 内置搜索，快速定位历史条目
- 支持置顶（Pin）、删除、忽略来源应用

🗂️ 自动类型识别
自动识别并分类十余种内容类型，支持按类型筛选：

| 类型 | 说明 |
|------|------|
| 文本 / 链接 / 代码 | 日常文本、URL、代码片段 |
| 路径 / 文件 | 本地文件与目录路径 |
| 颜色 | HEX 等颜色值，带色块预览 |
| 图片 | 剪贴板图像，缩略图展示 |
| 表格 / 数据 | 结构化数据识别 |
| 命令 / 网络 / 编号 | 命令行、IP、数字编号等 |

🎨 便捷操作
- **一键嗅探**：识别邮箱、网址、路径，快速打开、发邮件或定位文件
- **文本处理**：粘贴为纯文本、一键大小写转换
- **分组管理**：自定义分组，支持重命名与彩色标记
- **自动粘贴**：复制后自动粘贴到当前窗口（可选）

🔐 隐私与安全
- **敏感信息检测**：内置正则引擎，自动识别私钥、JWT、API Key、密码、验证码、证件号、银行卡号、钱包地址等敏感内容（含中英文口令模式）
- **敏感内容保护**：敏感条目在内存中限时保留（默认 10 分钟），可设置忽略或加密保护
- **剪贴板暂停**：一键暂停/恢复监听，也可设置暂停快捷键
- **排除规则**：忽略指定来源应用的复制内容
- **密码锁**：为数据目录设置访问密码
- **本地存储**：所有数据仅保存在本地 `%AppData%\ClipGlass`，不上传任何服务器
<img width="486" height="646" alt="2" src="https://github.com/user-attachments/assets/4fba89cd-aa10-46ba-ad0f-6edcf2437042" />

🌍 本地化
- 支持 **14 种界面语言**：简体中文、English、繁體中文、日本語、한국어、Español、Français、Deutsch、Português、Русский、Italiano、Türkçe、Tiếng Việt、Bahasa Indonesia
- 默认跟随系统语言，可手动切换
<img width="491" height="654" alt="1" src="https://github.com/user-attachments/assets/6b4d2e3c-b3d3-4601-bddc-d204b2ccb9ca" />

⚙️ 丰富设置
- **主题**：浅色 / 深色 / 跟随系统，多种强调色可选
- **外观**：玻璃模糊强度、面板透明度、UI 缩放、紧凑模式
- **悬浮球**：显示/隐藏、动画开关
- **快捷键**：主面板快捷键、暂停快捷键，支持自定义录制
- **面板位置**：屏幕中央 / 光标处 / 上次位置
- **其他**：开机自启、导出/导入备份、数据目录一键打开、恢复默认

  <img width="66" height="63" alt="4" src="https://github.com/user-attachments/assets/385f39e0-063e-46ee-b9fc-cd94109c961a" />


📦 安装

### 下载使用
从 [Releases](../../releases) 页面下载最新的 `ClipGlass.exe`，双击即可运行，无需安装。

### 从源码构建
需要 Go 1.24 及以上版本：

```bash
git clone https://github.com/yourname/clipglass.git
cd clipglass
go build -ldflags="-H windowsgui"
```

> 图标资源已通过 `rsrc_windows_amd64.syso` 嵌入，构建时自动链接。

🖥️ 系统要求

- Windows 10 / 11（64 位）
- 建议开启系统透明效果以获得最佳玻璃视觉效果

💡 使用指南

1. **启动**：双击 `ClipGlass.exe`，桌面出现玻璃悬浮球
2. **唤出面板**：点击悬浮球，或按下自定义快捷键（默认可在设置中录制）
3. **管理条目**：右键条目进行复制、粘贴、置顶、分组、删除等操作
4. **隐藏启动**：使用 `ClipGlass.exe --tray` 参数可后台静默启动
5. **托盘菜单**：右键托盘图标可显示面板、暂停监听、清空记录或退出
6. **单实例**：重复启动时会自动唤起已运行的实例

📁 数据目录

所有配置与历史数据保存在：

```
%AppData%\ClipGlass\
├── settings.json   # 设置
└── store.json      # 剪贴板历史
```

🛠️ 技术架构

- **语言**：Go 1.24，纯标准库，零第三方依赖
- **UI**：直接调用 Win32 API，自绘 DirectComposition / 分层窗口渲染
- **图形**：软件光栅化画布（BGRA 预乘 Alpha），支持亚像素抗锯齿、渐变、模糊
- **DPI 感知**：自动适配系统 DPI 缩放
- **i18n**：内置多语言表，运行时热切换

许可证

MIT License
