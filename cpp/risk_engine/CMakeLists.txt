cmake_minimum_required(VERSION 3.16)
project(tigerwallet_risk_engine CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

add_library(risk_engine STATIC src/risk_engine.cpp)
target_include_directories(risk_engine PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}/include
    ${CMAKE_CURRENT_SOURCE_DIR}/src)
find_package(Threads REQUIRED)
target_link_libraries(risk_engine PUBLIC Threads::Threads)

enable_testing()

add_executable(test_risk_engine tests/test_risk_engine.cpp)
target_link_libraries(test_risk_engine PRIVATE risk_engine)
add_test(NAME risk_engine COMMAND test_risk_engine)
