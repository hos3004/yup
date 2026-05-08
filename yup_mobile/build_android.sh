#!/bin/bash
# Build Rust crate for all Android targets and copy to Flutter jniLibs
# Requires: Rust Android targets, Android NDK 27

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RUST_DIR="$SCRIPT_DIR/rust"
JNI_DIR="$SCRIPT_DIR/android/app/src/main/jniLibs"

declare -A TARGET_MAP=(
  ["aarch64-linux-android"]="arm64-v8a"
  ["armv7-linux-androideabi"]="armeabi-v7a"
  ["x86_64-linux-android"]="x86_64"
)

cd "$RUST_DIR"

for TARGET in "${!TARGET_MAP[@]}"; do
  ABI="${TARGET_MAP[$TARGET]}"
  echo "Building for $TARGET ($ABI)..."
  cargo +stable-gnu build --release --target "$TARGET"
  mkdir -p "$JNI_DIR/$ABI"
  cp "target/$TARGET/release/libyup_crypto.so" "$JNI_DIR/$ABI/"
  echo "  -> $JNI_DIR/$ABI/libyup_crypto.so"
done

echo "Done. All .so files copied to jniLibs."
