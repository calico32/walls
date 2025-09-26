function walls
    command go run . -v -c config-test.kdl $argv
    return $status
end

export STARSHIP_ENVIRONMENT=walls
