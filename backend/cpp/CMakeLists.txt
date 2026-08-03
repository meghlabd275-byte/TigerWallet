# C++ High-Performance Order Matching Engine
# TigerWallet Backend

## CMakeLists.txt
cmake_minimum_required(VERSION 3.15)
project(TigerWalletEngine)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

add_executable(order_matching_engine src/main.cpp src/order_book.cpp src/matching_engine.cpp)

target_include_directories(order_matching_engine PUBLIC include)

# Performance optimizations
target_compile_options(order_matching_engine PRIVATE
    -O3
    -march=native
    -mtune=native
    -flto
)

# Link pthread
find_package(Threads REQUIRED)
target_link_libraries(order_matching_engine Threads::Threads)
