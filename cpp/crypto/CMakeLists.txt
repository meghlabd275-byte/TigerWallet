cmake_minimum_required(VERSION 3.20)
project(TigerCrypto CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Release optimizations for ultra-low latency
set(CMAKE_CXX_FLAGS_RELEASE "-O3 -march=native -mtune=native -flto")
set(CMAKE_C_FLAGS_RELEASE "-O3 -march=native -mtune=native -flto")

# The implementation lives in cpp/core, which is the wallet's crypto layer:
# SHA-256, Keccak-256, RIPEMD-160, HMAC, PBKDF2, AES-256-GCM, secp256k1,
# Ed25519, Base58 and Base58Check, Base32, BIP39 and BIP32. Each is checked
# against the published vectors for its specification.
#
# cpp/crypto_core used to sit here. It declared 32 types and implemented four,
# and nothing included or linked it, so it was removed rather than extended: a
# second half-written crypto library is a place for divergent behaviour to hide.
# The Ed25519 and Base32 pieces it only declared are now real, in cpp/core.

set(CORE_DIR "${CMAKE_CURRENT_SOURCE_DIR}/../core")
set(CORE_SOURCES
    "${CORE_DIR}/src/crypto.cpp"
    "${CORE_DIR}/src/transaction.cpp"
    "${CORE_DIR}/src/rpc.cpp"
    "${CORE_DIR}/src/mempool.cpp"
    "${CORE_DIR}/src/chain.cpp"
    "${CORE_DIR}/src/wallet.cpp"
    "${CORE_DIR}/src/hdwallet.cpp"
    "${CORE_DIR}/src/bip39_wordlist.cpp"
)

foreach(source IN LISTS CORE_SOURCES)
    if(NOT EXISTS "${source}")
        message(FATAL_ERROR "TigerCrypto: missing source ${source}")
    endif()
endforeach()

find_package(OpenSSL REQUIRED)
find_package(CURL REQUIRED)
find_package(Threads REQUIRED)

find_library(SECP256K1_LIBRARY NAMES secp256k1)
find_path(SECP256K1_INCLUDE_DIR NAMES secp256k1.h)
if(NOT SECP256K1_LIBRARY OR NOT SECP256K1_INCLUDE_DIR)
    message(FATAL_ERROR
        "TigerCrypto: libsecp256k1 was not found. Install it (Debian/Ubuntu: "
        "libsecp256k1-dev) before building.")
endif()

add_library(tiger_crypto STATIC ${CORE_SOURCES})

target_include_directories(tiger_crypto PUBLIC
    "${CORE_DIR}/include"
    "${SECP256K1_INCLUDE_DIR}"
)

target_link_libraries(tiger_crypto
    PUBLIC
        OpenSSL::Crypto
        CURL::libcurl
        Threads::Threads
        "${SECP256K1_LIBRARY}"
)

if(CMAKE_BUILD_TYPE STREQUAL "Release")
    set_target_properties(tiger_crypto PROPERTIES
        INTERPROCEDURAL_OPTIMIZATION_RELEASE TRUE)
endif()

install(TARGETS tiger_crypto ARCHIVE DESTINATION lib)
install(DIRECTORY "${CORE_DIR}/include/" DESTINATION include/tiger)
