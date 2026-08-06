cmake_minimum_required(VERSION 3.15)
project(TigerAdminTxnProcessor CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -O3 -march=native -flto")

# Find threads
find_package(Threads REQUIRED)

# Create the transaction processor library
add_library(txn_processor STATIC
    src/txn_processor.cpp
)

target_include_directories(txn_processor PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}/include
)

target_link_libraries(txn_processor
    Threads::Threads
)

# Build the test executable
add_executable(txn_processor_test
    src/txn_processor.cpp
)

target_include_directories(txn_processor_test PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}/include
)

target_link_libraries(txn_processor_test
    Threads::Threads
)

# Installation
install(TARGETS txn_processor
    ARCHIVE DESTINATION lib
    LIBRARY DESTINATION lib
)

install(DIRECTORY include/
    DESTINATION include
)
