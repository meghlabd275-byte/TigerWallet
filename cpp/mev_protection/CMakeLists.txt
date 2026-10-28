cmake_minimum_required(VERSION 3.16)
project(tigerwallet_mev_protection CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Header-only. The interface target carries the include paths for this module
# and for the shared crypto primitives it hashes with.
add_library(mev_protection INTERFACE)
target_include_directories(mev_protection INTERFACE
    ${CMAKE_CURRENT_SOURCE_DIR}
    ${CMAKE_CURRENT_SOURCE_DIR}/../common)

enable_testing()

add_executable(test_mev_protector tests/test_mev_protector.cpp)
target_link_libraries(test_mev_protector PRIVATE mev_protection)
add_test(NAME mev_protector COMMAND test_mev_protector)
