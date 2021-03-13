lucicfg.enable_experiment("crbug.com/1182002")  # short BBv2 names

luci.project(
    name = "project",
    buildbucket = "cr-buildbucket.appspot.com",
    scheduler = "luci-scheduler.appspot.com",
    swarming = "chromium-swarm.appspot.com",
)

luci.builder(
    name = "b1",
    bucket = luci.bucket(name = "ci"),
    executable = luci.recipe(name = "noop", cipd_package = "noop"),
    service_account = "noop@example.com",
    schedule = "triggered",
)

# Expect configs:
#
# === cr-buildbucket.cfg
# buckets {
#   name: "ci"
#   swarming {
#     builders {
#       name: "b1"
#       swarming_host: "chromium-swarm.appspot.com"
#       recipe {
#         name: "noop"
#         cipd_package: "noop"
#         cipd_version: "refs/heads/master"
#       }
#       service_account: "noop@example.com"
#     }
#   }
# }
# ===
#
# === luci-scheduler.cfg
# job {
#   id: "b1"
#   schedule: "triggered"
#   acl_sets: "ci"
#   buildbucket {
#     server: "cr-buildbucket.appspot.com"
#     bucket: "ci"
#     builder: "b1"
#   }
# }
# acl_sets {
#   name: "ci"
# }
# ===
#
# === project.cfg
# name: "project"
# ===
