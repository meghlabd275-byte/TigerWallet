cmake_minimum_required(VERSION 3.16)
project(tigerwallet_gas_optimizer CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Header-only.
add_library(gas_optimizer INTERFACE)
target_include_directories(gas_optimizer INTERFACE ${CMAKE_CURRENT_SOURCE_DIR})

enable_testing()

add_executable(test_gas_optimizer tests/test_gas_optimizer.cpp)
target_link_libraries(test_gas_optimizer PRIVATE gas_optimizer)
find_package(Threads REQUIRED)
target_link_libraries(test_gas_optimizer PRIVATE Threads::Threads)
add_test(NAME gas_optimizer COMMAND test_gas_optimizer)
