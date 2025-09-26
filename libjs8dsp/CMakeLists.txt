cmake_minimum_required(VERSION 3.16)
project(libjs8dsp VERSION 1.0.0 LANGUAGES C CXX)

# Set C++ standard
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Find required packages
find_package(PkgConfig REQUIRED)

# Try to find FFTW3 - if not found, we'll build without it for now
pkg_check_modules(FFTW3 fftw3)
if(FFTW3_FOUND)
    message(STATUS "Found FFTW3: ${FFTW3_LIBRARIES}")
    add_definitions(-DHAVE_FFTW3)
else()
    message(WARNING "FFTW3 not found - building stub version")
endif()

# Find Boost (if available system-wide)
find_package(Boost QUIET)
if(Boost_FOUND)
    message(STATUS "Found system Boost: ${Boost_VERSION}")
else()
    message(STATUS "Using JS8Call's boost headers (header-only)")
endif()

# Include directories
include_directories(
    ${CMAKE_CURRENT_SOURCE_DIR}
    ${CMAKE_CURRENT_SOURCE_DIR}/vendor
)

# Add FFTW3 if found
if(FFTW3_FOUND)
    include_directories(${FFTW3_INCLUDE_DIRS})
    link_directories(${FFTW3_LIBRARY_DIRS})
endif()

# Source files for the C API
set(SOURCES
    js8dsp.cpp
)

# Create static library
add_library(js8dsp STATIC ${SOURCES})

# Set target properties
set_target_properties(js8dsp PROPERTIES
    VERSION ${PROJECT_VERSION}
    SOVERSION 1
    PUBLIC_HEADER "js8dsp.h"
)

# Link libraries
if(FFTW3_FOUND)
    target_link_libraries(js8dsp ${FFTW3_LIBRARIES})
endif()

# Add math library on Unix
if(UNIX)
    target_link_libraries(js8dsp m)
endif()

# Installation
install(TARGETS js8dsp
    LIBRARY DESTINATION lib
    ARCHIVE DESTINATION lib
    PUBLIC_HEADER DESTINATION include
)

# Create a simple test program
add_executable(js8dsp_test test_js8dsp.cpp)
target_link_libraries(js8dsp_test js8dsp)

# Enable testing
enable_testing()
add_test(NAME js8dsp_basic_test COMMAND js8dsp_test)