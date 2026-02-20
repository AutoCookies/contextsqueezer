cmake_minimum_required(VERSION 3.16)
project(contextsqueeze_native LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

if(MSVC)
  add_compile_options(/W4)
else()
  add_compile_options(-Wall -Wextra -Wpedantic)
endif()

add_library(contextsqueeze SHARED
  src/contextsqueeze.cpp
  src/metrics.cpp
  ../internal/core/signature_registry.cpp
)

target_include_directories(contextsqueeze
  PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}/include
    ${CMAKE_CURRENT_SOURCE_DIR}/../internal
)

set_target_properties(contextsqueeze PROPERTIES
  LIBRARY_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/lib
  RUNTIME_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/bin
)

add_executable(contextsqueeze_tests
  test/test_contextsqueeze.cpp
)

target_include_directories(contextsqueeze_tests
  PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/include
    ${CMAKE_CURRENT_SOURCE_DIR}/../internal
)

target_link_libraries(contextsqueeze_tests
  PRIVATE
    contextsqueeze
)

set_target_properties(contextsqueeze_tests PROPERTIES
  RUNTIME_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/bin
)
