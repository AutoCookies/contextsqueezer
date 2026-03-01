cmake_minimum_required(VERSION 3.16)
project(cpp_integration LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Find ContextSqueezer using the exported targets
# In a real scenario, this would be find_package(ContextSqueeze REQUIRED)
# But here we point it to the build_static/install directory
find_package(ContextSqueeze REQUIRED PATHS ${CMAKE_CURRENT_SOURCE_DIR}/../../build_static/install)

add_executable(csq_example main.cpp)
target_link_libraries(csq_example PRIVATE ContextSqueeze::contextsqueeze)
