cmake_minimum_required(VERSION 3.20)
project(TigerWalletCore VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

# Output directories
set(CMAKE_RUNTIME_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/bin)
set(CMAKE_LIBRARY_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/lib)
set(CMAKE_ARCHIVE_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/lib)

# Find OpenSSL
find_package(OpenSSL REQUIRED)

# Find leveldb
find_package(LevelDB REQUIRED)

# Find sodium
find_package(Sodium REQUIRED)

# Headers
include_directories(
    ${CMAKE_SOURCE_DIR}/include
    ${OPENSSL_INCLUDE_DIR}
    ${LEVELDB_INCLUDE_DIR}
    ${SODIUM_INCLUDE_DIR}
)

# Core libraries
add_library(tigerwallet_core STATIC
    src/crypto/sha256.cpp
    src/crypto/aes.cpp
    src/crypto/keccak.cpp
    src/crypto/entropy.cpp
    src/wallet/wallet.cpp
    src/wallet/mnemonic.cpp
    src/wallet/derivation.cpp
    src/transaction/tx_builder.cpp
    src/transaction/signer.cpp
    src/chain/evm.cpp
    src/chain/solana.cpp
    src/chain/bitcoin.cpp
    src/chain/tron.cpp
    src/rpc/provider.cpp
    src/utils/logger.cpp
    src/utils/config.cpp
)

target_link_libraries(tigerwallet_core
    OpenSSL::SSL
    OpenSSL::Crypto
    LevelDB::LevelDB
    Sodium::sodium
    pthread
    dl
)

# Executables
add_executable(tx_processor src/tools/tx_processor.cpp)
target_link_libraries(tx_processor tigerwallet_core)

add_executable(wallet_generator src/tools/wallet_generator.cpp)
target_link_libraries(wallet_generator tigerwallet_core)

add_executable(bench_crypto src/tools/bench_crypto.cpp)
target_link_libraries(bench_crypto tigerwallet_core)

# Installation
install(TARGETS tigerwallet_core LIBRARY DESTINATION lib)
install(DIRECTORY include/ DESTINATION include/tigerwallet)
