import logging
import os

from .test_azure_provider_create import verify_cloud_providers

log = logging.getLogger(__name__)


def test_azure_provider_delete(qbert, cp_uuid):
    if os.getenv('AZURE_CLUSTER_DONT_DELETE'):
        return
    qbert.delete_cloud_provider(cp_uuid)
    verify_cloud_providers(qbert, set(['platform9']))
