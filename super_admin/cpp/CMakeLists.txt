cmake_minimum_required(VERSION 3.15)
project(TigerSuperAdminCppCore VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

find_package(Threads REQUIRED)

# Header-only library (super_admin_domains.hpp + processor.hpp): real POSIX-
# socket HTTP proxy handlers to the super_admin/go backend (127.0.0.1:8082).
add_library(tiger_super_admin_headers INTERFACE)
target_include_directories(tiger_super_admin_headers INTERFACE ${CMAKE_CURRENT_SOURCE_DIR}/include)
target_link_libraries(tiger_super_admin_headers INTERFACE Threads::Threads)

# Real CLI driver: forwards every command to super_admin/go (no fabricated
# data; a dead upstream surfaces as a transport error + non-zero exit).
add_executable(tiger_super_admin_cpp src/main.cpp)
target_link_libraries(tiger_super_admin_cpp PRIVATE tiger_super_admin_headers)

install(TARGETS tiger_super_admin_cpp RUNTIME DESTINATION bin)
install(DIRECTORY include/ DESTINATION include/tiger_super_admin FILES_MATCHING PATTERN "*.hpp")

