#!/usr/bin/env bash
pushd "$(dirname $0)"
python setup.py develop --uninstall
popd
