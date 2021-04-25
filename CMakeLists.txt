cmake_minimum_required(VERSION 3.19)

set(CMAKE_C_COMPILER /usr/bin/cc)
project(WTMP_client C)

set(CMAKE_C_STANDARD 99)

add_executable(WTMP_client main.c)