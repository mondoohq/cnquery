#!/usr/bin/env bats
# Tests for .github/scripts/determine-provider-release-base.sh
#
# Each test sets MAIN_GOMOD in the environment so the script runs without
# calling gh api. See the script's header for the input contract.

setup() {
  SCRIPT="$BATS_TEST_DIRNAME/../determine-provider-release-base.sh"
}

@test "matching major routes to main (v-prefixed tag)" {
  MAIN_GOMOD='module go.mondoo.com/mql/v14' run bash "$SCRIPT" v14.5.0
  [ "$status" -eq 0 ]
  [ "$output" = "main" ]
}

@test "matching major routes to main (no v prefix)" {
  MAIN_GOMOD='module go.mondoo.com/mql/v14' run bash "$SCRIPT" 14.5.0
  [ "$status" -eq 0 ]
  [ "$output" = "main" ]
}

@test "older major routes to v{major}" {
  MAIN_GOMOD='module go.mondoo.com/mql/v14' run bash "$SCRIPT" v13.35.1
  [ "$status" -eq 0 ]
  [ "$output" = "v13" ]
}

@test "future major routes to v{major}" {
  MAIN_GOMOD='module go.mondoo.com/mql/v14' run bash "$SCRIPT" v15.0.0
  [ "$status" -eq 0 ]
  [ "$output" = "v15" ]
}

@test "pre-release suffix does not change routing" {
  MAIN_GOMOD='module go.mondoo.com/mql/v14' run bash "$SCRIPT" v14.0.0-rc.1
  [ "$status" -eq 0 ]
  [ "$output" = "main" ]
}

@test "pre-release suffix on older major routes to v{major}" {
  MAIN_GOMOD='module go.mondoo.com/mql/v14' run bash "$SCRIPT" v13.35.1-rc.1
  [ "$status" -eq 0 ]
  [ "$output" = "v13" ]
}

@test "main on v13 today: v13 release routes to main" {
  MAIN_GOMOD='module go.mondoo.com/mql/v13' run bash "$SCRIPT" v13.35.1
  [ "$status" -eq 0 ]
  [ "$output" = "main" ]
}

@test "main on v13 today: v14 pre-release routes to v14" {
  MAIN_GOMOD='module go.mondoo.com/mql/v13' run bash "$SCRIPT" v14.0.0-rc.1
  [ "$status" -eq 0 ]
  [ "$output" = "v14" ]
}

@test "require line (not module line) is ignored" {
  # The go.mod on cnspec/server has a require line for mql. The mql
  # script must anchor on the module line at the top of the file.
  MAIN_GOMOD=$'module go.mondoo.com/mql/v14\n\nrequire go.mondoo.com/mql/v13 v13.0.0 // indirect' \
    run bash "$SCRIPT" v14.5.0
  [ "$status" -eq 0 ]
  [ "$output" = "main" ]
}

@test "missing module line fails loudly" {
  MAIN_GOMOD=$'go 1.24\n\nrequire go.mondoo.com/mql/v14 v14.0.0' \
    run bash "$SCRIPT" v14.5.0
  [ "$status" -ne 0 ]
  [[ "$output" == *"Could not extract"* ]]
}

@test "missing tag argument errors out" {
  MAIN_GOMOD='module go.mondoo.com/mql/v14' run bash "$SCRIPT"
  [ "$status" -ne 0 ]
}
