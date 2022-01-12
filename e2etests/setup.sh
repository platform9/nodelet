#!/usr/bin/env bash
pushd "$(dirname $0)"
pip install dictdiffer
pip install --upgrade botocore
pip install --upgrade requests
python setup.py develop
popd
