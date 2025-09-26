cmake_minimum_required(VERSION 3.14)
project(libjs8dsp VERSION 1.0.0 LANGUAGES CXX)

# Set C++ standard
set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Find dependencies (optional for now)
find_package(PkgConfig QUIET)
find_package(Boost QUIET COMPONENTS system)

# Try to find FFTW3 using pkg-config first, then fallback
if(PkgConfig_FOUND)
    pkg_check_modules(FFTW3 fftw3)
endif()

if(NOT FFTW3_FOUND)
    find_library(FFTW3_LIBRARIES fftw3 PATHS /opt/local/lib /usr/local/lib)
    find_path(FFTW3_INCLUDE_DIRS fftw3.h PATHS /opt/local/include /usr/local/include)
    if(FFTW3_LIBRARIES AND FFTW3_INCLUDE_DIRS)
        set(FFTW3_FOUND TRUE)
    endif()
endif()

if(NOT FFTW3_FOUND)
    message(WARNING "FFTW3 not found - building with mock FFT implementation")
    add_definitions(-DJS8DSP_MOCK_FFT)
endif()

# Try to find Eigen from js8call vendor directory first
set(EIGEN_INCLUDE_DIR "${CMAKE_CURRENT_SOURCE_DIR}/../js8call/vendor/Eigen")
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

if(FFTW3_FOUND)
    include_directories(${FFTW3_INCLUDE_DIRS})
    message(STATUS "Using FFTW3 from: ${FFTW3_INCLUDE_DIRS}")
endif()

# Source files
set(SOURCES
    src/js8_decoder.cpp
    src/varicode.cpp
    src/js8dsp_api.cpp
)

# Add mock FFT if FFTW3 not found
if(NOT FFTW3_FOUND)
    list(APPEND SOURCES src/mock_fft.cpp)
endif()

# Create static library
add_library(js8dsp STATIC ${SOURCES})

# Link libraries
if(Boost_FOUND)
    target_link_libraries(js8dsp ${Boost_LIBRARIES})
endif()
if(FFTW3_FOUND)
    target_link_libraries(js8dsp ${FFTW3_LIBRARIES})
endif()

# Compiler flags
target_compile_options(js8dsp PRIVATE -Wall -Wextra -O3)
if(FFTW3_FOUND AND FFTW3_CFLAGS)
    target_compile_options(js8dsp PRIVATE ${FFTW3_CFLAGS})
endif()

# Install headers
install(DIRECTORY include/ DESTINATION include)
install(TARGETS js8dsp DESTINATION lib)

# Tests (optional)
option(BUILD_TESTS "Build tests" ON)
if(BUILD_TESTS)
    add_subdirectory(tests)
endif()