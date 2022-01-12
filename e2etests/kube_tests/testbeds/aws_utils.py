# Copyright (c) 2016 Platform9 systems. All rights reserved

import os
import boto3
import ipaddress

def _get_platform9_horse_zone_id():
    # the Route53 Zone ID for platform9.horse
    return os.getenv('PLATFORM9_HORSE_ZONE_ID', 'Z2JRKZA5D7I02L')

def _get_route53_session_and_zone_id():
    # these will fail if env vars. undefined
    aws_access_key_id = os.environ['AWS_ACCESS_KEY']
    aws_secret_access_key = os.environ['AWS_SECRET_KEY']
    options = {
        'aws_access_key_id': aws_access_key_id,
        'aws_secret_access_key': aws_secret_access_key
    }
    session = boto3.client('route53', **options)
    zone_id = _get_platform9_horse_zone_id()
    return session, zone_id

def _resource_record_set_for_ips(ips, fqdn):
    return {
        'Name': fqdn + '.',
        'TTL': 60,
        'Type': 'A',
        'ResourceRecords': [ {'Value': ip} for ip in ips ]
    }

def is_IPv6(ipaddr):
    try:
        ipaddr_obj = ipaddress.IPv6Address(ipaddr)
        if ipaddr_obj.version == 6:
            return True
    except ipaddress.AddressValueError as ie:
        return False

def create_dns_record(ips, fqdn):
    """
    Creates a DNS record containing a list of A records, one for each of
    the specified host private IPs, to enable DNS-based load balancing.
    """
    session, zone_id = _get_route53_session_and_zone_id()
    rrs = _resource_record_set_for_ips(ips, fqdn)
    if is_IPv6(ips[0]):
        rrs['Type'] = 'AAAA'
    batch = {'Comment': 'update by duless-multimaster testbed',
             'Changes': [{'Action': 'UPSERT', 'ResourceRecordSet': rrs}]}
    session.change_resource_record_sets(HostedZoneId=zone_id, ChangeBatch=batch)

def delete_dns_record(ips, fqdn):
    """
    Creates a DNS record containing a list of A records, one for each of
    the specified host private IPs, to enable DNS-based load balancing.
    """
    session, zone_id = _get_route53_session_and_zone_id()
    rrs = _resource_record_set_for_ips(ips, fqdn)
    if is_IPv6(ips[0]):
        rrs['Type'] = 'AAAA'
    batch = {'Comment': 'delete by duless-multimaster testbed',
             'Changes': [{'Action': 'DELETE', 'ResourceRecordSet': rrs}]}
    session.change_resource_record_sets(HostedZoneId=zone_id, ChangeBatch=batch)
