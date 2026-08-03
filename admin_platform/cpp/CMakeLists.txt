cmake_minimum_required(VERSION 3.20)
project(TigerAdminCpp)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Find required packages
find_package(PkgConfig REQUIRED)
find_package(Threads REQUIRED)
find_package(OpenSSL REQUIRED)

# PostgreSQL
pkg_check_modules(LIBPQ REQUIRED libpq)

# Redis (hiredis)
pkg_check_modules(HIREDIS REQUIRED hiredis)

# JSON library
find_package(nlohmann_json 3.10.5 QUIET)
if(NOT nlohmann_json_FOUND)
    include(FetchContent)
    FetchContent_Declare(
        json
        GIT_REPOSITORY https://github.com/nlohmann/json.git
        GIT_TAG v3.10.5
    )
    FetchContent_MakeAvailable(json)
endif()

# JWT library
find_package(jwt-cpp QUIET)
if(NOT jwt-cpp_FOUND)
    include(FetchContent)
    FetchContent_Declare(
        jwt-cpp
        GIT_REPOSITORY https://github.com/Thalhammer/jwt-cpp.git
        GIT_TAG v5.1.0
    )
    FetchContent_MakeAvailable(jwt-cpp)
endif()

# Include directories
include_directories(${CMAKE_CURRENT_SOURCE_DIR}/include)
include_directories(${LIBPQ_INCLUDE_DIRS})
include_directories(${HIREDIS_INCLUDE_DIRS})

# Source files
set(SOURCES
    src/main.cpp
    src/server.cpp
    src/database.cpp
    src/redis_client.cpp
    src/session_manager.cpp
    src/admin_service.cpp
    src/user_service.cpp
    src/kyc_service.cpp
    src/token_service.cpp
    src/pair_service.cpp
    src/fee_service.cpp
    src/chain_service.cpp
    src/wallet_service.cpp
    src/whitelabel_service.cpp
    src/analytics_service.cpp
    src/auth_handler.cpp
    src/admin_handler.cpp
    src/user_handler.cpp
    src/kyc_handler.cpp
    src/token_handler.cpp
    src/pair_handler.cpp
    src/fee_handler.cpp
    src/chain_handler.cpp
    src/wallet_handler.cpp
    src/whitelabel_handler.cpp
    src/analytics_handler.cpp
    src/middleware/auth_middleware.cpp
    src/middleware/rate_limiter.cpp
    src/middleware/cors.cpp
    src/middleware/logging.cpp
    src/models/admin.cpp
    src/models/user.cpp
    src/models/kyc.cpp
    src/models/token.cpp
    src/models/pair.cpp
    src/models/transaction.cpp
    src/models/whitelabel.cpp
    src/utils/crypto.cpp
    src/utils/validation.cpp
    src/utils/json_helper.cpp
    src/config/config.cpp
)

# Create executable
add_executable(tiger_admin_cpp ${SOURCES})

# Link libraries
target_link_libraries(tiger_admin_cpp
    Threads::Threads
    OpenSSL::SSL
    OpenSSL::Crypto
    ${LIBPQ_LIBRARIES}
    ${HIREDIS_LIBRARIES}
    pthread
    dl
)

# Copy config file
configure_file(
    ${CMAKE_CURRENT_SOURCE_DIR}/config/admin_config.json
    ${CMAKE_CURRENT_BINARY_DIR}/admin_config.json
    COPYONLY
)

# Installation
install(TARGETS tiger_admin_cpp DESTINATION bin)
install(FILES config/admin_config.json DESTINATION etc/tigeradmin)
