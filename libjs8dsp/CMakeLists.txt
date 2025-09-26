cmake_minimum_required(VERSION 3.14)
project(libjs8dsp VERSION 1.0.0 LANGUAGES CXX)

# Set C++ standard
set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Find dependencies (optional for now)
find_package(Boost QUIET COMPONENTS system)

# Try to find Eigen from js8call vendor directory first
set(EIGEN_INCLUDE_DIR "/Users/doug/bin/js8call/vendor/Eigen")
if(NOT EXISTS ${EIGEN_INCLUDE_DIR})
    # Try system Eigen
    find_path(EIGEN_INCLUDE_DIR Eigen/Dense PATHS /usr/local/include /opt/local/include)
    if(NOT EIGEN_INCLUDE_DIR)
        message(WARNING "Eigen not found - some features may be disabled")
        add_definitions(-DJS8DSP_NO_EIGEN)
    endif()
endif()

# Include directories
include_directories(${CMAKE_CURRENT_SOURCE_DIR}/include)

if(EIGEN_INCLUDE_DIR)
    include_directories(${EIGEN_INCLUDE_DIR})
    message(STATUS "Using Eigen from: ${EIGEN_INCLUDE_DIR}")
endif()

if(Boost_FOUND)
    include_directories(${Boost_INCLUDE_DIRS})
    message(STATUS "Using Boost from: ${Boost_INCLUDE_DIRS}")
else()
    message(WARNING "Boost not found - some features may be disabled")
    add_definitions(-DJS8DSP_NO_BOOST)
endif()

# Source files
set(SOURCES
    src/js8_decoder.cpp
    src/varicode.cpp
    src/js8dsp_api.cpp
    src/bp_decoder.cpp
    src/baseline_computation.cpp
)

# Header files for installation
set(HEADERS
    include/js8dsp.h
    include/js8_decoder.h
    include/js8_constants.h
    include/varicode.h
    include/bp_decoder.h
    include/baseline_computation.h
)

# FFT is now handled by Go, no C++ FFT needed

# Create static library
add_library(js8dsp STATIC ${SOURCES})

# Link libraries
if(Boost_FOUND)
    target_link_libraries(js8dsp ${Boost_LIBRARIES})
endif()

# Compiler flags
target_compile_options(js8dsp PRIVATE -Wall -Wextra -O3)

# Install headers
install(DIRECTORY include/ DESTINATION include)
install(TARGETS js8dsp DESTINATION lib)

# Tests (optional)
option(BUILD_TESTS "Build tests" ON)
if(BUILD_TESTS)
    add_subdirectory(tests)
endif()