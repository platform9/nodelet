import logging
import os

from proboscis.asserts import assert_equal, assert_true

log = logging.getLogger(__name__)


def test_azure_provider_create(qbert, azure_testbed_profile):
    cp_uuid = _test_cloud_provider_create(qbert, azure_testbed_profile)
    _test_cloud_provider_read_properties(qbert, cp_uuid, azure_testbed_profile)
    return cp_uuid


def _test_cloud_provider_create(qbert, azure_testbed_profile):
    cp_name = os.getenv('QBERT_CLOUD_PROVIDER_NAME')
    if cp_name:
        log.info('Skipping cloud provider creation '
                 'due to specified cloud provider name: %s', cp_name)

        try:
            cps = qbert.list_cloud_providers()
            cp_uuid = next(cp['uuid'] for cp in cps
                           if cp['name'] == cp_name)
        except StopIteration:
            log.error('Did not find the specified cloud provider name: %s', cp_name)
            raise
    else:
        request_body = azure_testbed_profile.get_cloud_provider_create_input()
        cp_name = request_body['name']
        cp_uuid = qbert.create_cloud_provider(request_body)

    expected_cp_names = set(['platform9',  cp_name])
    verify_cloud_providers(qbert, expected_cp_names)
    return cp_uuid


def verify_cloud_providers(qbert, expected_cp_names):
    """Verify that cloud providers were created"""
    cps = qbert.list_cloud_providers()
    cp_names = set(cp['name'] for cp in cps)
    assert_true(expected_cp_names.issubset(cp_names))


def _test_cloud_provider_read_properties(qbert, cp_uuid, azure_testbed_profile):
    locations = qbert.get_cloud_provider_regions(cp_uuid)
    log.info(locations)

    assert_true(len(locations['Regions']) > 0)

    expected_location = azure_testbed_profile.get_cluster_create_input()['location']
    location_present = False
    for location in locations['Regions']:
        if location['RegionName'] == expected_location:
            location_present = True
            break

    assert_true(location_present)

    region_info = qbert.get_cloud_provider_region_info(cp_uuid, expected_location)

    assert_equal(len(region_info), 2)
    assert_true('skus' in region_info)
    assert_true('virtualNetworks' in region_info)
