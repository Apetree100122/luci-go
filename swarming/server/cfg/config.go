// Copyright 2023 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cfg

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"regexp"
	"sort"
	"time"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/config"
	"go.chromium.org/luci/config/cfgclient"
	"go.chromium.org/luci/config/validation"
	"go.chromium.org/luci/gae/service/datastore"

	configpb "go.chromium.org/luci/swarming/proto/config"
	"go.chromium.org/luci/swarming/server/cfg/internalcfgpb"
)

// Individually recognized config files.
const (
	settingsCfg = "settings.cfg"
	poolsCfg    = "pools.cfg"
	botsCfg     = "bots.cfg"
)

// hookScriptRe matches paths like `scripts/hooks.py`. It intentionally doesn't
// match subdirectories of `scripts/` since they contain unit tests the server
// doesn't care about.
var hookScriptRe = regexp.MustCompile(`scripts/[^/]+\.py`)

const (
	// A pseudo-revision of an empty config.
	emptyRev = "0000000000000000000000000000000000000000"
	// A digest of a default config (calculated in the test).
	emptyDigest = "0NpkIis/WMci8PDKkLD3PB/t8B86nbBVjyD59iosjOM"
)

// Config is an immutable queryable representation of Swarming server configs.
//
// It is a snapshot of configs at some particular revision. Use an instance of
// Provider to get it.
type Config struct {
	// Revision is the config repo commit the config was loaded from.
	Revision string
	// Digest is derived exclusively from the configs content.
	Digest string
	// Fetched is when the stored config was fetched from LUIC Config.
	Fetched time.Time
	// Refreshed is when the process config was fetched from the datastore.
	Refreshed time.Time

	settings  *configpb.SettingsCfg
	poolMap   map[string]*Pool // pool name => config
	poolNames []string         // sorted list of pool names
	botGroups *botGroups       // can map bot ID to a bot group config
}

// Pool returns a config for the given pool or nil if there's no such pool.
func (cfg *Config) Pool(name string) *Pool {
	return cfg.poolMap[name]
}

// Pools returns a sorted list of all known pools.
func (cfg *Config) Pools() []string {
	return cfg.poolNames
}

// BotGroup returns a BotGroup config matching the given bot ID.
//
// Understands composite bot IDs, see HostBotID(...). Always returns some
// config (never nil). If there's no config assigned to a bot, returns a default
// config.
func (cfg *Config) BotGroup(botID string) *BotGroup {
	hostID := HostBotID(botID)

	// If this is a composite bot ID, try to find if there's a config for this
	// *specific* composite ID first. This acts as an override if we need to
	// single-out a bot that uses a concrete composite IDs.
	if hostID != botID {
		if group := cfg.botGroups.directMatches[botID]; group != nil {
			return group
		}
	}

	// Otherwise look it up based on the host ID (which is the same as bot ID
	// for non-composite IDs).
	if group := cfg.botGroups.directMatches[hostID]; group != nil {
		return group
	}
	if _, group, ok := cfg.botGroups.prefixMatches.LongestPrefix(hostID); ok {
		return group.(*BotGroup)
	}
	return cfg.botGroups.defaultGroup
}

// UpdateConfigs fetches the most recent server configs from LUCI Config and
// stores them in the local datastore if they appear to be valid.
//
// Called from a cron job once a minute.
func UpdateConfigs(ctx context.Context) error {
	// Fetch known config files and everything that looks like a hooks script.
	files, err := cfgclient.Client(ctx).GetConfigs(ctx, "services/${appid}",
		func(path string) bool {
			return path == settingsCfg || path == poolsCfg || path == botsCfg || hookScriptRe.MatchString(path)
		}, false)
	if err != nil && !errors.Is(err, config.ErrNoConfig) {
		return errors.Annotate(err, "failed to fetch the most recent configs from LUCI Config").Err()
	}

	// Get the revision, to check if we have seen it already.
	var revision string
	if len(files) == 0 {
		// This can happen in new deployments.
		logging.Warningf(ctx, "There are no configuration files in LUCI Config")
		revision = emptyRev
	} else {
		// Per GetConfigs logic, all files are at the same revision. Pick the first.
		for _, cfg := range files {
			revision = cfg.Revision
			break
		}
	}

	// No need to do anything if already processed this revision.
	lastRev := &configBundleRev{Key: configBundleRevKey(ctx)}
	switch err := datastore.Get(ctx, lastRev); {
	case err == nil && lastRev.Revision == revision:
		logging.Infof(ctx, "Configs are already up-to-date at rev %s (fetched %s ago)", lastRev.Revision, clock.Since(ctx, lastRev.Fetched).Round(time.Second))
		return nil
	case err == nil && lastRev.Revision != revision:
		logging.Infof(ctx, "Configs revision changed %s => %s", lastRev.Revision, revision)
	case errors.Is(err, datastore.ErrNoSuchEntity):
		logging.Infof(ctx, "First config import ever at rev %s", revision)
	case err != nil:
		return errors.Annotate(err, "failed to fetch the latest processed revision from datastore").Err()
	}

	// Parse and re-validate.
	bundle, err := parseAndValidateConfigs(ctx, revision, files)
	if err != nil {
		return errors.Annotate(err, "bad configs at rev %s", revision).Err()
	}

	// Store as the new authoritative config.
	err = datastore.RunInTransaction(ctx, func(ctx context.Context) error {
		now := clock.Now(ctx).UTC()
		return datastore.Put(ctx,
			&configBundle{
				Key:      configBundleKey(ctx),
				Revision: bundle.Revision,
				Digest:   bundle.Digest,
				Fetched:  now,
				Bundle:   bundle,
			},
			&configBundleRev{
				Key:      configBundleRevKey(ctx),
				Revision: bundle.Revision,
				Digest:   bundle.Digest,
				Fetched:  now,
			})
	}, nil)

	if err != nil {
		return errors.Annotate(err, "failed to store configs at rev %s in the datastore", bundle.Revision).Err()
	}

	logging.Infof(ctx, "Stored configs at rev %s (digest %s)", bundle.Revision, bundle.Digest)
	return nil
}

// defaultConfigs returns default config protos used on an "empty" server.
func defaultConfigs() *internalcfgpb.ConfigBundle {
	return &internalcfgpb.ConfigBundle{
		Revision: emptyRev,
		Digest:   emptyDigest,
		Settings: withDefaultSettings(&configpb.SettingsCfg{}),
		Pools:    &configpb.PoolsCfg{},
		Bots: &configpb.BotsCfg{
			TrustedDimensions: []string{"pool"},
			BotGroup: []*configpb.BotGroup{
				{
					Dimensions: []string{"pool:unassigned"},
					Auth: []*configpb.BotAuth{
						{RequireLuciMachineToken: true, LogIfFailed: true},
					},
				},
			},
		},
		Scripts: map[string]string{},
	}
}

// parseAndValidateConfigs parses config files fetched from LUCI Config, if any.
func parseAndValidateConfigs(ctx context.Context, rev string, files map[string]config.Config) (*internalcfgpb.ConfigBundle, error) {
	bundle := defaultConfigs()
	bundle.Revision = rev

	// Parse and validate files individually.
	if err := parseAndValidate(ctx, files, settingsCfg, bundle.Settings, validateSettingsCfg); err != nil {
		return nil, err
	}
	if err := parseAndValidate(ctx, files, poolsCfg, bundle.Pools, validatePoolsCfg); err != nil {
		return nil, err
	}
	if err := parseAndValidate(ctx, files, botsCfg, bundle.Bots, validateBotsCfg); err != nil {
		return nil, err
	}

	// Make sure all referenced hook scripts are actually present. Collect them.
	for idx, group := range bundle.Bots.BotGroup {
		if group.BotConfigScript == "" {
			continue
		}
		if _, ok := bundle.Scripts[group.BotConfigScript]; ok {
			continue
		}
		body, ok := files["scripts/"+group.BotConfigScript]
		if !ok {
			return nil, errors.Reason("bot group #%d refers to undefined bot config script %q", idx+1, group.BotConfigScript).Err()
		}
		bundle.Scripts[group.BotConfigScript] = body.Content
	}

	// Derive the digest based exclusively on configs content, regardless of the
	// revision. The revision can change even if configs are unchanged. The digest
	// is used to know when configs are changing **for real**.
	visit := []string{settingsCfg, poolsCfg, botsCfg}
	for script := range bundle.Scripts {
		visit = append(visit, "scripts/"+script)
	}
	sort.Strings(visit)
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "version 1\n")
	for _, path := range visit {
		_, _ = fmt.Fprintf(h, "%s\n%d\n%s\n", path, len(files[path].Content), files[path].Content)
	}
	bundle.Digest = base64.RawStdEncoding.EncodeToString(h.Sum(nil))

	return bundle, nil
}

// parseAndValidate parses and validated one text proto config file.
func parseAndValidate[T any, TP interface {
	*T
	proto.Message
}](ctx context.Context,
	files map[string]config.Config,
	path string,
	cfg *T,
	validate func(ctx *validation.Context, t *T),
) error {
	// Parse it if it is present. Otherwise use the default value of `cfg`.
	if body := files[path]; body.Content != "" {
		if err := prototext.Unmarshal([]byte(body.Content), TP(cfg)); err != nil {
			return errors.Annotate(err, "%s", path).Err()
		}
	} else {
		logging.Warningf(ctx, "There's no %s config, using default", path)
	}
	// Pass through the validation, abort on fatal errors, allow warnings.
	valCtx := validation.Context{Context: ctx}
	validate(&valCtx, cfg)
	if err := valCtx.Finalize(); err != nil {
		var valErr *validation.Error
		if errors.As(err, &valErr) {
			blocking := valErr.WithSeverity(validation.Blocking)
			if blocking != nil {
				return errors.Annotate(blocking, "%s", path).Err()
			}
		} else {
			return errors.Annotate(err, "%s", path).Err()
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

// configBundle is an entity that stores latest configs as compressed protos.
type configBundle struct {
	_ datastore.PropertyMap `gae:"-,extra"`

	Key      *datastore.Key              `gae:"$key"`
	Revision string                      `gae:",noindex"`
	Digest   string                      `gae:",noindex"`
	Fetched  time.Time                   `gae:",noindex"`
	Bundle   *internalcfgpb.ConfigBundle `gae:",zstd"`
}

// configBundleRev just stores the metadata for faster fetches.
type configBundleRev struct {
	_ datastore.PropertyMap `gae:"-,extra"`

	Key      *datastore.Key `gae:"$key"`
	Revision string         `gae:",noindex"`
	Digest   string         `gae:",noindex"`
	Fetched  time.Time      `gae:",noindex"`
}

// configBundleKey is a key of the singleton configBundle entity.
func configBundleKey(ctx context.Context) *datastore.Key {
	return datastore.NewKey(ctx, "ConfigBundle", "", 1, nil)
}

// configBundleRevKey is a key of the singleton configBundleRev entity.
func configBundleRevKey(ctx context.Context) *datastore.Key {
	return datastore.NewKey(ctx, "ConfigBundleRev", "", 1, configBundleKey(ctx))
}

// fetchFromDatastore fetches the config from the datastore.
//
// If there's no config in the datastore, returns some default empty config.
//
// If `cur` is not nil its (immutable) parts may be used to construct the
// new Config in case they didn't change.
func fetchFromDatastore(ctx context.Context, cur *Config) (*Config, error) {
	// If already have a config, check if we really need to reload it.
	if cur != nil {
		rev := &configBundleRev{Key: configBundleRevKey(ctx)}
		switch err := datastore.Get(ctx, rev); {
		case errors.Is(err, datastore.ErrNoSuchEntity):
			rev.Revision = emptyRev
			rev.Digest = emptyDigest
		case err != nil:
			return nil, errors.Annotate(err, "fetching configBundleRev").Err()
		}
		if cur.Digest == rev.Digest {
			clone := *cur
			clone.Revision = rev.Revision
			clone.Fetched = rev.Fetched
			clone.Refreshed = clock.Now(ctx).UTC()
			return &clone, nil
		}
	}

	// Either have no config or the one in the datastore is different. Get it.
	bundle := &configBundle{Key: configBundleKey(ctx)}
	switch err := datastore.Get(ctx, bundle); {
	case errors.Is(err, datastore.ErrNoSuchEntity):
		bundle.Revision = emptyRev
		bundle.Digest = emptyDigest
		bundle.Bundle = defaultConfigs()
	case err != nil:
		return nil, errors.Annotate(err, "fetching configBundle").Err()
	}

	// Transform config protos into data structures optimized for config queries.
	// This should never really fail, since we store only validated configs. If
	// this fails, the entire service will eventually go offline since new
	// processes won't be able to load initial copy of the config (while old
	// processes will keep using last known good copies, until eventually they
	// all terminate).
	cfg, err := buildQueriableConfig(ctx, bundle)
	if err != nil {
		logging.Errorf(ctx,
			"Broken config in the datastore at rev %s (digest %s, fetched %s ago): %s",
			bundle.Revision, bundle.Digest, clock.Since(ctx, bundle.Fetched), err,
		)
		return nil, errors.Annotate(err, "broken config in the datastore").Err()
	}
	logging.Infof(ctx, "Loaded configs at rev %s", cfg.Revision)
	return cfg, nil
}

// buildQueriableConfig transforms config protos into data structures optimized
// for config queries.
func buildQueriableConfig(ctx context.Context, ent *configBundle) (*Config, error) {
	pools, err := newPoolsConfig(ent.Bundle.Pools)
	if err != nil {
		return nil, errors.Annotate(err, "bad pools.cfg").Err()
	}
	poolNames := make([]string, 0, len(pools))
	for name := range pools {
		poolNames = append(poolNames, name)
	}
	sort.Strings(poolNames)

	botGroups, err := newBotGroups(ent.Bundle.Bots)
	if err != nil {
		return nil, errors.Annotate(err, "bad bots.cfg").Err()
	}

	return &Config{
		Revision:  ent.Revision,
		Digest:    ent.Digest,
		Fetched:   ent.Fetched,
		Refreshed: clock.Now(ctx).UTC(),
		settings:  withDefaultSettings(ent.Bundle.Settings),
		poolMap:   pools,
		poolNames: poolNames,
		botGroups: botGroups,
	}, nil
}
