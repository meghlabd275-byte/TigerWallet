cmake_minimum_required(VERSION 3.16)
project(tigerwallet_transaction_processor CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

find_package(OpenSSL REQUIRED)
find_package(Threads REQUIRED)

# The header-only processor, which main.cpp and the tests are written against.
# It carries the shared crypto primitives it hashes and validates with.
add_library(transaction_processor INTERFACE)
target_include_directories(transaction_processor INTERFACE
    ${CMAKE_CURRENT_SOURCE_DIR}
    ${CMAKE_CURRENT_SOURCE_DIR}/../common)
target_link_libraries(transaction_processor INTERFACE OpenSSL::Crypto Threads::Threads)

add_executable(transaction_processor_demo main.cpp)
target_link_libraries(transaction_processor_demo PRIVATE transaction_processor)

enable_testing()

add_executable(test_transaction_processor tests/test_transaction_processor.cpp)
target_link_libraries(test_transaction_processor PRIVATE transaction_processor)
add_test(NAME transaction_processor COMMAND test_transaction_processor)

