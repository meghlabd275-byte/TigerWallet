cmake_minimum_required(VERSION 3.16)
project(wl_control_plane CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

add_library(wl_gate SHARED src/wl_gate.cpp)
target_include_directories(wl_gate PUBLIC include)

add_library(wl_gate_static STATIC src/wl_gate.cpp)
target_include_directories(wl_gate_static PUBLIC include)

add_library(wl_auto_approver SHARED src/wl_auto_approver.cpp)
target_include_directories(wl_auto_approver PUBLIC include)

add_library(wl_auto_approver_static STATIC src/wl_auto_approver.cpp)
target_include_directories(wl_auto_approver_static PUBLIC include)

add_executable(test_wl_gate tests/test_wl_gate.cpp)
target_link_libraries(test_wl_gate PRIVATE wl_gate_static)

add_executable(test_wl_auto_approver tests/test_wl_auto_approver.cpp)
target_link_libraries(test_wl_auto_approver PRIVATE wl_auto_approver_static)

enable_testing()
add_test(NAME wl_gate COMMAND test_wl_gate)
add_test(NAME wl_auto_approver COMMAND test_wl_auto_approver)
