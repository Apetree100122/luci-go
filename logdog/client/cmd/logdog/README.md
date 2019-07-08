LogDog CLI
==========

LogDog's `logdog` tool is a command-line LogDog **Coordinator** client.

## Subcommands

The `logdog` tool supports several subcommands. To see a list, run `logdog`
without any additional flags.

### Authentication (auth-login)

The `auth-login` subcommand can be used to obtain authentication credentials.
After running `auth-login` successfully, subsequent accesses with the tool will
use the cached authentication credentials.

You can log out using the `auth-logout` subcommand, or check your authentication
status with `auth-info`.

### cat

The `cat` subcommand allows a log stream to be viewed. If the log stream is
still streaming, `logdog` will block, showing new stream data as it becomes
available.

```shell
$ logdog -project <project> cat <prefix>/+/<name>
```

The project may also be integrated into the log stream path. For example, the
previous command is equivalent to:

```shell
$ logdog cat <project>/<prefix>/+/<name>
```

### query

The `query` subcommand allows queries to be executed against a **Coordinator**
instance.

```shell
$ logdog query <params>...
```

The `-json` parameter can be supplied to cause the query to produce detailed
JSON output.

Several types of query constraints are supported. Note that these constraints
are a subset of LogDog's full query API; consequently, support for additional
query constraints may be added in the future.

#### Path

Path queries identify log streams that match the supplied path constraint.
Both the Prefix and Name components of the path can be specified either fully
or globbed with `*` characters according to some rules:

* Full prefix constraints will return log streams that share a Prefix
  * For example `-path 'foo/bar'` will return all log streams that have the
    prefix, "foo/bar".
* Single-component globbing.
  * For example, `-path 'foo/*/baz'`.
* Right-open globbing via `**` will match all log streams that begin with a
  specified path.
  * For example, `-path 'foo/bar/**'`
* Left-open globbing via `**` will match all log streams that end with a
  specified path.
  * For example, `-path '**/baz'`
* Right-open and left-open globbing **cannot** be used in the same Prefix/Name.
* Globbing can be applied to both Prefix and Name.
  * For example, `-path 'foo/bar/**/+/**/stdout` will find all streams that
    have "stdout" in their final name component and belong to a prefix
    beginning with "foo/bar".

#### Timestamps

Queries can be limited by timestamp using the `-before` and `-after` flags.
Timestamps are expressed as [RFC 3339](https://www.ietf.org/rfc/rfc3339.txt)
time strings.

For example:
```shell
$ logdog query -after '1985-04-12T23:20:50.52Z'
```

#### Tags

Queries can be restricted to streams that match supplied tags using one or
more `-tag` constraints.

Tags are specified in one of two forms:

* `-tag <key>` matches all streams that have the "<key>" tag, regardless of its
  value.
* `-tag <key>=<value>` matches all streams that have a "<key>" tag with thed
  value, "<value>".

### ls

The `ls` subcommand allows the user to navigate the log stream space as if it
were a hierarchial directory structure.

To view project-level streams:

```shell
$ logdog ls
myproject

$ logdog ls myproject
foo
bar

$ logdog ls myproject/foo
+

$ logdog ls myproject/foo/+
baz

$ logdog ls myproject/foo/+/baz
```

The `-l` flag may be supplied to cause metadata about each hierarchy component
to be printed.
