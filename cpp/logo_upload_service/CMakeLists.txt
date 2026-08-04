cmake_minimum_required(VERSION 3.15)
project(logo_upload_service)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Find required packages
find_package(CURL REQUIRED)
find_package(OpenSSL REQUIRED)
find_package(Threads REQUIRED)

# Include directories
include_directories(${CMAKE_SOURCE_DIR})

# Source files
set(SOURCES
    logo_upload_service.cpp
)

# Create executable
add_executable(logo_upload_service ${SOURCES})

# Link libraries
target_link_libraries(logo_upload_service
    ${CURL_LIBRARIES}
    ${OPENSSL_LIBRARIES}
    Threads::Threads
)

# Installation
install(TARGETS logo_upload_service DESTINATION bin)
install(DIRECTORY DESTINATION var/lib/tigerwallet/logos)

# Compiler options
target_compile_options(logo_upload_service PRIVATE
    -Wall -Wextra -O3
    $<$<CONFIG:Release>:-DNDEBUG>
)
