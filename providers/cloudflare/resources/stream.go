// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
)

// metaString returns the string value at key in a Cloudflare `meta` map, or ""
// if the key is missing or the value is not a string. The Cloudflare stream
// API allows arbitrary user-set keys here.
func metaString(meta map[string]any, key string) string {
	if meta == nil {
		return ""
	}
	v, ok := meta[key].(string)
	if !ok {
		return ""
	}
	return v
}

func (c *mqlCloudflareStreamsLiveInput) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareStreamsVideo) id() (string, error) {
	if c.Id.Error != nil {
		return "", c.Id.Error
	}
	return c.Id.Data, nil
}

func (c *mqlCloudflareZone) liveInputs() ([]any, error) {
	return fetchLiveInputs(c.MqlRuntime, c.Account.Data.GetId().Data)
}

func (c *mqlCloudflareAccount) liveInputs() ([]any, error) {
	return fetchLiveInputs(c.MqlRuntime, c.Id.Data)
}

func fetchLiveInputs(runtime *plugin.Runtime, account_id string) ([]any, error) {
	conn := runtime.Connection.(*connection.CloudflareConnection)

	raw, err := conn.Cf.Raw(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("/accounts/%s/stream/live_inputs", account_id),
		nil,
		nil,
	)
	if err != nil {
		return nil, err
	}

	var results []struct {
		Uid                      string         `json:"uid"`
		Modified                 string         `json:"modified"`
		Created                  string         `json:"created"`
		DeleteRecordingAfterDays int            `json:"deleteRecordingAfterDays"`
		Meta                     map[string]any `json:"meta"`
	}
	if err := json.Unmarshal(raw.Result, &results); err != nil {
		return nil, fmt.Errorf("cloudflare stream live_inputs: unmarshal: %w", err)
	}

	var res []any
	for _, result := range results {
		input, err := NewResource(runtime, "cloudflare.streams.liveInput", map[string]*llx.RawData{
			"id":                       llx.StringData(result.Uid),
			"uid":                      llx.StringData(result.Uid),
			"deleteRecordingAfterDays": llx.IntData(result.DeleteRecordingAfterDays),
			"name":                     llx.StringData(metaString(result.Meta, "name")),
		})
		if err != nil {
			return nil, err
		}

		res = append(res, input)
	}

	return res, nil
}

func (c *mqlCloudflareZone) videos() ([]any, error) {
	return fetchVideos(c.MqlRuntime, c.Account.Data.GetId().Data)
}

func (c *mqlCloudflareAccount) videos() ([]any, error) {
	return fetchVideos(c.MqlRuntime, c.Id.Data)
}

func fetchVideos(runtime *plugin.Runtime, account_id string) ([]any, error) {
	conn := runtime.Connection.(*connection.CloudflareConnection)

	results, err := conn.Cf.StreamListVideos(context.Background(), cloudflare.StreamListParameters{
		AccountID: account_id,
	})
	if err != nil {
		return nil, err
	}

	var result []any
	for i := range results {
		video := results[i]

		res, err := NewResource(runtime, "cloudflare.streams.video", map[string]*llx.RawData{
			"id":                    llx.StringData(video.UID),
			"uid":                   llx.StringData(video.UID),
			"name":                  llx.StringData(metaString(video.Meta, "name")),
			"creator":               llx.StringData(video.Creator),
			"duration":              llx.FloatData(video.Duration),
			"height":                llx.IntData(video.Input.Height),
			"width":                 llx.IntData(video.Input.Width),
			"liveInput":             llx.StringData(video.LiveInput),
			"dash":                  llx.StringData(video.Playback.Dash),
			"hls":                   llx.StringData(video.Playback.HLS),
			"preview":               llx.StringData(video.Preview),
			"ready":                 llx.BoolData(video.ReadyToStream),
			"requireSignedUrls":     llx.BoolData(video.RequireSignedURLs),
			"scheduledDeletion":     llx.TimeDataPtr(video.ScheduledDeletion),
			"size":                  llx.IntData(video.Size),
			"thumbnail":             llx.StringData(video.Thumbnail),
			"thumbnailTimestampPct": llx.FloatData(video.ThumbnailTimestampPct),
			"uploaded":              llx.TimeDataPtr(video.Uploaded),
		})
		if err != nil {
			return nil, err
		}

		result = append(result, res)
	}

	return result, nil
}
