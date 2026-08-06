cmake_minimum_required(VERSION 3.15)
project(TigerAdminProcessor CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -O3 -march=native -flto")

find_package(Threads REQUIRED)

add_library(processor STATIC src/processor.cpp)

target_include_directories(processor PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}/include)

target_link_libraries(processor Threads::Threads)

install(TARGETS processor ARCHIVE DESTINATION lib LIBRARY DESTINATION lib)
install(DIRECTORY include/ DESTINATION include)
