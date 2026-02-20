cmake_minimum_required(VERSION 3.16)
project(contextsqueeze_native LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

add_library(contextsqueeze SHARED
  src/contextsqueeze.cpp
)

target_include_directories(contextsqueeze
  PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}/include
)

if (MSVC)
  target_compile_options(contextsqueeze PRIVATE /W4)
else()
  target_compile_options(contextsqueeze PRIVATE -Wall -Wextra -Wpedantic)
endif()

add_executable(contextsqueeze_tests
  test/test_contextsqueeze.cpp
)

target_include_directories(contextsqueeze_tests
  PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/include
)

target_link_libraries(contextsqueeze_tests PRIVATE contextsqueeze)

if (MSVC)
  target_compile_options(contextsqueeze_tests PRIVATE /W4)
else()
  target_compile_options(contextsqueeze_tests PRIVATE -Wall -Wextra -Wpedantic)
endif()

set_target_properties(contextsqueeze PROPERTIES
  LIBRARY_OUTPUT_DIRECTORY ${CMAKE_SOURCE_DIR}/../build/native/lib
)

set_target_properties(contextsqueeze_tests PROPERTIES
  RUNTIME_OUTPUT_DIRECTORY ${CMAKE_SOURCE_DIR}/../build/native/bin
)
