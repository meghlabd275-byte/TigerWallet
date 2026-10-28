cmake_minimum_required(VERSION 3.16)
project(tigerwallet_privacy CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

find_package(PkgConfig REQUIRED)
find_package(Threads REQUIRED)
# The module hashes with Crypto++ and signs proof transcripts with libsecp256k1.
# Crypto++ installs no pkg-config file, so the library is located directly.
find_path(CRYPTOPP_INCLUDE_DIR cryptopp/keccak.h REQUIRED)
find_library(CRYPTOPP_LIBRARY NAMES cryptopp REQUIRED)
pkg_check_modules(SECP256K1 REQUIRED IMPORTED_TARGET libsecp256k1)

add_library(tigerwallet_privacy INTERFACE)
target_include_directories(tigerwallet_privacy INTERFACE
    ${CMAKE_CURRENT_SOURCE_DIR}
    ${CMAKE_CURRENT_SOURCE_DIR}/../common)
target_include_directories(tigerwallet_privacy INTERFACE ${CRYPTOPP_INCLUDE_DIR})
target_link_libraries(tigerwallet_privacy INTERFACE
    ${CRYPTOPP_LIBRARY} PkgConfig::SECP256K1 Threads::Threads)

enable_testing()

add_executable(test_privacy_features tests/test_privacy_features.cpp)
target_link_libraries(test_privacy_features PRIVATE tigerwallet_privacy)
target_compile_options(test_privacy_features PRIVATE -Wall -Wextra)
add_test(NAME privacy_features COMMAND test_privacy_features)
