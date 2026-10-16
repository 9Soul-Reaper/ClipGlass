#!/bin/bash
# 交叉编译一个"最小可用"的 Windows ffmpeg.exe(只含提取/转换音频所需的组件)。
# 需要: mingw-w64, curl, tar, make, gcc
set -euxo pipefail
cd "$(dirname "$0")/.."   # video2audio/

BUILD="$PWD/.build"
OUT="$BUILD/ffmpeg.exe"
if [ -f "$OUT" ]; then
	echo "==> ffmpeg.exe 已存在($OUT),跳过构建"
	exit 0
fi

mkdir -p "$BUILD/src"
PREFIX_LAME="$BUILD/lame"

# ---- LAME (libmp3lame) ----
if [ ! -f "$PREFIX_LAME/lib/libmp3lame.a" ]; then
	echo "==> 下载 LAME 源码"
	cd "$BUILD/src"
	if [ ! -d lame ]; then
		curl -sL -o lame.tar.gz "https://codeload.github.com/zlargon/lame/tar.gz/refs/heads/master"
		tar xf lame.tar.gz
		mv lame-master lame
	fi
	# 旧版 config.guess/config.sub 无法识别 x86_64-w64-mingw32,换成系统里的现代版本
	for d in /usr/share/misc /usr/share/automake* /usr/share/libtool/build-aux; do
		if [ -f "$d/config.sub" ]; then cp -f "$d/config.sub" lame/config.sub; break; fi
	done
	for d in /usr/share/misc /usr/share/automake* /usr/share/libtool/build-aux; do
		if [ -f "$d/config.guess" ]; then cp -f "$d/config.guess" lame/config.guess; break; fi
	done
	if ! lame/config.sub x86_64-w64-mingw32 >/dev/null 2>&1; then
		echo "==> 从 savannah 更新 config.sub/config.guess"
		curl -sL -o lame/config.sub "https://git.savannah.gnu.org/cgit/config.git/plain/config.sub"
		curl -sL -o lame/config.guess "https://git.savannah.gnu.org/cgit/config.git/plain/config.guess"
	fi
	chmod +x lame/config.sub lame/config.guess
	lame/config.sub x86_64-w64-mingw32
	cd lame
	echo "==> 编译 LAME (mingw 静态库)"
	./configure --host=x86_64-w64-mingw32 --disable-shared --enable-static \
		--disable-frontend --disable-gtktest --prefix="$PREFIX_LAME"
	make -j"$(nproc)"
	make install
	test -f "$PREFIX_LAME/lib/libmp3lame.a"
fi

# ---- FFmpeg ----
echo "==> 下载 FFmpeg n7.1.5 源码"
cd "$BUILD/src"
if [ ! -d ffmpeg ]; then
	curl -sL -o ffmpeg.tar.gz "https://codeload.github.com/FFmpeg/FFmpeg/tar.gz/refs/tags/n7.1.5"
	tar xf ffmpeg.tar.gz
	mv FFmpeg-n7.1.5 ffmpeg
fi
cd ffmpeg

echo "==> 配置 FFmpeg(最小音频组件集, mingw 静态)"
./configure \
	--target-os=mingw32 --arch=x86_64 --cross-prefix=x86_64-w64-mingw32- \
	--pkg-config=false \
	--extra-cflags="-I$PREFIX_LAME/include" \
	--extra-ldflags="-L$PREFIX_LAME/lib -static" \
	--extra-libs="-lmp3lame" \
	--enable-static --disable-shared \
	--disable-debug --disable-doc --disable-ffplay --disable-ffprobe \
	--disable-network --disable-autodetect --disable-everything --disable-x86asm \
	--enable-gpl --enable-version3 --enable-libmp3lame \
	--enable-avcodec --enable-avformat --enable-avutil --enable-swresample \
	--enable-protocol=file --enable-protocol=pipe \
	--enable-demuxer=mov,matroska,avi,mpegts,flv,asf,ogg,wav,mp3,aac,ac3,mpeg,rm \
	--enable-muxer=mp3,ipod,mov,wav,flac,adts,ogg,matroska,ac3,eac3,asf,mp2 \
	--enable-decoder=aac,aac_latm,mp3,mp3float,ac3,eac3,dca,opus,vorbis,flac,alac,wmav1,wmav2,mp2,mp1,amrnb,amrwb,ra_144,ra_28,sipr,cook,pcm_s16le,pcm_s16be,pcm_s24le,pcm_s24be,pcm_s32le,pcm_u8,pcm_f32le,pcm_mulaw,pcm_alaw,adpcm_ms,adpcm_ima_wav,adpcm_ima_qt \
	--enable-encoder=libmp3lame,aac,flac,pcm_s16le,pcm_s24le \
	--enable-parser=aac,aac_latm,mpegaudio,ac3,dca,opus,vorbis,flac,alac \
	--enable-bsf=aac_adtstoasc \
	--enable-filter=aresample,anull

echo "==> 编译 ffmpeg.exe"
make -j"$(nproc)"
ls -la ffmpeg*.exe || true

cp -f ffmpeg.exe "$OUT"
echo "==> 完成: $OUT ($(du -h "$OUT" | cut -f1))"
