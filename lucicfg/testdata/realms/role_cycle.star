lucicfg.enable_experiment('crbug.com/1085650')

luci.project(name = 'proj')
luci.custom_role(
    name = 'customRole/r1',
    extends = ['customRole/r2'],
)
luci.custom_role(
    name = 'customRole/r2',
    extends = ['customRole/r1'],
)

# Expect errors like:
#
# Traceback (most recent call last):
#   //testdata/realms/role_cycle.star: in <toplevel>
#   ...
# Error: relation "extends" between luci.custom_role("customRole/r1") and luci.custom_role("customRole/r2") introduces a cycle
