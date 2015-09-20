BlobSnap
========

BlobSnap is a snapshot-based backup system built on top of [BlobStash](https://github.com/tsileo/blobstash), designed to provide "time machine" like features.

## Features

- Content addressed (with [BLAKE2b](https://blake2.net) as hashing algorithm), files are split into blobs, and retrieved by hash, blobs are deduplicated (incremental backups by default).
- Read-only FUSE file system to navigate backups/snapshots.
- Take snapshot automatically every x minutes, using a separate client-side scheduler (provides Arq/time machine like backup).
- Possibility to incrementally archive blobs to AWS Glacier (see BlobStash docs).
- Support for backing-up multiple hosts (you can force a different host to split backups into "different buckets").

Draws inspiration from [Camlistore](camlistore.org) and [bup](https://github.com/bup/bup) (files are split into multiple blobs using a rolling checksum).

## Components

### Fuse file system

The (read-only) Fuse file system is the most convenient way to restore/navigate snapshots.

There is two magic directories at the root:

- **latest**: it contains the latest version of every snapshots/backups.
- **snapshots**: it let you navigate for every snapshots, you can see every versions.

```console
$ blobstash mount /backups
2014/05/12 17:26:34 Mounting read-only filesystem on /backups
Ctrl+C to unmount.
```

```console
$ ls /backups
tomt0m
$ ls /backups/tomt0m
latest  snapshots
$ ls /backups/tomt0m/latest
writing
$ ls /backups/tomt0m/latest/writing
file1  file2  file3
$ ls /backups/tomt0m/snapshots/writing
2014-05-11T11:01:07+02:00  2014-05-11T18:36:06+02:00  2014-05-12T17:25:47+02:00
$ ls /backups/tomt0m/snapshots/writing/2014-05-11T18:36:06+02:00/writing
file1  file2  file3
```
### Command-line client

**blobsnap** is the command-line client to perform/restore snapshots/backups.

```console
$ blobsnap put /path/to/dir/or/file
```

### Backup scheduler

The backup scheduler allows you to perform snapshots on a given basis.

The spec expect a cron-like spec, or a custom spec supported by [http://godoc.org/github.com/robfig/cron](http://godoc.org/github.com/robfig/cron).

The scheduler support a special [anacron-like](http://anacron.sourceforge.net/) mode, designed for laptop users.

```json
{
    "anacron_mode": false,
    "snapshots": [
        {
            "path": "/path/to/backup",
            "spec": "0 30 * * * *"
        },
        {
            "path": "/path/to/another/backup",
            "spec": "@every 12h"
        }
    ]
}
```

```console
$ blobsnap sched
```

## Roadmap / Ideas

- A **stats** subcommand
- an Android app to backup Android devices
- Follow .gitignore file
- Fill an issue!

## Donate!

[![Flattr this git repo](http://api.flattr.com/button/flattr-badge-large.png)](https://flattr.com/submit/auto?user_id=tsileo&url=https%3A%2F%2Fgithub.com%2Ftsileo%2Fblobsnap)

BTC 1HpHxwNUmXfrU9MR9WTj8Mpg1YUEry9MF4

## License

Copyright (c) 2014-2015 Thomas Sileo and contributors. Released under the MIT license.
