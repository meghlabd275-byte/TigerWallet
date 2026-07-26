cmake_minimum_required(VERSION 3.20)
project(TigerCrypto CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Release optimizations for ultra-low latency
set(CMAKE_CXX_FLAGS_RELEASE "-O3 -march=native -mtune=native -ffast-math -flto")
set(CMAKE_C_FLAGS_RELEASE "-O3 -march=native -mtune=native -ffast-math -flto")

# Find packages
find_package(Threads REQUIRED)
find_package(OpenSSL REQUIRED)

# secp256k1
include(FetchContent)
FetchContent_Declare(
    secp256k1
    GIT_REPOSITORY https://github.com/bitcoin-core/secp256k1.git
    GIT_TAG v0.4.0
    GIT_SHALLOW TRUE
)
FetchContent_MakeAvailable(secp256k1)

# Crypto++ (for AES-GCM, SHA)
include(FetchContent)
FetchContent_Declare(
    cryptopp
    GIT_REPOSITORY https://github.com/weidai11/cryptopp.git
    GIT_TAG CRYPTOPP_8_9_0
    GIT_SHALLOW TRUE
)
FetchContent_MakeAvailable(cryptopp)

# Add library
add_library(tiger_crypto STATIC 
    tiger_crypto.cpp
)

target_include_directories(tiger_crypto PUBLIC 
    ${CMAKE_CURRENT_SOURCE_DIR}
    ${secp256k1_INCLUDE_DIRS}
)

target_link_libraries(tiger_crypto PUBLIC
    secp256k1
    cryptopp
    OpenSSL::Crypto
    Threads::Threads
)

# Enable LTO for release builds
if(CMAKE_BUILD_TYPE STREQUAL "Release")
    target_property(tiger_crypto APPEND_STRING 
        COMPILE_FLAGS " -flto")
endif()

# Benchmark executable
add_executable(crypto_benchmark benchmark.cpp)
target_link_libraries(crypto_benchmark tiger_crypto)
