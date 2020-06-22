lucicfg.enable_experiment("crbug.com/1085650")

luci.project(
    name = "proj",
    bindings = [
        luci.binding(
            roles = "bad role",
        ),
    ],
)

# Expect errors like:
#
# Traceback (most recent call last):
#   //testdata/realms/bad_role.star: in <toplevel>
#   ...
# Error: bad "roles": "bad role" should start with "role/" or "customeRole/"
