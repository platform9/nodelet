__author__ = 'Platform9'

try:
    from setuptools import setup, find_packages
except ImportError:
    from ez_setup import use_setuptools
    use_setuptools()
    from setuptools import setup, find_packages

setup(
    name='kube_tests',
    version='0.1',
    author='Platform9',
    author_email='support@platform9.net',
    install_requires=[
        'fabric>=1.8',
        'requests',
        'kubernetes'
    ],
    packages=find_packages()
)
