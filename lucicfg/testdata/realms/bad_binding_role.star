lucicfg.enable_experiment("crbug.com/1085650")

luci.project(name = "proj")
luci.realm(
    name = "realm",
    bindings = [
        luci.binding(
            roles = "customRole/undefined",
            users = "a@example.com",
        ),
    ],
)

# Expect errors like:
#
# Traceback (most recent call last):
#   //testdata/realms/bad_binding_role.star: in <toplevel>
#   ...
# Error: luci.binding("customRole/undefined") in "roles" refers to undefined luci.custom_role("customRole/undefined")
