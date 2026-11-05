cmake_minimum_required(VERSION 3.15)
project(TigerAdminCppCore VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

if(CMAKE_BUILD_TYPE STREQUAL "Release")
    set(CMAKE_CXX_FLAGS_RELEASE "${CMAKE_CXX_FLAGS_RELEASE} -O2")
endif()

find_package(Threads REQUIRED)
find_package(OpenSSL REQUIRED)

# All sources live under include/ (headers + their .cpp companions). The
# executable entry point (int main) is in admin_core.cpp.
file(GLOB ADMIN_SOURCES CONFIGURE_DEPENDS ${CMAKE_CURRENT_SOURCE_DIR}/include/*.cpp)
list(REMOVE_DUPLICATES ADMIN_SOURCES)

add_executable(tiger_admin_cpp ${ADMIN_SOURCES})

target_include_directories(tiger_admin_cpp PRIVATE ${CMAKE_CURRENT_SOURCE_DIR}/include)

target_link_libraries(tiger_admin_cpp
    Threads::Threads
    OpenSSL::Crypto
)

install(TARGETS tiger_admin_cpp RUNTIME DESTINATION bin)
install(DIRECTORY include/ DESTINATION include/tiger_admin FILES_MATCHING PATTERN "*.hpp")


# Tests for the security primitives. Built only when testing is enabled so the
# production target stays free of test code.
enable_testing()

add_executable(tiger_admin_security_tests
    tests/test_admin_security.cpp
    include/admin_security.cpp
    include/admin_logger.cpp
)
target_include_directories(tiger_admin_security_tests PRIVATE ${CMAKE_CURRENT_SOURCE_DIR}/include)
target_link_libraries(tiger_admin_security_tests
    Threads::Threads
    OpenSSL::Crypto
)
add_test(NAME admin_security COMMAND tiger_admin_security_tests)
