// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

type mqlLuksVolumeInternal struct {
	dump     luksDump
	parsed   bool
	blockDev *mqlLsblkEntry
}

func (l *mqlLuks) id() (string, error) {
	return "luks", nil
}

func (l *mqlLuks) volumes() ([]any, error) {
	lsblkRes, err := CreateResource(l.MqlRuntime, "lsblk", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	list := lsblkRes.(*mqlLsblk).GetList()
	if list.Error != nil {
		return nil, list.Error
	}

	volumes := []any{}
	for _, raw := range list.Data {
		entry := raw.(*mqlLsblkEntry)
		if !isLuksFstype(entry.Fstype.Data) {
			continue
		}

		dump, err := runLuksDump(l.MqlRuntime, entry.Name.Data)
		if err != nil {
			log.Debug().Err(err).Str("device", entry.Name.Data).Msg("luks: skipping device")
			continue
		}

		vol, err := newLuksVolume(l.MqlRuntime, entry, dump)
		if err != nil {
			return nil, err
		}
		volumes = append(volumes, vol)
	}
	return volumes, nil
}

func runLuksDump(runtime *plugin.Runtime, device string) (luksDump, error) {
	o, err := CreateResource(runtime, "command", map[string]*llx.RawData{
		"command": llx.StringData("cryptsetup luksDump " + device),
	})
	if err != nil {
		return luksDump{}, err
	}
	cmd := o.(*mqlCommand)
	if exit := cmd.GetExitcode(); exit.Data != 0 {
		return luksDump{}, errors.New("cryptsetup luksDump failed: " + cmd.Stderr.Data)
	}
	return parseLuksDump(cmd.Stdout.Data)
}

func newLuksVolume(runtime *plugin.Runtime, entry *mqlLsblkEntry, dump luksDump) (*mqlLuksVolume, error) {
	res, err := CreateResource(runtime, "luks.volume", map[string]*llx.RawData{
		"name":          llx.StringData(entry.Name.Data),
		"uuid":          llx.StringData(dump.UUID),
		"version":       llx.IntData(int64(dump.Version)),
		"label":         llx.StringData(dump.Label),
		"subsystem":     llx.StringData(dump.Subsystem),
		"masterKeyBits": llx.IntData(int64(dump.MasterKeyBits)),
		"payloadOffset": llx.IntData(int64(dump.PayloadOffset)),
		"tokens":        llx.ArrayData(luksTokensToDicts(dump.Tokens), types.Dict),
	})
	if err != nil {
		return nil, err
	}
	vol := res.(*mqlLuksVolume)
	vol.dump = dump
	vol.parsed = true
	vol.blockDev = entry
	return vol, nil
}

// luksTokensToDicts converts parsed token maps to llx-compatible dict
// values. The parser stores keyslot indices as []int64; llx dicts
// expect []any for nested arrays.
func luksTokensToDicts(tokens []map[string]any) []any {
	out := make([]any, 0, len(tokens))
	for _, t := range tokens {
		normalized := make(map[string]any, len(t))
		for k, v := range t {
			if slice, ok := v.([]int64); ok {
				asAny := make([]any, len(slice))
				for i, n := range slice {
					asAny[i] = n
				}
				normalized[k] = asAny
				continue
			}
			normalized[k] = v
		}
		out = append(out, normalized)
	}
	return out
}

func (v *mqlLuksVolume) id() (string, error) {
	return v.Uuid.Data, nil
}

func (v *mqlLuksVolume) blockDevice() (*mqlLsblkEntry, error) {
	if v.blockDev != nil {
		return v.blockDev, nil
	}
	v.BlockDevice.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (v *mqlLuksVolume) cipher() (*mqlLuksVolumeCipher, error) {
	if !v.parsed {
		return nil, errors.New("luks volume dump not available")
	}
	c := v.dump.Cipher
	res, err := CreateResource(v.MqlRuntime, "luks.volume.cipher", map[string]*llx.RawData{
		"__id":    llx.StringData(v.Uuid.Data + "/cipher"),
		"name":    llx.StringData(c.Name),
		"mode":    llx.StringData(c.Mode),
		"spec":    llx.StringData(c.Spec),
		"keySize": llx.IntData(int64(c.KeySize)),
		"hash":    llx.StringData(c.Hash),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlLuksVolumeCipher), nil
}

func (v *mqlLuksVolume) keyslots() ([]any, error) {
	if !v.parsed {
		return nil, errors.New("luks volume dump not available")
	}
	out := make([]any, 0, len(v.dump.Keyslots))
	for _, k := range v.dump.Keyslots {
		res, err := CreateResource(v.MqlRuntime, "luks.keyslot", map[string]*llx.RawData{
			"__id":              llx.StringData(fmt.Sprintf("%s/keyslot/%d", v.Uuid.Data, k.Index)),
			"index":             llx.IntData(int64(k.Index)),
			"state":             llx.StringData(k.State),
			"kdf":               llx.StringData(k.KDF),
			"iterations":        llx.IntData(int64(k.Iterations)),
			"time":              llx.IntData(int64(k.Time)),
			"memory":            llx.IntData(int64(k.Memory)),
			"parallel":          llx.IntData(int64(k.Parallel)),
			"hash":              llx.StringData(k.Hash),
			"stripes":           llx.IntData(int64(k.Stripes)),
			"keyMaterialOffset": llx.IntData(int64(k.KeyMaterialOffset)),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}
