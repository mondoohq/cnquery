package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.io/mondoo/motor/providers/aws"
	"go.mondoo.io/mondoo/resources/library/jobpool"
	"go.mondoo.io/mondoo/resources/packs/core"
)

func (c *mqlAwsConfig) id() (string, error) {
	return "aws.config", nil
}

func (c *mqlAwsConfig) GetRecorders() ([]interface{}, error) {
	provider, err := awsProvider(c.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(c.getRecorders(provider), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
	}
	return res, nil
}

func (c *mqlAwsConfig) getRecorders(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := provider.ConfigService(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			params := &configservice.DescribeConfigurationRecordersInput{}
			configRecorders, err := svc.DescribeConfigurationRecorders(ctx, params)
			if err != nil {
				return nil, err
			}
			recorderStatusMap, err := c.describeConfigRecorderStatus(svc, regionVal)
			if err != nil {
				return nil, err
			}
			for _, r := range configRecorders.ConfigurationRecorders {
				var recording bool
				var lastStatus string
				name := getName(core.ToString(r.Name), regionVal)
				if val, ok := recorderStatusMap[name]; ok {
					recording = val.recording
					lastStatus = val.lastStatus
				}
				mqlRecorder, err := c.MotorRuntime.CreateResource("aws.config.recorder",
					"name", core.ToString(r.Name),
					"roleArn", core.ToString(r.RoleARN),
					"allSupported", r.RecordingGroup.AllSupported,
					"includeGlobalResourceTypes", r.RecordingGroup.IncludeGlobalResourceTypes,
					"recording", recording,
					"region", regionVal,
					"lastStatus", lastStatus,
				)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlRecorder)
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func getName(name string, region string) string {
	return name + "/" + region
}

func (c *mqlAwsConfig) describeConfigRecorderStatus(svc *configservice.Client, regionVal string) (map[string]recorder, error) {
	statusMap := make(map[string]recorder)
	ctx := context.Background()

	params := &configservice.DescribeConfigurationRecorderStatusInput{}
	configRecorderStatus, err := svc.DescribeConfigurationRecorderStatus(ctx, params)
	if err != nil {
		return statusMap, err
	}
	for _, r := range configRecorderStatus.ConfigurationRecordersStatus {
		name := getName(core.ToString(r.Name), regionVal)
		statusMap[name] = recorder{recording: r.Recording, lastStatus: string(r.LastStatus)}
	}
	return statusMap, nil
}

type recorder struct {
	recording  bool
	lastStatus string
}

func (c *mqlAwsConfigRecorder) id() (string, error) {
	name, err := c.Name()
	if err != nil {
		return "", err
	}
	region, err := c.Region()
	if err != nil {
		return "", err
	}
	return getName(name, region), nil
}

func (c *mqlAwsConfig) GetRules() ([]interface{}, error) {
	provider, err := awsProvider(c.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(c.getRules(provider), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
	}
	return res, nil
}

func (c *mqlAwsConfig) getRules(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := provider.ConfigService(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			params := &configservice.DescribeConfigRulesInput{}
			rules, err := svc.DescribeConfigRules(ctx, params)
			if err != nil {
				return nil, err
			}
			for _, r := range rules.ConfigRules {
				jsonSource, err := core.JsonToDict(r.Source)
				if err != nil {
					return nil, err
				}
				mqlRule, err := c.MotorRuntime.CreateResource("aws.config.rule",
					"arn", core.ToString(r.ConfigRuleArn),
					"state", string(r.ConfigRuleState),
					"source", jsonSource,
				)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlRule)
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (c *mqlAwsConfigRule) id() (string, error) {
	return c.Arn()
}
