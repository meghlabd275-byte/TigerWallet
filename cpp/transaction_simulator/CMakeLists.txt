cmake_minimum_required(VERSION 3.16)
project(tigerwallet_transaction_simulator CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

find_package(OpenSSL REQUIRED)
find_package(Threads REQUIRED)
find_package(CURL REQUIRED)
find_package(PkgConfig REQUIRED)
# The shared crypto layer signs with libsecp256k1.
pkg_check_modules(SECP256K1 REQUIRED IMPORTED_TARGET libsecp256k1)

# A simulation runs the transaction on a node, so the module links the shared
# RPC client, and it hashes with the shared crypto and integer helpers.
add_library(tigerwallet_transaction_simulator STATIC transaction_simulator.cpp)
target_include_directories(tigerwallet_transaction_simulator PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}
    ${CMAKE_CURRENT_SOURCE_DIR}/../common
    ${CMAKE_CURRENT_SOURCE_DIR}/../core/include)
target_link_libraries(tigerwallet_transaction_simulator PUBLIC
    OpenSSL::Crypto Threads::Threads CURL::libcurl)

enable_testing()

add_executable(test_transaction_simulator
    tests/test_transaction_simulator.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/../core/src/rpc.cpp
    ${CMAKE_CURRENT_SOURCE_DIR}/../core/src/crypto.cpp)
target_link_libraries(test_transaction_simulator PRIVATE
    tigerwallet_transaction_simulator PkgConfig::SECP256K1)
add_test(NAME transaction_simulator COMMAND test_transaction_simulator)
