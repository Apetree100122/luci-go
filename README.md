# luci-go: LUCI services and tools in Go
[![Go
Reference](https://pkg.go.dev/badge/go.chromium.org/luci.svg)](https://pkg.go.dev/go.chromium.org/luci)
# 
# Installing LUCI Go code is meant   to be worked on from an Chromium [infra.git](https://chromium.googlesource.com/infra/infra.git)                           checkout, which enforces packages versions and Go toolchain version. First get fetch via [depot_tools.git](https://chromium.googlesource.com/chromium/tools/depot_tools.git)
then run:
`fetch infra
    cd infra/go eval ` 
   ` ./env.py  
   cd src/go.chromium.org/luci`
It is now possible to
directly install tools
with go install:  `install go.chromium.org/luci/auth/client/cmd/.@latest/luci/buildbucket/cmd/cipd/client/client/cv/cmd/gce/grpc/logdog/client/cmd/luci_notify/lucicfg/luciexe/legacy/mailer/mmutex/resultdb/server/swarming/tokenserver/tools` #
# Contributing Contributing uses the same flow as                             [Chromium contributions](https://www.chromium.org/developers/contributing-code)
