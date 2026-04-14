// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/dataproc/v1"
)

func TestDetermineDataprocJobType(t *testing.T) {
	t.Run("hadoop job", func(t *testing.T) {
		job := &dataproc.Job{HadoopJob: &dataproc.HadoopJob{}}
		assert.Equal(t, "hadoop", determineDataprocJobType(job))
	})

	t.Run("spark job", func(t *testing.T) {
		job := &dataproc.Job{SparkJob: &dataproc.SparkJob{}}
		assert.Equal(t, "spark", determineDataprocJobType(job))
	})

	t.Run("pyspark job", func(t *testing.T) {
		job := &dataproc.Job{PysparkJob: &dataproc.PySparkJob{}}
		assert.Equal(t, "pyspark", determineDataprocJobType(job))
	})

	t.Run("hive job", func(t *testing.T) {
		job := &dataproc.Job{HiveJob: &dataproc.HiveJob{}}
		assert.Equal(t, "hive", determineDataprocJobType(job))
	})

	t.Run("pig job", func(t *testing.T) {
		job := &dataproc.Job{PigJob: &dataproc.PigJob{}}
		assert.Equal(t, "pig", determineDataprocJobType(job))
	})

	t.Run("sparkR job", func(t *testing.T) {
		job := &dataproc.Job{SparkRJob: &dataproc.SparkRJob{}}
		assert.Equal(t, "sparkR", determineDataprocJobType(job))
	})

	t.Run("sparkSql job", func(t *testing.T) {
		job := &dataproc.Job{SparkSqlJob: &dataproc.SparkSqlJob{}}
		assert.Equal(t, "sparkSql", determineDataprocJobType(job))
	})

	t.Run("presto job", func(t *testing.T) {
		job := &dataproc.Job{PrestoJob: &dataproc.PrestoJob{}}
		assert.Equal(t, "presto", determineDataprocJobType(job))
	})

	t.Run("flink job", func(t *testing.T) {
		job := &dataproc.Job{FlinkJob: &dataproc.FlinkJob{}}
		assert.Equal(t, "flink", determineDataprocJobType(job))
	})

	t.Run("trino job", func(t *testing.T) {
		job := &dataproc.Job{TrinoJob: &dataproc.TrinoJob{}}
		assert.Equal(t, "trino", determineDataprocJobType(job))
	})

	t.Run("unknown job type", func(t *testing.T) {
		job := &dataproc.Job{}
		assert.Equal(t, "unknown", determineDataprocJobType(job))
	})

	t.Run("first match wins when multiple set", func(t *testing.T) {
		job := &dataproc.Job{
			HadoopJob: &dataproc.HadoopJob{},
			SparkJob:  &dataproc.SparkJob{},
		}
		assert.Equal(t, "hadoop", determineDataprocJobType(job))
	})
}
