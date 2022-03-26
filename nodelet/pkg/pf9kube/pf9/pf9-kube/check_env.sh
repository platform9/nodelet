#!/bin/sh
# Purpose: This script is meant to be used by anything attempting to build K8s artifacts as a sanity check.
# Inputs: all environment variables files
# Outputs: none
# Side effects: Exits with error if environment variables arent set correctly.
if test "${AGENT_VERSION}" == "" ; then \
        echo "*** ERROR *** AGENT_VERSION not set"; \
		env; \
        exit 1; \
fi
if test "${BUILD_NUMBER}" == "" ; then \
        echo "*** ERROR *** BUILD_NUMBER not set"; \
		env; \
        exit 1; \
fi
if test "${BUILD_DIR}" == "" ; then \
        echo "*** ERROR *** BUILD_DIR not set"; \
		env; \
        exit 1; \
fi
if test "${QBERT_SRC_DIR}" == "" ; then \
        echo "*** ERROR *** QBERT_SRC_DIR not set"; \
		env; \
        exit 1; \
fi
if test "${KUBERNETES_DIR:0:1}" == "/" ; then \
		echo "KUBERNETES_DIR is correctly formatted ${KUBERNETES_DIR}"; \
else \
		echo "*** ERROR *** KUBERNETES_DIR not absolute ${KUBERNETES_DIR} "; \
		env; \
		exit 1; \
fi