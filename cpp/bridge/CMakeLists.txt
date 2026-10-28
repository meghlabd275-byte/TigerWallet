cmake_minimum_required(VERSION 3.16)
project(tigerwallet_bridge CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

find_package(OpenSSL REQUIRED)
find_package(Threads REQUIRED)

# The bridge layer builds on the shared crypto and integer helpers.
add_library(tigerwallet_bridge INTERFACE)
target_include_directories(tigerwallet_bridge INTERFACE
    ${CMAKE_CURRENT_SOURCE_DIR}
    ${CMAKE_CURRENT_SOURCE_DIR}/../common)
target_link_libraries(tigerwallet_bridge INTERFACE OpenSSL::Crypto Threads::Threads)

enable_testing()

add_executable(test_bridge tests/test_bridge.cpp)
target_link_libraries(test_bridge PRIVATE tigerwallet_bridge)
add_test(NAME bridge COMMAND test_bridge)
