// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"fmt"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ddsclient "github.com/alibabacloud-go/dds-20151201/v9/client"
	ecsclient "github.com/alibabacloud-go/ecs-20140526/v6/client"
	polardbclient "github.com/alibabacloud-go/polardb-20170801/v7/client"
	rkvclient "github.com/alibabacloud-go/r-kvstore-20150101/v6/client"
	ramclient "github.com/alibabacloud-go/ram-20150501/v2/client"
	rdsclient "github.com/alibabacloud-go/rds-20140815/v11/client"
	slbclient "github.com/alibabacloud-go/slb-20140515/v4/client"
	stsclient "github.com/alibabacloud-go/sts-20150401/v2/client"
	vpcclient "github.com/alibabacloud-go/vpc-20160428/v6/client"
	oss "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	osscred "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

// endpoint builds the public Alibaba Cloud service endpoint for a region. RAM is
// a global (region-less) service, so it always uses its central endpoint.
func endpoint(service, region string) string {
	if service == "ram" {
		return "ram.aliyuncs.com"
	}
	return fmt.Sprintf("%s.%s.aliyuncs.com", service, region)
}

// config assembles the shared OpenAPI config for a service client. Every
// Darabonba-generated client accepts *openapi.Config (an alias of
// models.Config), so one builder serves them all.
func (c *AlicloudConnection) config(service, region string) *openapi.Config {
	ep := endpoint(service, region)
	return &openapi.Config{
		Credential: c.cred,
		RegionId:   &region,
		Endpoint:   &ep,
	}
}

// cachedClient returns the client stored under key, or builds and stores it via
// build. Access is serialized so concurrent field resolution shares one client.
func (c *AlicloudConnection) cachedClient(key string, build func() (any, error)) (any, error) {
	c.clientLock.Lock()
	defer c.clientLock.Unlock()
	if client, ok := c.clients[key]; ok {
		return client, nil
	}
	client, err := build()
	if err != nil {
		return nil, err
	}
	c.clients[key] = client
	return client, nil
}

func (c *AlicloudConnection) EcsClient(region string) (*ecsclient.Client, error) {
	client, err := c.cachedClient("ecs/"+region, func() (any, error) {
		return ecsclient.NewClient(c.config("ecs", region))
	})
	if err != nil {
		return nil, err
	}
	return client.(*ecsclient.Client), nil
}

func (c *AlicloudConnection) VpcClient(region string) (*vpcclient.Client, error) {
	client, err := c.cachedClient("vpc/"+region, func() (any, error) {
		return vpcclient.NewClient(c.config("vpc", region))
	})
	if err != nil {
		return nil, err
	}
	return client.(*vpcclient.Client), nil
}

// RamClient returns the global RAM client. RAM has no regional endpoints, so the
// client is cached under a single key.
func (c *AlicloudConnection) RamClient() (*ramclient.Client, error) {
	client, err := c.cachedClient("ram", func() (any, error) {
		return ramclient.NewClient(c.config("ram", c.region))
	})
	if err != nil {
		return nil, err
	}
	return client.(*ramclient.Client), nil
}

func (c *AlicloudConnection) SlbClient(region string) (*slbclient.Client, error) {
	client, err := c.cachedClient("slb/"+region, func() (any, error) {
		return slbclient.NewClient(c.config("slb", region))
	})
	if err != nil {
		return nil, err
	}
	return client.(*slbclient.Client), nil
}

func (c *AlicloudConnection) StsClient(region string) (*stsclient.Client, error) {
	client, err := c.cachedClient("sts/"+region, func() (any, error) {
		return stsclient.NewClient(c.config("sts", region))
	})
	if err != nil {
		return nil, err
	}
	return client.(*stsclient.Client), nil
}

func (c *AlicloudConnection) RdsClient(region string) (*rdsclient.Client, error) {
	client, err := c.cachedClient("rds/"+region, func() (any, error) {
		return rdsclient.NewClient(c.config("rds", region))
	})
	if err != nil {
		return nil, err
	}
	return client.(*rdsclient.Client), nil
}

// RedisClient returns the ApsaraDB for Redis (Tair) client. The service's
// endpoint prefix is r-kvstore.
func (c *AlicloudConnection) RedisClient(region string) (*rkvclient.Client, error) {
	client, err := c.cachedClient("r-kvstore/"+region, func() (any, error) {
		return rkvclient.NewClient(c.config("r-kvstore", region))
	})
	if err != nil {
		return nil, err
	}
	return client.(*rkvclient.Client), nil
}

// MongoDBClient returns the ApsaraDB for MongoDB (dds) client. The service's
// endpoint prefix is mongodb.
func (c *AlicloudConnection) MongoDBClient(region string) (*ddsclient.Client, error) {
	client, err := c.cachedClient("mongodb/"+region, func() (any, error) {
		return ddsclient.NewClient(c.config("mongodb", region))
	})
	if err != nil {
		return nil, err
	}
	return client.(*ddsclient.Client), nil
}

func (c *AlicloudConnection) PolarDBClient(region string) (*polardbclient.Client, error) {
	client, err := c.cachedClient("polardb/"+region, func() (any, error) {
		return polardbclient.NewClient(c.config("polardb", region))
	})
	if err != nil {
		return nil, err
	}
	return client.(*polardbclient.Client), nil
}

// OssClient returns an Object Storage Service client for a region. OSS uses its
// own SDK and credential provider type, so it is built from the retained static
// credential (falling back to the OSS environment credential provider when no
// static credential was supplied).
func (c *AlicloudConnection) OssClient(region string) (*oss.Client, error) {
	client, err := c.cachedClient("oss/"+region, func() (any, error) {
		var provider osscred.CredentialsProvider
		if c.accessKeyID != "" && c.accessKeySecret != "" {
			provider = osscred.NewStaticCredentialsProvider(c.accessKeyID, c.accessKeySecret, c.securityToken)
		} else {
			provider = osscred.NewEnvironmentVariableCredentialsProvider()
		}
		cfg := oss.LoadDefaultConfig().
			WithRegion(region).
			WithCredentialsProvider(provider)
		return oss.NewClient(cfg), nil
	})
	if err != nil {
		return nil, err
	}
	return client.(*oss.Client), nil
}

// GetRegions returns the region IDs to scan. When the caller pinned a region
// filter, that list is returned verbatim; otherwise every region enabled on the
// account is enumerated via the ECS DescribeRegions API.
func (c *AlicloudConnection) GetRegions() ([]string, error) {
	if len(c.regionFilter) > 0 {
		return c.regionFilter, nil
	}

	client, err := c.EcsClient(c.region)
	if err != nil {
		return nil, err
	}

	resp, err := client.DescribeRegions(&ecsclient.DescribeRegionsRequest{})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Body == nil || resp.Body.Regions == nil {
		return nil, errors.New("alicloud: empty region list returned by DescribeRegions")
	}

	regions := make([]string, 0, len(resp.Body.Regions.Region))
	for _, r := range resp.Body.Regions.Region {
		if r != nil && r.RegionId != nil {
			regions = append(regions, *r.RegionId)
		}
	}
	return regions, nil
}

// Identify looks up the account (UID) the credential belongs to via the STS
// GetCallerIdentity API and caches it on the connection.
func (c *AlicloudConnection) Identify() (string, error) {
	if c.accountID != "" {
		return c.accountID, nil
	}

	client, err := c.StsClient(c.region)
	if err != nil {
		return "", err
	}

	resp, err := client.GetCallerIdentity()
	if err != nil {
		return "", err
	}
	if resp == nil || resp.Body == nil || resp.Body.AccountId == nil {
		return "", errors.New("alicloud: empty caller identity returned by STS")
	}

	c.accountID = *resp.Body.AccountId
	return c.accountID, nil
}
