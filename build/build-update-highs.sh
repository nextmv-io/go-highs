# ================= README =================
# Builds all highs static libraries
# Optional parameters.
# CROSS_COMPILATION_TARGET: non-empty string for cross compilation. Currently
# only supporeted values are "" or "linux-arm64" if GOOS="linux"
# HIGHS_CXXFLAGS: Can be used to pass additional CXX flags to cmake
# HIGHS_CFLAGS: Can be used to pass additional C flags to cmake


echo "🐰 Checking cmake dependency"
if command -v cmake >/dev/null 2>&1 ; then
    echo "🐰 cmake dependency found"
else
    echo "❌ cmake not found"
    echo "❌ You must have cmake installed to run this script"
    exit 1
fi

echo "🐰 Checking zig dependency"
if command -v zig >/dev/null 2>&1 ; then
    echo "🐰 zig dependency found"
else
    echo "❌ zig not found"
    echo "❌ You must have zig installed to run this script"
    exit 1
fi

# Change to the directory of this script.
cd "$(dirname "$0")/.."
echo "🐰 Current dir is: $(pwd)"

OUT_PATH="external/highs"
HIGHS_COMMIT=$(cat build/COMMIT)
HIGHS_WD="./$OUT_PATH"

GOVERSION=$(go env GOVERSION)
GOARCH=$(go env GOARCH)
GOOS=$(go env GOOS)
TARGET="$GOOS-$GOARCH"
if [ "$CROSS_COMPILATION_TARGET" != "" ]; then
    TARGET=$CROSS_COMPILATION_TARGET
fi

echo "🐰 Working directory: $HIGHS_WD"
echo "🐰 Target : $TARGET"
echo "🐰 GOVERSION : $GOVERSION"
echo "🐰 GOARCH : $GOARCH"
echo "🐰 GOOS : $GOOS"
echo "🐰 System C information"
if [ "$GOOS" == "linux" ]; then
    ldd --version
    gcc -v
fi
if [ "$GOOS" == "darwin" ]; then
    clang -v
fi
ld -v
echo "🐰 Godspeed!"

if [ -d "$HIGHS_WD" ]; then
    mkdir -p "$HIGHS_WD"
fi

if [ -d "$HIGHS_WD/HiGHS" ]; then
    echo "🐰 Deleting previous version of HiGHS"
    rm -rf "$HIGHS_WD/HiGHS"
fi

mkdir -p "$HIGHS_WD"
cd "$HIGHS_WD"
echo "🐰 Cloning HiGHS"
git clone https://github.com/ERGO-Code/HiGHS.git
pushd HiGHS
echo "🐰 Checkout specific HiGHS commit"
git checkout $HIGHS_COMMIT
git apply --whitespace=fix "./plugins/mip/solvers/highs/highs.patch"
if [ "$CROSS_COMPILATION_TARGET" != "" ]; then
    git apply --whitespace=fix "./plugins/mip/solvers/highs/highs-no-tests.patch"
fi
popd

HIGHS_CXXFLAGS=""
HIGHS_CFLAGS=""
OUT="$HIGHS_WD/$TARGET"
pushd "$HIGHS_WD/HiGHS"
if [ -d "build" ]; then
    rm -rf build
fi
mkdir build
pushd build
if [ "$GOOS" = "darwin" ] && [ "$CROSS_COMPILATION_TARGET" == "" ]; then
    echo "🐰 Building for $TARGET"
    export CXX="clang++"
    export CC="clang"
    HIGHS_CXXFLAGS="-mmacosx-version-min=11.0" #>= big sur fixes two failing tests on highs
    HIGHS_CFLAGS="-mmacosx-version-min=11.0" #>= big sur fixes two failing tests on highs
    export HIGHS_NO_POPCNT=1
elif [ "$GOOS" = "linux" ] && [ "$CROSS_COMPILATION_TARGET" == "" ]; then
    echo "🐰 Building for $TARGET"
    export CXX="g++"
    export CC="gcc"
    export HIGHS_NO_POPCNT=0
elif [ "$GOOS" = "linux" ] && [ "$CROSS_COMPILATION_TARGET" == "linux-arm64" ]; then
    echo "🐰 Cross-compiling for $CROSS_COMPILATION_TARGET on $GOOS-$GOARCH"
    export CXX="zig c++ -target aarch64-linux-gnu"
    export CC="zig cc -target aarch64-linux-gnu"
    export HIGHS_NO_POPCNT=0
else
    echo "❌ OS/TARGET not supported"
    exit 1
fi

# add -DCMAKE_VERBOSE_MAKEFILE:BOOL=ON if you want to enable cmake verbose mode
cmake .. -DFAST_BUILD=off -DSHARED=off -DBUILD_SHARED_LIBS=off \
-DCMAKE_INSTALL_PREFIX=install -DCMAKE_BUILD_TYPE=Release \
-DCMAKE_CXX_FLAGS="$HIGHS_CXXFLAGS" -DCMAKE_C_FLAGS="$HIGHS_CFLAGS"
cmake --build . --parallel --config Release

# Test if we do not cross compile
if [ "$CROSS_COMPILATION_TARGET" == "" ]; then
    ctest --parallel 2 --timeout 300 --output-on-failure -C Release
fi

make install
if [ -d $OUT ]; then
        rm -rf $OUT
fi
mkdir $OUT
mkdir "$OUT/lib"
mkdir "$OUT/include"
cp install/lib/libhighs.a "$OUT/lib/libhighs.a"
cp -R install/include/highs "$OUT/include"
