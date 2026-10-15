cmake_minimum_required(VERSION 3.16)
project(Optics C)

set(CMAKE_C_STANDARD 11)
# Static CRT: the exe runs on a machine with no VC++ redistributable (a fresh
# test VM, a user who just downloaded it) instead of exiting 0xC0000135.
set(CMAKE_MSVC_RUNTIME_LIBRARY "MultiThreaded$<$<CONFIG:Debug>:Debug>")
set(CMAKE_RUNTIME_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/bin)
foreach(cfg Debug Release RelWithDebInfo MinSizeRel)
    string(TOUPPER ${cfg} CFG)
    set(CMAKE_RUNTIME_OUTPUT_DIRECTORY_${CFG} ${CMAKE_BINARY_DIR}/bin)
endforeach()

# Third-party code, built with its own warnings left alone. See NOTICE.
add_library(thirdparty STATIC third_party/ufbx/ufbx.c)
target_include_directories(thirdparty PUBLIC third_party/ufbx third_party/stb)

# Everything both executables share: scanning, zip, model loading, drawing.
add_library(core STATIC src/scan.c src/zip.c src/inflate.c src/model.c src/render.c)
target_include_directories(core PUBLIC src)
target_link_libraries(core PUBLIC thirdparty opengl32)

# Resource 1: Explorer and the taskbar take the lowest-numbered icon.
add_executable(Optics WIN32 src/main.c src/optics.manifest assets/optics.rc)
target_link_libraries(Optics PRIVATE core comctl32 shlwapi uxtheme ole32 shell32 advapi32)

add_executable(optics-check src/check.c)
target_link_libraries(optics-check PRIVATE core)

foreach(t core Optics optics-check)
    target_compile_definitions(${t} PRIVATE UNICODE _UNICODE _CRT_SECURE_NO_WARNINGS WIN32_LEAN_AND_MEAN)
    if(MSVC)
        target_compile_options(${t} PRIVATE /W4 /WX)
        if(CMAKE_C_COMPILER_ID STREQUAL "Clang")
            # clang-cl (netlab's builder) flags `PIXELFORMATDESCRIPTOR pfd = { a, b }`;
            # zero-filling the rest of a Win32 struct is the intent, not a slip.
            target_compile_options(${t} PRIVATE -Wno-missing-field-initializers -Wno-missing-braces)
        endif()
    else()
        target_compile_options(${t} PRIVATE -Wall -Wextra -Werror)
    endif()
endforeach()
if(MSVC)
    # wmain in the console tool
    target_link_options(optics-check PRIVATE /ENTRY:wmainCRTStartup)
endif()
