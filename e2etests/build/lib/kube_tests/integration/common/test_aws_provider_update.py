import logging
import os
from proboscis.asserts import assert_equal, assert_true

log = logging.getLogger(__name__)


def test_aws_provider_update(qbert, region, uuid):
    _test_cloud_provider_update(qbert, uuid)
    _test_cloud_provider_read_properties(qbert, uuid, region)

def _test_cloud_provider_update(qbert, uuid):
    request_body = {
        'name'   : 'aws-1',
        'key'    : os.environ['HYBRID_ACCESS_KEY'],
        'secret' : os.environ['HYBRID_ACCESS_SECRET']
    }
    qbert.update_cloud_provider(uuid, request_body)

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