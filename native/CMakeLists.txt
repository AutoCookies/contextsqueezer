cmake_minimum_required(VERSION 3.16)
project(contextsqueeze_native VERSION 1.0.0 LANGUAGES CXX)

include(GNUInstallDirs)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

option(BUILD_SHARED_LIBS "Build shared libraries" ON)
option(CSQ_WITH_SIMD "Enable SIMD optimizations" ON)

if(MSVC)
  add_compile_options(/W4)
else()
  add_compile_options(-Wall -Wextra -Wpedantic)
  if(CSQ_WITH_SIMD)
    if(CMAKE_SYSTEM_PROCESSOR MATCHES "x86_64|amd64")
      add_compile_options(-msse4.2 -mavx2 -mfma)
    elseif(CMAKE_SYSTEM_PROCESSOR MATCHES "arm|aarch64")
      add_compile_options(-march=armv8-a+simd)
    endif()
  endif()
endif()

add_library(contextsqueeze
  src/contextsqueeze.cpp
  src/metrics.cpp
  ../internal/core/signature_registry.cpp
)

target_include_directories(contextsqueeze
  PUBLIC
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
    $<INSTALL_INTERFACE:${CMAKE_INSTALL_INCLUDEDIR}>
  PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/../internal
)

set_target_properties(contextsqueeze PROPERTIES
  LIBRARY_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/lib
  ARCHIVE_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/lib
  RUNTIME_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/bin
  VERSION ${PROJECT_VERSION}
  SOVERSION 1
)

# Installation
install(TARGETS contextsqueeze
  EXPORT ContextSqueezeTargets
  LIBRARY DESTINATION ${CMAKE_INSTALL_LIBDIR}
  ARCHIVE DESTINATION ${CMAKE_INSTALL_LIBDIR}
  RUNTIME DESTINATION ${CMAKE_INSTALL_BINDIR}
  INCLUDES DESTINATION ${CMAKE_INSTALL_INCLUDEDIR}
)

install(FILES include/contextsqueeze.h include/metrics.h
  DESTINATION ${CMAKE_INSTALL_INCLUDEDIR}
)

include(CMakePackageConfigHelpers)

write_basic_package_version_file(
  "${CMAKE_CURRENT_BINARY_DIR}/ContextSqueezeConfigVersion.cmake"
  VERSION ${PROJECT_VERSION}
  COMPATIBILITY AnyNewerVersion
)

configure_package_config_file(
  "${CMAKE_CURRENT_SOURCE_DIR}/cmake/ContextSqueezeConfig.cmake.in"
  "${CMAKE_CURRENT_BINARY_DIR}/ContextSqueezeConfig.cmake"
  INSTALL_DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/ContextSqueeze
)

install(FILES
  "${CMAKE_CURRENT_BINARY_DIR}/ContextSqueezeConfig.cmake"
  "${CMAKE_CURRENT_BINARY_DIR}/ContextSqueezeConfigVersion.cmake"
  DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/ContextSqueeze
)

install(EXPORT ContextSqueezeTargets
  FILE ContextSqueezeTargets.cmake
  NAMESPACE ContextSqueeze::
  DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/ContextSqueeze
)

# Tests
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
