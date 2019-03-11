LogDog Archivist
================

The LogDog Archivist is tasked by the **Coordinator**, and has the job of
collecting log stream data from **Intermediate Storage** (where it was deposited
by the **Collector**) and loading it into **Archival Storage**. It does this by
scanning through **Intermediate Storage** for consecutive log entries and
constructing archive files:

* The Logs file, consisting of Record IO entries containing the
  `LogStreamDescriptor` protobuf followed by every `LogEntry` protobuf in the
  stream.
* The Index file, consisting of a `LogStreamDescriptor` RecordIO entry followed
  by a `LogIndex` protobuf entry.
* An optional Data file, consisting of the reassembled contiguous raw log stream
  data.

These files are written into **Archival Storage** by the **Archivist** during
archival. After archival is complete, the **Archivist** notifies the
**Coordinator** and the log stream's state is updated.

**Archivist** microservices are designed to operate cooperatively as part of
a scalable cluster. Deploying additional **Archivist** instances will linearly
increase the archival throughput.

**Archivist** instances load the global LogDog configuration, and are
additionally configured via the `Archivist` configuration message in
[config.proto](../../../api/config/svcconfig/config.proto). Configuration
is loaded from the **Coordinator** and the **Configuration Service**.

## Staging

Archival is initially written to a staging storage location. After the archival
successfully completes, the staged files are moved to permanent location using
an inexpensive rename operation.

## Incomplete Logs

It is possible for log streams to be missing data at the time of archival.
