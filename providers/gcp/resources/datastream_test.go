// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"cloud.google.com/go/datastream/apiv1/datastreampb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatastreamBackfillStrategy(t *testing.T) {
	t.Run("nil stream", func(t *testing.T) {
		assert.Equal(t, "", datastreamBackfillStrategy(nil))
	})

	t.Run("strategy unset", func(t *testing.T) {
		assert.Equal(t, "", datastreamBackfillStrategy(&datastreampb.Stream{}))
	})

	t.Run("backfill all", func(t *testing.T) {
		s := &datastreampb.Stream{
			BackfillStrategy: &datastreampb.Stream_BackfillAll{
				BackfillAll: &datastreampb.Stream_BackfillAllStrategy{},
			},
		}
		assert.Equal(t, "all", datastreamBackfillStrategy(s))
	})

	t.Run("backfill none", func(t *testing.T) {
		s := &datastreampb.Stream{
			BackfillStrategy: &datastreampb.Stream_BackfillNone{
				BackfillNone: &datastreampb.Stream_BackfillNoneStrategy{},
			},
		}
		assert.Equal(t, "none", datastreamBackfillStrategy(s))
	})
}

func TestDatastreamProfileToDict(t *testing.T) {
	tests := []struct {
		name           string
		cp             *datastreampb.ConnectionProfile
		wantType       string
		wantBucket     string
		wantDictNotNil bool
	}{
		{
			name:     "no profile set",
			cp:       &datastreampb.ConnectionProfile{},
			wantType: "",
		},
		{
			name: "oracle",
			cp: &datastreampb.ConnectionProfile{
				Profile: &datastreampb.ConnectionProfile_OracleProfile{
					OracleProfile: &datastreampb.OracleProfile{Hostname: "h", Port: 1521, Username: "u"},
				},
			},
			wantType:       "oracle",
			wantDictNotNil: true,
		},
		{
			name: "mysql",
			cp: &datastreampb.ConnectionProfile{
				Profile: &datastreampb.ConnectionProfile_MysqlProfile{
					MysqlProfile: &datastreampb.MysqlProfile{Hostname: "h", Port: 3306, Username: "u"},
				},
			},
			wantType:       "mysql",
			wantDictNotNil: true,
		},
		{
			name: "postgresql",
			cp: &datastreampb.ConnectionProfile{
				Profile: &datastreampb.ConnectionProfile_PostgresqlProfile{
					PostgresqlProfile: &datastreampb.PostgresqlProfile{Hostname: "h", Port: 5432, Username: "u"},
				},
			},
			wantType:       "postgresql",
			wantDictNotNil: true,
		},
		{
			name: "sqlserver",
			cp: &datastreampb.ConnectionProfile{
				Profile: &datastreampb.ConnectionProfile_SqlServerProfile{
					SqlServerProfile: &datastreampb.SqlServerProfile{Hostname: "h", Port: 1433, Username: "u"},
				},
			},
			wantType:       "sqlserver",
			wantDictNotNil: true,
		},
		{
			name: "mongodb",
			cp: &datastreampb.ConnectionProfile{
				Profile: &datastreampb.ConnectionProfile_MongodbProfile{
					MongodbProfile: &datastreampb.MongodbProfile{Username: "u"},
				},
			},
			wantType:       "mongodb",
			wantDictNotNil: true,
		},
		{
			name: "bigquery",
			cp: &datastreampb.ConnectionProfile{
				Profile: &datastreampb.ConnectionProfile_BigqueryProfile{
					BigqueryProfile: &datastreampb.BigQueryProfile{},
				},
			},
			wantType: "bigquery",
		},
		{
			name: "gcs surfaces bucket",
			cp: &datastreampb.ConnectionProfile{
				Profile: &datastreampb.ConnectionProfile_GcsProfile{
					GcsProfile: &datastreampb.GcsProfile{Bucket: "my-bucket", RootPath: "/r"},
				},
			},
			wantType:       "gcs",
			wantBucket:     "my-bucket",
			wantDictNotNil: true,
		},
		{
			name: "salesforce",
			cp: &datastreampb.ConnectionProfile{
				Profile: &datastreampb.ConnectionProfile_SalesforceProfile{
					SalesforceProfile: &datastreampb.SalesforceProfile{Domain: "d"},
				},
			},
			wantType:       "salesforce",
			wantDictNotNil: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotType, gotDict, gotBucket, err := datastreamProfileToDict(tc.cp)
			require.NoError(t, err)
			assert.Equal(t, tc.wantType, gotType)
			assert.Equal(t, tc.wantBucket, gotBucket)
			if tc.wantDictNotNil {
				assert.NotNil(t, gotDict)
			}
		})
	}
}

func TestDatastreamConnectivityToDict(t *testing.T) {
	tests := []struct {
		name         string
		cp           *datastreampb.ConnectionProfile
		wantType     string
		wantPrivConn string
	}{
		{
			name:     "no connectivity set",
			cp:       &datastreampb.ConnectionProfile{},
			wantType: "",
		},
		{
			name: "static service IP",
			cp: &datastreampb.ConnectionProfile{
				Connectivity: &datastreampb.ConnectionProfile_StaticServiceIpConnectivity{
					StaticServiceIpConnectivity: &datastreampb.StaticServiceIpConnectivity{},
				},
			},
			wantType: "staticServiceIp",
		},
		{
			name: "forward SSH",
			cp: &datastreampb.ConnectionProfile{
				Connectivity: &datastreampb.ConnectionProfile_ForwardSshConnectivity{
					ForwardSshConnectivity: &datastreampb.ForwardSshTunnelConnectivity{
						Hostname: "h", Username: "u", Port: 22,
					},
				},
			},
			wantType: "forwardSsh",
		},
		{
			name: "private connectivity surfaces private connection name",
			cp: &datastreampb.ConnectionProfile{
				Connectivity: &datastreampb.ConnectionProfile_PrivateConnectivity{
					PrivateConnectivity: &datastreampb.PrivateConnectivity{
						PrivateConnection: "projects/p/locations/l/privateConnections/pc",
					},
				},
			},
			wantType:     "privateConnectivity",
			wantPrivConn: "projects/p/locations/l/privateConnections/pc",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotType, _, gotPriv, err := datastreamConnectivityToDict(tc.cp)
			require.NoError(t, err)
			assert.Equal(t, tc.wantType, gotType)
			assert.Equal(t, tc.wantPrivConn, gotPriv)
		})
	}
}
