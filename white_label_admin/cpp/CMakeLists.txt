cmake_minimum_required(VERSION 3.15)
project(TigerWLAdminDomains CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Header-only library: WL admin domain handlers (governance records only,
# no fund movement). The previous build referenced a non-existent
# src/processor.cpp (wrong tx-processor content); removed.
add_library(wl_admin_domains INTERFACE)
target_include_directories(wl_admin_domains INTERFACE ${CMAKE_CURRENT_SOURCE_DIR}/include)

install(DIRECTORY include/ DESTINATION include)
