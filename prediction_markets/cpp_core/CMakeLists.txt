cmake_minimum_required(VERSION 3.16)
project(tiger_prediction_matching_engine CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

if(NOT CMAKE_BUILD_TYPE)
    set(CMAKE_BUILD_TYPE Release)
endif()

add_compile_options(-Wall -Wextra)

find_package(Threads REQUIRED)

# The matching engine is header-heavy: the hot paths are inlined so the
# compiler can keep them in registers.
add_library(tiger_prediction_engine STATIC prediction_matching_engine.cpp)
target_include_directories(tiger_prediction_engine PUBLIC ${CMAKE_CURRENT_SOURCE_DIR})
target_link_libraries(tiger_prediction_engine PUBLIC Threads::Threads)

enable_testing()

add_executable(test_matching_engine test_matching_engine.cpp)
target_link_libraries(test_matching_engine PRIVATE tiger_prediction_engine)

add_test(NAME matching_engine COMMAND test_matching_engine)
