#!/usr/bin/env bash
# Build verification for the TigerWallet WL control-plane C++ components.
# Builds the real sources (no stubs), runs the unit tests, then compiles and
# runs the fail-closed smoke test. Verified with g++ 14.2 / cmake 3.31.
set -u

cd "$(dirname "$0")"
CXX="${CXX:-g++}"
STD="${STD:-c++20}"   # c++17 also compiles; c++20 matches CMakeLists.txt

echo "==> 1/3 Direct compile check (no cmake required)"
$CXX -std=$STD -O2 -Wall -Wextra -c src/wl_gate.cpp -I include -o /tmp/wl_gate.o || exit 1
$CXX -std=$STD -O2 -Wall -Wextra -c src/wl_auto_approver.cpp -I include -o /tmp/wl_auto_approver.o || exit 1
echo "    OK: src/wl_gate.cpp and src/wl_auto_approver.cpp compile cleanly"

echo "==> 2/3 CMake build + ctest (or plain g++ fallback if cmake is unavailable)"
if command -v cmake >/dev/null 2>&1; then
    cmake -B build -DCMAKE_BUILD_TYPE=Release >/dev/null || exit 1
    cmake --build build -j >/dev/null || exit 1
    ctest --test-dir build --output-on-failure || exit 1
else
    echo "    cmake not found; building + running unit tests with plain g++"
    $CXX -std=$STD -O2 -I include tests/test_wl_gate.cpp src/wl_gate.cpp -o /tmp/test_wl_gate || exit 1
    $CXX -std=$STD -O2 -I include tests/test_wl_auto_approver.cpp src/wl_auto_approver.cpp -o /tmp/test_wl_auto_approver || exit 1
    /tmp/test_wl_gate || exit 1
    /tmp/test_wl_auto_approver || exit 1
fi

echo "==> 3/3 Fail-closed smoke test (invalid license must deny everything)"
$CXX -std=$STD -O2 -Wall -Wextra -I include     tests/smoke_fail_closed.cpp src/wl_gate.cpp src/wl_auto_approver.cpp     -o /tmp/smoke_fail_closed || exit 1
/tmp/smoke_fail_closed || exit 1

echo "BUILD VERIFICATION PASSED"
