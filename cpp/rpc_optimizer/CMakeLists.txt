cmake_minimum_required(VERSION 3.16)
project(tigerwallet_rpc_optimizer CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

find_package(CURL REQUIRED)
find_package(Threads REQUIRED)

# Header-only. The interface target carries the include path and the transport
# dependency so a consumer does not have to rediscover libcurl.
add_library(rpc_optimizer INTERFACE)
target_include_directories(rpc_optimizer INTERFACE ${CMAKE_CURRENT_SOURCE_DIR})
target_link_libraries(rpc_optimizer INTERFACE CURL::libcurl Threads::Threads)

enable_testing()

add_executable(test_rpc_optimizer tests/test_rpc_optimizer.cpp)
target_link_libraries(test_rpc_optimizer PRIVATE rpc_optimizer)
add_test(NAME rpc_optimizer COMMAND test_rpc_optimizer)
