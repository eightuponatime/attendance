#!/bin/bash

adb devices
adb reverse tcp:5173 tcp:5173
adb reverse tcp:8080 tcp:8080
adb reverse --list
