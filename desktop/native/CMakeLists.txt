cmake_minimum_required(VERSION 3.21)
project(rednote_player_native LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

set(NODE_ROOT ${CMAKE_CURRENT_SOURCE_DIR}/third_party/electron)
set(FFMPEG_ROOT ${CMAKE_CURRENT_SOURCE_DIR}/third_party/ffmpeg)
set(NAPI_INCLUDE ${CMAKE_CURRENT_SOURCE_DIR}/../node_modules/node-addon-api)

add_library(rednote_player SHARED
    src/binding.cpp
    src/PlayerCore.cpp
    src/PlayerCore.h
    src/AudioRenderer.cpp
    src/AudioRenderer.h
)

target_include_directories(rednote_player PRIVATE
    src
    ${NODE_ROOT}/include/node
    ${NAPI_INCLUDE}
    ${CMAKE_CURRENT_SOURCE_DIR}/third_party
    ${FFMPEG_ROOT}/include
)

target_link_directories(rednote_player PRIVATE
    ${FFMPEG_ROOT}/lib
    ${NODE_ROOT}
)

target_link_libraries(rednote_player PRIVATE
    avformat avcodec avutil swscale swresample
    ${NODE_ROOT}/electron.lib
    ole32 uuid winmm
)

target_compile_definitions(rednote_player PRIVATE
    NAPI_CPP_EXCEPTIONS
    NOMINMAX
)

set_target_properties(rednote_player PROPERTIES
    PREFIX ""
    SUFFIX ".node"
    OUTPUT_NAME "rednote_player"
)
