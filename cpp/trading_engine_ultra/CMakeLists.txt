cmake_minimum_required(VERSION 3.16)
project(tigerwallet_trading_engine_ultra CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# The engine is a header-only library.
add_library(trading_engine_ultra INTERFACE)
target_include_directories(trading_engine_ultra INTERFACE ${CMAKE_CURRENT_SOURCE_DIR})

enable_testing()

add_executable(test_trading_engine tests/test_trading_engine.cpp)
target_link_libraries(test_trading_engine PRIVATE trading_engine_ultra)
find_package(Threads REQUIRED)
target_link_libraries(test_trading_engine PRIVATE Threads::Threads)

add_test(NAME trading_engine COMMAND test_trading_engine)
