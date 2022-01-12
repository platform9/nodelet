import logging
import os
import random
import time
import uuid

from proboscis.asserts import assert_equal, assert_true, fail

from pf9lab.retry import retry
from kube_tests.integration.common import constants, wait_for_cluster_taskstatus, wait_for_cluster_status


CLUSTER_LAUNCH_DELAY = 30
EXTERNAL_DNS_TEMPLATE = 'test-pf9-{0}-api.platform9.systems'
KEY_NAME = 'Used-By-Tests'
KUBERNETES_RESOURCE_TAG = 'KubernetesCluster'
KUBERNETES_RESOURCE_PREFIX = 'kubernetes.io/cluster/'
SERVICES_DNS_TEMPLATE = 'test-pf9-{0}-svc.platform9.systems'
SPOT_PRICE = '0.08' # Slightly higher than on-demand price

class ClusterProfile:
    def __init__(self, number_of_masters, master_flavor, number_of_workers,
     worker_flavor, number_of_spot_workers, spot_worker_flavor, number_of_workers_max):
        self.number_of_masters = number_of_masters
        self.master_flavor = master_flavor
        self.number_of_workers = number_of_workers
        self.worker_flavor = worker_flavor
        self.number_of_spot_workers = number_of_spot_workers
        self.spot_worker_flavor = spot_worker_flavor
        self.number_of_min_workers = number_of_workers
        self.number_of_max_workers = number_of_workers_max

SMALL_PROFILE = ClusterProfile(1, "t2.medium", 1, "t2.medium", 1, "t2.medium", 0)
AUTOSCALER_PROFILE = ClusterProfile(1, "t2.medium", 1, "t2.medium", 0, "t2.medium", 3)
LARGE_PROFILE = ClusterProfile(3, "t2.medium", 1, "t2.medium", 2, "t2.medium", 0)

log = logging.getLogger(__name__)

def test_aws_cluster_create_with_custom_flavor(config, master_flavor, worker_flavor, spot_flavor):
    global MASTER_FLAVOR
    global WORKER_FLAVOR
    global SPOT_WORKER_FLAVOR
    MASTER_FLAVOR = master_flavor
    WORKER_FLAVOR = worker_flavor
    SPOT_WORKER_FLAVOR = spot_flavor
    all_cluster_uuids, all_subnets = test_aws_cluster_create(**config)
    return all_cluster_uuids, all_subnets

def test_aws_cluster_create(qbert, cp_uuid, region, ec2, _ec2,
                            template_key, is_private, runtime_config, kubeRoleVersions,
                            mtu_size=None, upgrade=False, runtime="docker"):
    suffix = _get_cluster_suffix()
    domain_id = _get_domain_id(qbert, cp_uuid, region)
    az_names = _get_random_az_names(qbert, cp_uuid, region)
    nodepool_uuid = _get_cp_nodepool_uuid(qbert, cp_uuid)
    visibility = 'private' if is_private else 'public'

    all_cluster_uuids = dict()
    network_plugin = 'flannel' if upgrade else 'calico'
    kubeRoleVersion = kubeRoleVersions[0]
    just_launched, aws_complete_uuid = _get_or_deploy_cluster(
        qbert, False, 'QBERT_AWS_COMPLETE_CLUSTER_NAME',
        '{0}-aws-complete-{1}-{2}'.format(template_key, visibility, suffix),
        template_key, region, domain_id, nodepool_uuid,
        azList=az_names, is_private=is_private,
        runtime_config=runtime_config, network_plugin=network_plugin, mtu_size=mtu_size,
        kubeRoleVersion=kubeRoleVersion, runtime=runtime)
    all_cluster_uuids['aws_complete_uuid'] = aws_complete_uuid
    # All resources are created once a cluster's taskStatus moves from
    # 'creating' to 'converging' because that means `terraform apply` has finished
    wait_for_cluster_taskstatus('converging', qbert, [aws_complete_uuid])

    # Expect a public subnet for every availability zone used by the cluster
    num_expected_subnets = len(az_names)
    if is_private:
        # Expect an additional private subnet for each public subnet
        num_expected_subnets = 2 * num_expected_subnets
    _wait_cluster_vpc_subnets(num_expected_subnets, qbert, aws_complete_uuid, cp_uuid, region)

    vpc = _get_cluster_vpc(qbert, aws_complete_uuid, cp_uuid, region)

    subnet_ids = [subnet['SubnetId'] for subnet in vpc['Subnets']
                  if subnet['MapPublicIpOnLaunch']]
    private_subnet_ids = [subnet['SubnetId']
                          for subnet in vpc['Subnets']
                          if not subnet['MapPublicIpOnLaunch']]
    if is_private:
        assert_equal(len(subnet_ids), len(private_subnet_ids))
    else:
        assert_equal(len(private_subnet_ids), 0)

    _test_subnet_conflict(qbert, template_key, region, domain_id, nodepool_uuid,
                          subnets=subnet_ids, vpc_id=vpc['VpcId'],
                          private_subnets=private_subnet_ids,
                          is_private=is_private)

    aws_subnets = _get_or_create_subnets(
        'QBERT_AWS_CLUSTER_SUBNET_IDS', vpc['VpcId'], az_names,
        '10.0.2{0}.0/24', True, ec2, _ec2)

    if is_private:
        aws_private_subnets = _get_or_create_subnets(
            'QBERT_AWS_PRIVATE_PRIVATE_SUBNET_IDS',
            vpc['VpcId'], az_names, '10.0.3{0}.0/24',
            False, ec2, _ec2)
        _associate_private_subnets_with_nat(
            vpc['VpcId'], aws_private_subnets, ec2)
    else:
        aws_private_subnets = []

    subnet_ids = [s.subnet_id for s in aws_subnets]
    private_subnet_ids = [s.subnet_id for s in aws_private_subnets]
    network_plugin = 'flannel' if upgrade else 'calico'

    if (len(kubeRoleVersions) > 1):
        kubeRoleVersion = kubeRoleVersions[1]

    _, aws_uuid = _get_or_deploy_cluster(
        qbert, just_launched, 'QBERT_AWS_PRIVATE_CLUSTER_NAME',
        '{0}-aws-{1}-{2}'.format(template_key, visibility, suffix),
        template_key, region, domain_id, nodepool_uuid,
        vpc_id=vpc['VpcId'], subnets=subnet_ids,
        private_subnets=private_subnet_ids, is_private=is_private,
        runtime_config=runtime_config, network_plugin=network_plugin, profile=SMALL_PROFILE,
        kubeRoleVersion=kubeRoleVersion, runtime=runtime)

    all_cluster_uuids['aws_uuid'] = aws_uuid
    wait_for_cluster_taskstatus('converging', qbert, [aws_uuid])

    _wait_subnets_tagged(qbert, aws_subnets + aws_private_subnets, aws_uuid)

    if qbert.subnet_shareable:
        # Cluster should be able to share subnets if they are
        # created outside Qbert, which is the case here.
        _, aws_uuid_shared = _get_or_deploy_cluster(
            qbert, just_launched, 'QBERT_AWS_PRIVATE_CLUSTER_NAME',
            '{0}-aws-shared-subnet-{1}-{2}'.format(template_key, visibility, suffix),
            template_key, region, domain_id, nodepool_uuid,
            vpc_id=vpc['VpcId'], subnets=subnet_ids,
            private_subnets=private_subnet_ids, is_private=is_private,
            runtime_config=runtime_config, network_plugin='flannel', profile=AUTOSCALER_PROFILE,
            kubeRoleVersion=kubeRoleVersion, runtime=runtime)
        all_cluster_uuids['aws_uuid_shared'] = aws_uuid_shared

    all_subnets = [aws_subnets, aws_private_subnets]

    # Wait for all clusters to finish converging
    wait_for_cluster_taskstatus('success', qbert, list(all_cluster_uuids.values()))
    wait_for_cluster_status('ok', qbert, list(all_cluster_uuids.values()))
    return all_cluster_uuids, all_subnets


def _get_cluster_suffix():
    user = os.environ['USER']
    build_num = os.getenv('BUILD_NUMBER')
    if build_num:
        suffix = 'bld{0}'.format(build_num)
    else:
        suffix = str(uuid.uuid4())[-3:]
    return '{0}-{1}'.format(user, suffix)


def _get_random_az_names(qbert, cp_uuid, region):
    """
    Pick a random number between 1 and number of AZs available. Then
    pick randomly pick that number of AZs. This is to ensure we try
    different possible scenarios of picking AZs but it is not
    guaranteed that every scenario will get tested.
    """
    region_info = qbert.get_cloud_provider_region_info(cp_uuid, region)
    azs = region_info['azs']
    select_azs_count = int(
        os.getenv('NUM_AZS_TO_USE', random.randint(1, len(azs))))
    assert_true(0 < select_azs_count <= len(azs))
    selected_azs = _random_combination(azs, select_azs_count)
    azNames = []
    for az in selected_azs:
        azNames.append(az['ZoneName'])

    log.info('Selected %d AZs: %s', len(selected_azs), azNames)
    return azNames


def _random_combination(iterable, r):
    """
    Randomly select r items from an iterable set of elements.
    """
    pool = tuple(iterable)
    n = len(pool)
    indices = sorted(random.sample(range(n), r))
    return tuple(pool[i] for i in indices)


def _get_or_deploy_cluster(qbert, just_launched, name_env_var,
                           *args, **kwargs):
    if os.getenv(name_env_var):
        name = os.getenv(name_env_var)
        return just_launched, qbert.get_cluster(name)['uuid']
    if just_launched:
        time.sleep(CLUSTER_LAUNCH_DELAY)
    return True, _deploy_cluster(qbert, *args, **kwargs)


def _deploy_cluster(qbert, name, image, region, domain_id, nodepool_uuid,
                    vpc_id=None, azList=None, private_subnets=None,
                    is_private=None, subnets=None, runtime_config=None, profile=LARGE_PROFILE,
                    network_plugin=None, mtu_size=None, kubeRoleVersion=None, runtime="docker"):
    dns_tag = str(uuid.uuid4())[:5]
    external_dns_name = EXTERNAL_DNS_TEMPLATE.format(dns_tag)
    services_dns_name = SERVICES_DNS_TEMPLATE.format(dns_tag)
    # Today, we don't have support for VPN based testbeds with AWS. This piece
    # of code only deploys AWS clusters today. The accidental setup of the
    # USE_PROXY env variable can lead to the tests using proxy, when it cannot
    # reach the local proxy. Till these types of testbeds are supported, set the
    # proxy to be ''.
    #http_proxy = os.environ.get('USE_PROXY', '')
    http_proxy = ''
    internal_elb = os.environ.get('USE_INTERNAL_ELB', False)
    appcatalog_enabled = os.getenv('USE_APP_CATALOG') == 'true'
    body = {'name': name,
            'nodePoolUuid': nodepool_uuid,
            'containersCidr': constants.CONTAINERS_CIDR,
            'servicesCidr': constants.SERVICES_CIDR,
            'externalDnsName': external_dns_name,
            'serviceFqdn': services_dns_name,
            'numMasters': profile.number_of_masters,
            'ami': image,
            'masterFlavor': profile.master_flavor,
            'workerFlavor': profile.worker_flavor,
            'region': region,
            'sshKey': KEY_NAME,
            'domainId': domain_id,
            'debug': 'true',
            'kubeRoleVersion': kubeRoleVersion,
            'isPrivate': is_private,
            'httpProxy': http_proxy,
            'internalElb': internal_elb,
            'appCatalogEnabled': appcatalog_enabled,
            'allowWorkloadsOnMaster': os.getenv('ALLOW_WORKLOADS_ON_MASTER', 'true') == 'true',
            'containerRuntime': runtime}
    if profile.number_of_max_workers > profile.number_of_min_workers:
        body['numMinWorkers'] = profile.number_of_min_workers
        body['numMaxWorkers'] = profile.number_of_max_workers
        body['enableCAS'] = True
    else:
        body['numWorkers'] = profile.number_of_workers
    if network_plugin:
        body['networkPlugin'] = network_plugin
        body['privileged'] = True
    if azList:
        body['azs'] = azList
    if subnets:
        body['subnets'] = subnets
    if private_subnets:
        body['privateSubnets'] = private_subnets
    if 'ENABLE_PRIVILEGED_CONTAINERS' in os.environ:
        body['privileged'] = True
    if 'EXTRA_OPTS' in os.environ:
        body['extraOpts'] = os.environ['EXTRA_OPTS']
    if vpc_id:
        body['vpc'] = vpc_id
    if runtime_config:
        body['runtimeConfig'] = runtime_config
    if mtu_size:
        body['mtuSize'] = str(mtu_size)
    if 'USE_SPOT_INSTANCES' in os.environ:
        body['numSpotWorkers'] = profile.number_of_spot_workers
        body['spotWorkerFlavor'] = profile.spot_worker_flavor
        body['spotPrice'] = SPOT_PRICE
    # AWS may not work without encapsulation and NAT'ing
    # Test BGP and non-encap in an on-prem environment without security groups
    if network_plugin == 'calico':
        body['calicoIpIpMode'] = 'Always'
        body['calicoNatOutgoing'] = True

    log.info('Creating cluster %s', name)
    return qbert.create_cluster(body, 'v4')


@retry(log=log, max_wait=300, interval=20)
def _wait_cluster_vpc_subnets(num_expected_subnets, qbert, cluster_uuid,
        cp_uuid, region):
    """
    Wait until the details of the VPC used by the cluster contain complete
    subnets information.
    """
    vpc = _get_cluster_vpc(qbert, cluster_uuid, cp_uuid, region)
    return 'Subnets' in vpc and len(vpc['Subnets']) >= num_expected_subnets


def _get_cluster_vpc(qbert, cluster_uuid, cp_uuid, region):
    """
    Return the details of the VPC used by the cluster.
    """
    region_info = qbert.get_cloud_provider_region_info(cp_uuid, region)
    for vpc in region_info['vpcs']:
        for tag in vpc['Tags']:
            key = tag['Key']
            val = tag['Value']
            if key == 'ClusterUuid' and val == cluster_uuid:
                return vpc
    raise RuntimeError("No vpc found for cluster %s" % cluster_uuid)


def _test_subnet_conflict(qbert, image, region, domain_id, nodepool_uuid, **kwargs):
    try:
        _deploy_cluster(qbert, 'ubuntu-aws-conflict', image, region,
                        domain_id, nodepool_uuid, **kwargs)
        fail('Expected error to be thrown when deploying to subnet '
             'already in use by a cluster')
    except Exception as e:
        # FIXME(vann): Remove in 3.3
        if 'already belongs to a cluster' in str(e):
            assert_true('already belongs to a cluster' in str(e))
        else:
            assert_true('400 Client Error: Bad Request' in str(e))


def _get_or_create_subnets(subnet_env_var, vpc_id, az_names,
                           subnet_cidr_template, map_public_ip_on_launch,
                           ec2, _ec2):
    """
    Create subnets from scratch if needed. See FIXME IAAS-7382
    for details on why we don't re-use subnets created from the
    complete clusters
    """
    if os.getenv(subnet_env_var):
        subnet_ids = os.getenv(subnet_env_var).split(',')
        subnets = [ec2.Subnet(sid) for sid in subnet_ids]
    else:
        subnets = _create_subnets(
            vpc_id,
            az_names,
            subnet_cidr_template,
            map_public_ip_on_launch,
            ec2,
            _ec2)
    return subnets


def _create_subnets(vpc_id, az_names, subnet_cidr_template,
                    map_public_ip_on_launch, ec2, _ec2):
    """
    Create a subnet in the vpc specified for each az_name

    The subnet cidr template specified should not conflict with the
    hardcoded values in the Qbert AWS cloud provider code.
    :returns list of boto3.ec2.subnet
    """
    subnets = []
    for i, az_name in enumerate(az_names):
        subnet_cidr = subnet_cidr_template.format(i)
        subnet = ec2.create_subnet(VpcId=vpc_id, CidrBlock=subnet_cidr,
                                   AvailabilityZone=az_name)
        subnets.append(subnet)

    if not map_public_ip_on_launch:
        return subnets
    for subnet in subnets:
        _ec2.modify_subnet_attribute(
            SubnetId=subnet.subnet_id,
            MapPublicIpOnLaunch={'Value': True})

    return subnets


def _associate_private_subnets_with_nat(vpc_id, subnets, ec2):
    """
    :param subnets list of boto3.ec2.Subnet'
    """
    route_table_filters = [{
        'Name': 'vpc-id',
        'Values': [vpc_id]
    }]
    vpc_rts = ec2.route_tables.filter(Filters=route_table_filters)
    nat_rt = next(rt for rt in vpc_rts for route in rt.routes
                  if route.nat_gateway_id)
    for subnet in subnets:
        nat_rt.associate_with_subnet(SubnetId=subnet.id)


@retry(log=log, max_wait=300, interval=20)
def _wait_subnet_tagged(qbert, subnet, expected_cluster_uuid):
    subnet.reload()
    tags = subnet.tags
    if tags is None:
        return False
    match_found = False
    qbert.subnet_shareable = False
    #we have to iterate through all tags can not fail
    #fast on first mismatch as subnets can be shared now
    #and thus can have multiple tags
    for tag in tags:
        #older clusters
        if tag['Key'] == KUBERNETES_RESOURCE_TAG:
            match_found =  tag['Value'] == expected_cluster_uuid
        #new clusters
        if tag['Key'].startswith(KUBERNETES_RESOURCE_PREFIX):
            match_found = tag['Key'].endswith(expected_cluster_uuid)
            #This state represents if we allow sharing of subnets or not.
            #piggy-backing on existing loop over tags to save cost.
            qbert.subnet_shareable = True
        if match_found:
            break
    return match_found


def _wait_subnets_tagged(qbert, subnets, cluster_uuid):
    """
    :param subnets list of boto3.ec2.subnets:
    """
    for subnet in subnets:
        _wait_subnet_tagged(qbert, subnet, cluster_uuid)


def _get_cp_nodepool_uuid(qbert, cp_uuid):
    return next(cp['nodePoolUuid']
                for cp in qbert.list_cloud_providers()
                if cp['uuid'] == cp_uuid)


def _get_domain_id(qbert, cp_uuid, region):
    region_info = qbert.get_cloud_provider_region_info(cp_uuid, region)
    return region_info['domains'][0]['Id']
