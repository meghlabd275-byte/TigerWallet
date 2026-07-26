cmake_minimum_required(VERSION 3.15)
project(TigerWalletMEV VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

# Release optimizations for ultra-low latency
set(CMAKE_CXX_FLAGS_RELEASE "${CMAKE_CXX_FLAGS_RELEASE} -O3 -march=native -mtune=native")
set(CMAKE_CXX_FLAGS_DEBUG "${CMAKE_CXX_FLAGS_DEBUG} -g -O0")

# Find required packages
find_package(Threads REQUIRED)

# Source files
set(SOURCES
    src/mev_blocker.cpp
)

set(HEADERS
    include/mev_blocker.hpp
    include/transaction.hpp
    include/keccak.hpp
)

# Create library
add_library(tiger_mev STATIC ${SOURCES} ${HEADERS})

# Include directories
target_include_directories(tiger_mev PUBLIC 
    ${CMAKE_CURRENT_SOURCE_DIR}/include
)

# Link libraries
target_link_libraries(tiger_mev PRIVATE Threads::Threads)

# Enable LTO for better performance
include(CheckIPOSupported)
check_ipo_supported(SUPPORTS_IPO)
if(SUPPORTS_IPO)
    set_target_properties(tiger_mev PROPERTIES
        INTERPROCEDURAL_OPTIMIZATION_RELEASE TRUE
    )
endif()

# Install
install(TARGETS tiger_mev
    ARCHIVE DESTINATION lib
    LIBRARY DESTINATION lib
)

install(FILES ${HEADERS}
    DESTINATION include
)

# Tests
enable_testing()
add_executable(test_mev test/test_mev.cpp)
target_link_libraries(test_mev tiger_mev Threads::Threads)
add_test(NAME MEVTest COMMAND test_mev)

# Benchmarks
add_executable(benchmark_mev benchmark/benchmark_mev.cpp)
target_link_libraries(benchmark_mev tiger_mev Threads::Threads)
