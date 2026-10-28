cmake_minimum_required(VERSION 3.16)
project(tigerwallet_common CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Header-only primitives shared by the wallet's C++ modules.
add_library(tigerwallet_common INTERFACE)
target_include_directories(tigerwallet_common INTERFACE ${CMAKE_CURRENT_SOURCE_DIR})

enable_testing()

add_executable(test_keccak256 tests/test_keccak256.cpp)
target_link_libraries(test_keccak256 PRIVATE tigerwallet_common)
add_test(NAME keccak256 COMMAND test_keccak256)

add_executable(test_rlp tests/test_rlp.cpp)
target_link_libraries(test_rlp PRIVATE tigerwallet_common)
add_test(NAME rlp COMMAND test_rlp)

add_executable(test_uint256 tests/test_uint256.cpp)
target_link_libraries(test_uint256 PRIVATE tigerwallet_common)
add_test(NAME uint256 COMMAND test_uint256)
