luci.project(
    name = "zzz",
    acls = [acl.entry(acl.CQ_COMMITTER, groups = ["g"])],
)

luci.cq_group(
    name = "group",
    watch = [
        cq.refset("https://example.googlesource.com/repo"),
        cq.refset("https://example.googlesource.com/a/repo"),
    ],
)

# Expect errors like:
#
# Traceback (most recent call last):
#   //testdata/cq/dup_repo_1.star: in <toplevel>
#   ...
# Error: ref regexp "refs/heads/master" of "https://example.googlesource.com/a/repo" is already covered by a cq_group, previous declaration:
# Traceback (most recent call last):
#   //testdata/cq/dup_repo_1.star: in <toplevel>
#   ...
