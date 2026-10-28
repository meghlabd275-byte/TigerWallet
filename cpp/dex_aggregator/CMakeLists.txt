cmake_minimum_required(VERSION 3.16)
project(tigerwallet_dex_aggregator CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

find_package(OpenSSL REQUIRED)
find_package(Threads REQUIRED)
find_package(CURL REQUIRED)
find_package(PkgConfig REQUIRED)
# The shared crypto layer signs with libsecp256k1.
pkg_check_modules(SECP256K1 REQUIRED IMPORTED_TARGET libsecp256k1)

# The aggregator reads pools and prices over JSON-RPC, so it links the shared
# RPC client as well as the crypto and integer helpers.
add_library(tigerwallet_dex_aggregator STATIC dex_aggregator.cpp)
target_include_directories(tigerwallet_dex_aggregator PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}
    ${CMAKE_CURRENT_SOURCE_DIR}/../common
    ${CMAKE_CURRENT_SOURCE_DIR}/../core/include)
target_link_libraries(tigerwallet_dex_aggregator PUBLIC
    OpenSSL::Crypto Threads::Threads CURL::libcurl)

enable_testing()

add_executable(test_dex_aggregator
    tests/test_dex_aggregator.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/../core/src/rpc.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/../core/src/crypto.cpp)
target_link_libraries(test_dex_aggregator PRIVATE
    tigerwallet_dex_aggregator PkgConfig::SECP256K1)
add_test(NAME dex_aggregator COMMAND test_dex_aggregator)
