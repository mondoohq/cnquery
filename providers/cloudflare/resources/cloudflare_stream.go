// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/cloudflare/connection"
)

func (c *mqlCloudflareStreamsLive_input) id() (string, error) {
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

func (c *mqlCloudflareZone) live_inputs() ([]any, error) {
	return fetchLiveInputs(c.MqlRuntime, c.Account.Data.GetId().Data)
}

func (c *mqlCloudflareAccount) live_inputs() ([]any, error) {
	return fetchLiveInputs(c.MqlRuntime, c.Id.Data)
}

func fetchLiveInputs(runtime *plugin.Runtime, account_id string) ([]any, error) {
	conn := runtime.Connection.(*connection.CloudflareConnection)

	req, _ := http.NewRequest(
		"GET", fmt.Sprintf("%s/accounts/%s/stream/live_inputs", conn.Cf.BaseURL, account_id), nil,
	)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", conn.Cf.APIToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var results struct {
		Result []struct {
			Uid                      string                 `json:"uid"`
			Modified                 string                 `json:"modified"`
			Created                  string                 `json:"created"`
			DeleteRecordingAfterDays int                    `json:"deleteRecordingAfterDays"`
			Meta                     map[string]interface{} `json:"meta"`
		} `json:"result"`
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bytes, &results); err != nil {
		return nil, err
	}

	var res []any

	for _, result := range results.Result {
		input, err := NewResource(runtime, "cloudflare.streams.live_input", map[string]*llx.RawData{
			"id":                          llx.StringData(result.Uid),
			"uid":                         llx.StringData(result.Uid),
			"delete_recording_after_days": llx.IntData(result.DeleteRecordingAfterDays),
			"name":                        llx.StringData(result.Meta["name"].(string)),
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
			"id":                      llx.StringData(video.UID),
			"uid":                     llx.StringData(video.UID),
			"name":                    llx.StringData(video.Meta["name"].(string)),
			"creator":                 llx.StringData(video.Creator),
			"duration":                llx.FloatData(video.Duration),
			"height":                  llx.IntData(video.Input.Height),
			"width":                   llx.IntData(video.Input.Width),
			"live_input":              llx.StringData(video.LiveInput),
			"dash":                    llx.StringData(video.Playback.Dash),
			"hls":                     llx.StringData(video.Playback.HLS),
			"preview":                 llx.StringData(video.Preview),
			"ready":                   llx.BoolData(video.ReadyToStream),
			"require_signed_urls":     llx.BoolData(video.RequireSignedURLs),
			"scheduled_deletion":      llx.TimeDataPtr(video.ScheduledDeletion),
			"size":                    llx.IntData(video.Size),
			"thumbnail":               llx.StringData(video.Thumbnail),
			"thumbnail_timestamp_pct": llx.FloatData(video.ThumbnailTimestampPct),
			"uploaded":                llx.TimeDataPtr(video.Uploaded),
		})
		if err != nil {
			return nil, err
		}

		result = append(result, res)
	}

	return result, nil
}
