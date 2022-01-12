#!/usr/bin/env bash
set -eu

CONTENT_TYPE="content-type: application/json"
TACKBOARD_URL=http://localhost:9080/tackboard/
DEBUG=${DEBUG:-""}
TEST=${TEST:-""}
PASS="true"

function main()
{
    if [ -n "$TEST" ]; then
        tests::run_all
        exit 0
    fi

    [[ $# -ge 3 ]] || usage
    local host="$1"
    local ports=${@:2:$(($# - 2))}
    local uuid="${@: -1}"

    local output=$(report "$host" "$ports")
    if [ -n "$DEBUG" ]; then
        echo "$output"
    else
        echo "$output" \
            | NO_PROXY=localhost curl -s \
                   -X POST \
                   -H "${CONTENT_TYPE}" \
                   -H "uuid: ${uuid}" \
                   ${TACKBOARD_URL} \
                   -d@-
    fi
}

function usage()
{
    echo "Usage: $0 <host> <port1 port2 ...> <uuid>"
    exit 1
}

function report()
{
    local host="$1"
    local ports="$2"

    local output=$(
        first=1
        for port in $ports; do
            if ! is_reachable "$host" "$port"; then
                reachable='false'
            else
                reachable='true'
            fi
            [[ $first -eq 1 ]] && first=0 || echo -n ","
            echo -n "{ \"port\": ${port}, \"reachable\": ${reachable} }"
        done
    )
    echo -n "{ \"host\": \"${host}\", \"reachability\": [ ${output} ] }"
}

function is_reachable()
{
    local host=$1
    local port=$2
    local timeout=1s
    # from http://stackoverflow.com/a/19866239
    &>/dev/null timeout $timeout bash -c "echo "" > /dev/tcp/${host}/${port}"
    return $?
}

function tests::reachable()
{
    local host=8.8.8.8
    local ports=(53)
    report "$host" "$ports"
}

function tests::host_unreachable()
{
    local host=169.169.169.169
    local ports=(1)
    report "$host" "$ports"
}

function tests::port_unreachable()
{
    local host=127.0.0.1
    local ports=(9876)
    report "$host" "$ports"
}

function tests::run_test()
{
    local func=$1
    local expected=$2

    echo -en "-----\nRunning test '$func': "
    local output=$($func)
    if [ "$output" != "$expected" ]; then
        echo "Test failed!"
        PASS="false"
    else
        echo "Test passed."
    fi
    echo -e "Expected\n${expected}\nOutput\n${output}"
}

function tests::run_all()
{
    set +e # Run all tests, even if some fail
    tests::run_test \
        tests::reachable \
        "{ \"host\": \"8.8.8.8\", \"reachability\": [ { \"port\": 53, \"reachable\": true } ] }"
    tests::run_test \
        tests::host_unreachable \
        "{ \"host\": \"169.169.169.169\", \"reachability\": [ { \"port\": 1, \"reachable\": false } ] }"
    tests::run_test \
        tests::port_unreachable \
        "{ \"host\": \"127.0.0.1\", \"reachability\": [ { \"port\": 9876, \"reachable\": false } ] }"
    if [ "$PASS" == "false" ]; then
        exit 1
    fi
}

main "$@"
