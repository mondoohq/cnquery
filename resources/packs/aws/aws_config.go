package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
	aws_transport "go.mondoo.io/mondoo/motor/providers/aws"
	"go.mondoo.io/mondoo/resources/packs/core"
)

func (c *lumiAwsConfig) id() (string, error) {
	return "aws.config", nil
}

func (c *lumiAwsConfig) GetRecorders() ([]interface{}, error) {
	at, err := awstransport(c.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(c.getRecorders(at), 5)
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

func (c *lumiAwsConfig) getRecorders(at *aws_transport.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := at.ConfigService(regionVal)
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
				lumiRecorder, err := c.MotorRuntime.CreateResource("aws.config.recorder",
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
				res = append(res, lumiRecorder)
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

func (c *lumiAwsConfig) describeConfigRecorderStatus(svc *configservice.Client, regionVal string) (map[string]recorder, error) {
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

func (c *lumiAwsConfigRecorder) id() (string, error) {
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

func (c *lumiAwsConfig) GetRules() ([]interface{}, error) {
	at, err := awstransport(c.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(c.getRules(at), 5)
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

func (c *lumiAwsConfig) getRules(at *aws_transport.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := at.ConfigService(regionVal)
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
				lumiRule, err := c.MotorRuntime.CreateResource("aws.config.rule",
					"arn", core.ToString(r.ConfigRuleArn),
					"state", string(r.ConfigRuleState),
					"source", jsonSource,
				)
				if err != nil {
					return nil, err
				}
				res = append(res, lumiRule)
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (c *lumiAwsConfigRule) id() (string, error) {
	return c.Arn()
}
