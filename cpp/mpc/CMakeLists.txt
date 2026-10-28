cmake_minimum_required(VERSION 3.16)
project(tigerwallet_enterprise_mpc CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

find_package(OpenSSL REQUIRED)
find_package(Threads REQUIRED)

# The MPC engine builds on the shared curve arithmetic and address checks.
add_library(enterprise_mpc INTERFACE)
target_include_directories(enterprise_mpc INTERFACE
    ${CMAKE_CURRENT_SOURCE_DIR}
    ${CMAKE_CURRENT_SOURCE_DIR}/../common)
target_link_libraries(enterprise_mpc INTERFACE OpenSSL::Crypto Threads::Threads)

enable_testing()

add_executable(test_enterprise_mpc tests/test_enterprise_mpc.cpp)
target_link_libraries(test_enterprise_mpc PRIVATE enterprise_mpc)
add_test(NAME enterprise_mpc COMMAND test_enterprise_mpc)
