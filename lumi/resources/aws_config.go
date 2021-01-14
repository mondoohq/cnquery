package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
)

func (c *lumiAwsConfig) id() (string, error) {
	return "aws.config", nil
}

func (c *lumiAwsConfig) GetRecorders() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(c.getRecorders(), 5)
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

func (c *lumiAwsConfig) getRecorders() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(c.Runtime.Motor.Transport)
	if err != nil {
		return []*jobpool.Job{&jobpool.Job{Err: err}}
	}
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{&jobpool.Job{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := at.ConfigService(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			params := &configservice.DescribeConfigurationRecordersInput{}
			configRecorders, err := svc.DescribeConfigurationRecordersRequest(params).Send(ctx)
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
				name := getName(toString(r.Name), regionVal)
				if val, ok := recorderStatusMap[name]; ok {
					recording = val.recording
					lastStatus = val.lastStatus
				}
				lumiRecorder, err := c.Runtime.CreateResource("aws.config.recorder",
					"name", toString(r.Name),
					"roleArn", toString(r.RoleARN),
					"allSupported", toBool(r.RecordingGroup.AllSupported),
					"includeGlobalResourceTypes", toBool(r.RecordingGroup.IncludeGlobalResourceTypes),
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
	configRecorderStatus, err := svc.DescribeConfigurationRecorderStatusRequest(params).Send(ctx)
	if err != nil {
		return statusMap, err
	}
	for _, r := range configRecorderStatus.ConfigurationRecordersStatus {
		stringState, err := configservice.RecorderStatus.MarshalValue(r.LastStatus)
		if err != nil {
			return statusMap, err
		}
		name := getName(toString(r.Name), regionVal)
		statusMap[name] = recorder{recording: toBool(r.Recording), lastStatus: stringState}
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
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(c.getRules(), 5)
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

func (c *lumiAwsConfig) getRules() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(c.Runtime.Motor.Transport)
	if err != nil {
		return []*jobpool.Job{&jobpool.Job{Err: err}}
	}
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{&jobpool.Job{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := at.ConfigService(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			params := &configservice.DescribeConfigRulesInput{}
			rules, err := svc.DescribeConfigRulesRequest(params).Send(ctx)
			if err != nil {
				return nil, err
			}
			for _, r := range rules.ConfigRules {
				stringState, err := r.ConfigRuleState.MarshalValue()
				if err != nil {
					return nil, err
				}
				jsonSource, err := jsonToDict(r.Source)
				if err != nil {
					return nil, err
				}
				lumiRule, err := c.Runtime.CreateResource("aws.config.rule",
					"arn", toString(r.ConfigRuleArn),
					"state", stringState,
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
