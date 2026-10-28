cmake_minimum_required(VERSION 3.16)
project(tigerwallet_account_abstraction CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

find_package(OpenSSL REQUIRED)
find_package(Threads REQUIRED)

# The account abstraction layer builds on the shared curve arithmetic, the
# Keccak-256 implementation and the 256-bit integer type.
add_library(account_abstraction INTERFACE)
target_include_directories(account_abstraction INTERFACE
    ${CMAKE_CURRENT_SOURCE_DIR}
    ${CMAKE_CURRENT_SOURCE_DIR}/../common)
target_link_libraries(account_abstraction INTERFACE OpenSSL::Crypto Threads::Threads)

enable_testing()

add_executable(test_account_abstraction tests/test_account_abstraction.cpp)
target_link_libraries(test_account_abstraction PRIVATE account_abstraction)
add_test(NAME account_abstraction COMMAND test_account_abstraction)
