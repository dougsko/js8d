cmake_minimum_required(VERSION 3.14)

# Simple test executable
add_executable(js8dsp_test test_basic.cpp)
target_link_libraries(js8dsp_test js8dsp)
target_include_directories(js8dsp_test PRIVATE ../include)