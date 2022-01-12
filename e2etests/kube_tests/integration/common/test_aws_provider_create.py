import logging
import os
import uuid

from proboscis.asserts import assert_equal, assert_true

log = logging.getLogger(__name__)


def test_aws_provider_create(qbert, region):
    cp_uuid = _test_cloud_provider_create(qbert)
    _test_cloud_provider_read_properties(qbert, cp_uuid, region)
    return cp_uuid


def _test_cloud_provider_create(qbert):
    cp_name = os.getenv('QBERT_CLOUD_PROVIDER_NAME')
    if cp_name:
        log.info('Skipping cloud provider creation '
                 'due to specified cloud provider name: %s',
                 cp_name)

        try:
            cps = qbert.list_cloud_providers()
            cp_uuid = next(cp['uuid'] for cp in cps
                           if cp['name'] == cp_name)
        except StopIteration:
            log.error('Did not find the specified cloud provider name: %s',
                      cp_name)
            raise
    else:
        cp_name = 'aws-{0}'.format(uuid.uuid4())
        request_body = {
            'name'   : cp_name,
            'type'   : 'aws',
            'key'    : os.environ['HYBRID_ACCESS_KEY'],
            'secret' : os.environ['HYBRID_ACCESS_SECRET']
        }
        cp_uuid = qbert.create_cloud_provider(request_body)

    expected_cp_names = set(['platform9',  cp_name])
    verify_cloud_providers(qbert, expected_cp_names)
    return cp_uuid


def verify_cloud_providers(qbert, expected_cp_names):
    """Verify that cloud providers were created"""
    cps = qbert.list_cloud_providers()
    cp_names = set(cp['name'] for cp in cps)
    assert_true(expected_cp_names.issubset(cp_names))


def _test_cloud_provider_read_properties(qbert, cp_uuid, region):
    regions = qbert.get_cloud_provider_regions(cp_uuid)
    assert_true(len(regions['Regions']) > 5)

    # Get vpc, keys, flavors, azs, domains for a region
    region_info = qbert.get_cloud_provider_region_info(cp_uuid, region)

    assert_equal(len(region_info), 6)

    # The exact numbers can vary so just looking for presence
    assert_true(len(region_info['vpcs']) >= 0)
    assert_true(len(region_info['keyPairs']) >= 1)
    assert_true(len(region_info['azs']) >= 1)
    assert_true(len(region_info['flavors']) >= 1)
    assert_true(len(region_info['domains']) >= 1)
    assert_true(len(region_info['operatingSystems']) >= 1)
