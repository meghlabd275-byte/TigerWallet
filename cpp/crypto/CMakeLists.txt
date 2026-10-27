cmake_minimum_required(VERSION 3.20)
project(TigerCrypto CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Release optimizations for ultra-low latency
set(CMAKE_CXX_FLAGS_RELEASE "-O3 -march=native -mtune=native -flto")
set(CMAKE_C_FLAGS_RELEASE "-O3 -march=native -mtune=native -flto")

# The crypto implementation lives in cpp/crypto_core: it is self-contained
# (SHA-256, Keccak, HMAC, PBKDF2, Base58/Base58Check, secure random) and pulls
# no third-party crypto library. This target previously referenced a
# tiger_crypto.cpp that does not exist and fetched secp256k1 and Crypto++ over
# the network at configure time, so it never built. Nothing consumes it yet, so
# the stale references are dropped rather than guessed at.
set(CRYPTO_CORE_DIR "${CMAKE_CURRENT_SOURCE_DIR}/../crypto_core")
set(CRYPTO_CORE_SRC "${CRYPTO_CORE_DIR}/src/tiger_crypto.cpp")

if(NOT EXISTS "${CRYPTO_CORE_SRC}")
    message(FATAL_ERROR
        "TigerCrypto: expected implementation at ${CRYPTO_CORE_SRC}")
endif()

find_package(Threads REQUIRED)

add_library(tiger_crypto STATIC "${CRYPTO_CORE_SRC}")

target_include_directories(tiger_crypto PUBLIC
    "${CRYPTO_CORE_DIR}/include"
)

target_link_libraries(tiger_crypto PUBLIC Threads::Threads)

if(CMAKE_BUILD_TYPE STREQUAL "Release")
    set_target_properties(tiger_crypto PROPERTIES
        INTERPROCEDURAL_OPTIMIZATION_RELEASE TRUE)
endif()

install(TARGETS tiger_crypto ARCHIVE DESTINATION lib)
install(FILES "${CRYPTO_CORE_DIR}/include/tiger_crypto.h"
    DESTINATION include/tiger)
