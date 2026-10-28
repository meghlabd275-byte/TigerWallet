cmake_minimum_required(VERSION 3.16)
project(tigerwallet_transaction_simulator CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

add_library(transaction_simulator STATIC transaction_simulator.cpp)
target_include_directories(transaction_simulator PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}
    ${CMAKE_CURRENT_SOURCE_DIR}/../common)

find_package(Threads REQUIRED)
target_link_libraries(transaction_simulator PUBLIC Threads::Threads)

enable_testing()

add_executable(test_transaction_simulator tests/test_transaction_simulator.cpp)
target_link_libraries(test_transaction_simulator PRIVATE transaction_simulator)
add_test(NAME transaction_simulator COMMAND test_transaction_simulator)
