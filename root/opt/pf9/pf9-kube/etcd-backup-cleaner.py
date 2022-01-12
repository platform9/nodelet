# Copyright (c) 2019, Platform9 Systems, Inc. All rights reserved.

import argparse
from calendar import timegm
import os
from os.path import isfile, join
import re
import time


def sort_and_delete_extra_files(etcd_backup_dir, file_regex, num_files_to_retain):
    print("Looking for file pattern: {} and retaining {} file(s) in {} directory".
          format(file_regex, num_files_to_retain, etcd_backup_dir))
    file_name_to_epoch_list = []

    for file_name in os.listdir(etcd_backup_dir):
        if isfile(join(etcd_backup_dir, file_name)):
            print("file: {}".format(file_name))
            # 2019-10-03_18:54:08
            file_match = re.match(file_regex, file_name)
            if file_match:
                print("Found a match: {}".format(file_match.group(1)))
                file_name_date = file_match.group(1)
                try:
                    utc_time = time.strptime(file_name_date, "%Y-%m-%d_%H:%M:%S")
                    print("utc_time: {}".format(utc_time))
                    epoch_time = timegm(utc_time)
                    print("epoch_time: {}".format(epoch_time))
                    file_name_to_epoch_list.append((file_name, epoch_time,))
                except Exception as e:
                    print("Error in parsing {}, so ignoring".format(file_name_date))
    print("Before sorting: {}".format(file_name_to_epoch_list))

    file_name_to_epoch_list = sorted(file_name_to_epoch_list, key=lambda x: x[1])
    print("Sorted order by datetime: {}".format(file_name_to_epoch_list))

    if len(file_name_to_epoch_list) > num_files_to_retain:
        for files_to_delete in file_name_to_epoch_list[:-num_files_to_retain]:
            print("deleting backup file: {}".format(files_to_delete))
            os.remove(join(etcd_backup_dir, files_to_delete[0]))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("etcd_backup_directory", help="ETCD backup directory")
    parser.add_argument("max_num_backups", type=int, default=2, help="Maximum number of backups")

    args = parser.parse_args()

    etcd_backup_dir = args.etcd_backup_directory
    max_files_to_keep = args.max_num_backups

    # Keep upto max_files_to_keep of .db files
    sort_and_delete_extra_files(etcd_backup_dir, 'etcd-snapshot-([\d\-_:]+)_UTC\.db$', max_files_to_keep)
    # Keep 1 of .db.part file if at all it exists
    sort_and_delete_extra_files(etcd_backup_dir, 'etcd-snapshot-([\d\-_:]+)_UTC\.db.part$', 1)


if __name__ == '__main__':
    main()
