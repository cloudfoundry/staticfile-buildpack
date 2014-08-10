#!/usr/bin/env bats

@test "git binary is found in PATH" {
  run which gittt
  [ "$status" -eq 0 ]
}
