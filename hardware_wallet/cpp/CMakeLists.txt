cmake_minimum_required(VERSION 3.16)

project(tiger_hardware_wallet CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

if(NOT CMAKE_BUILD_TYPE)
    set(CMAKE_BUILD_TYPE Release)
endif()

# OpenSSL supplies SHA-256, RIPEMD-160, Keccak-256, HMAC, PBKDF2 and the
# secp256k1 curve. Without it the module cannot do its job, so a missing
# library is a hard error rather than a silent fallback.
find_package(OpenSSL 3.0 REQUIRED)
find_package(Threads REQUIRED)

add_library(tiger_hardware_wallet STATIC
    src/hardware_wallet.cpp
    src/json_value.cpp
    src/json_rpc.cpp)
target_include_directories(tiger_hardware_wallet PUBLIC include)
target_link_libraries(tiger_hardware_wallet PUBLIC OpenSSL::SSL OpenSSL::Crypto Threads::Threads)

target_compile_options(tiger_hardware_wallet PRIVATE
    -Wall -Wextra -Wpedantic -Wconversion -Wsign-conversion)

enable_testing()

add_executable(test_hardware_wallet tests/test_hardware_wallet.cpp)
target_link_libraries(test_hardware_wallet PRIVATE tiger_hardware_wallet)

target_compile_options(test_hardware_wallet PRIVATE -Wall -Wextra)

add_test(NAME hardware_wallet COMMAND test_hardware_wallet)
