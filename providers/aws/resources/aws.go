// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/smithy-go/transport/http"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
	"go.mondoo.com/cnquery/v11/providers/network/resources/certificates"
	"go.mondoo.com/cnquery/v11/types"
	"k8s.io/client-go/util/cert"
)

const (
	vpcArnPattern               = "arn:aws:vpc:%s:%s:id/%s"
	elbv1LbArnPattern           = "arn:aws:elasticloadbalancing:%s:%s:loadbalancer/classic/%s"
	cloudwatchAlarmArnPattern   = "arn:aws:cloudwatch:%s:%s:metricalarm/%s/%s"
	ec2InstanceArnPattern       = "arn:aws:ec2:%s:%s:instance/%s"
	securityGroupArnPattern     = "arn:aws:ec2:%s:%s:security-group/%s"
	volumeArnPattern            = "arn:aws:ec2:%s:%s:volume/%s"
	snapshotArnPattern          = "arn:aws:ec2:%s:%s:snapshot/%s"
	internetGwArnPattern        = "arn:aws:ec2:%s:%s:gateway/%s"
	vpnConnArnPattern           = "arn:aws:ec2:%s:%s:vpn-connection/%s"
	networkAclArnPattern        = "arn:aws:ec2:%s:%s:network-acl/%s"
	imageArnPattern             = "arn:aws:ec2:%s:%s:image/%s"
	keypairArnPattern           = "arn:aws:ec2:%s:%s:keypair/%s"
	subnetArnPattern            = "arn:aws:ec2:%s:%s:subnet/%s"
	s3ArnPattern                = "arn:aws:s3:::%s"
	dynamoTableArnPattern       = "arn:aws:dynamodb:%s:%s:table/%s"
	limitsArn                   = "arn:aws:dynamodb:%s:%s"
	dynamoGlobalTableArnPattern = "arn:aws:dynamodb:-:%s:globaltable/%s"
	rdsInstanceArnPattern       = "arn:aws:rds:%s:%s:db:%s"
	apiArnPattern               = "arn:aws:apigateway:%s:%s::/apis/%s"
	apiStageArnPattern          = "arn:aws:apigateway:%s:%s::/apis/%s/stages/%s"
)

func NewSecurityGroupArn(region, accountID, sgID string) string {
	return fmt.Sprintf(securityGroupArnPattern, region, accountID, sgID)
}

func (a *mqlAws) regions() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	regions, err := conn.Regions()
	for i := range regions {
		res = append(res, regions[i])
	}
	return res, err
}

func Is400AccessDeniedError(err error) bool {
	var respErr *http.ResponseError
	if errors.As(err, &respErr) {
		if (respErr.HTTPStatusCode() == 400 || respErr.HTTPStatusCode() == 403) && (strings.Contains(respErr.Error(), "AccessDenied") || strings.Contains(respErr.Error(), "UnauthorizedOperation") || strings.Contains(respErr.Error(), "AuthorizationError")) {
			return true
		}
	}
	return false
}

func strMapToInterface(m map[string]string) map[string]interface{} {
	res := map[string]interface{}{}
	for k, v := range m {
		res[k] = v
	}
	return res
}

func toInterfaceArr(a []string) []interface{} {
	res := []interface{}{}
	for i := range a {
		res = append(res, a[i])
	}
	return res
}

func GetRegionFromArn(arnVal string) (string, error) {
	parsedArn, err := arn.Parse(arnVal)
	if err != nil {
		return "", err
	}
	return parsedArn.Region, nil
}

func CertificatesToMqlCertificates(runtime *plugin.Runtime, certs []*x509.Certificate) ([]interface{}, error) {
	res := []interface{}{}
	// to create certificate resources
	for i := range certs {
		cert := certs[i]

		if cert == nil {
			continue
		}

		certdata, err := certificates.EncodeCertAsPEM(cert)
		if err != nil {
			return nil, err
		}

		r, err := CreateResource(runtime, "certificate", map[string]*llx.RawData{
			"pem": llx.StringData(string(certdata)),
			// NOTE: if we do not set the hash here, it will generate the cache content before we can store it
			// we are using the hashes for the id, therefore it is required during creation
			"fingerprints": llx.MapData(certificates.Fingerprints(cert), types.String),
		})
		if err != nil {
			return nil, err
		}

		c := r.(*mqlAwsAcmCertificate)
		// c.Certificate = plugin.TValue[*x509.Certificate]{
		// 	Pem:   llx.StringData(cert.Pem),
		// 	Data:  cert,
		// 	State: plugin.StateIsSet,
		// } // TODO: revisit all this cert stuff. can we share resources across providers??

		res = append(res, c)
	}
	return res, nil
}

func ParseCertsFromPEM(r io.Reader) ([]*x509.Certificate, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	certs, err := cert.ParseCertsPEM(data)
	if err != nil {
		return nil, err
	}

	return certs, nil
}

func EncodeCertAsPEM(cert *x509.Certificate) ([]byte, error) {
	certBuffer := bytes.Buffer{}
	if err := pem.Encode(&certBuffer, &pem.Block{Type: CertificateBlockType, Bytes: cert.Raw}); err != nil {
		return nil, err
	}
	return certBuffer.Bytes(), nil
}

const (
	// CertificateBlockType is a possible value for pem.Block.Type.
	CertificateBlockType = "CERTIFICATE"
)

type assetIdentifier struct {
	name string
	arn  string
}

func getAssetIdentifier(runtime *plugin.Runtime) *assetIdentifier {
	var a *inventory.Asset
	if conn, ok := runtime.Connection.(*connection.AwsConnection); ok {
		a = conn.Asset()
	}
	if a == nil {
		return nil
	}
	arn := ""
	for _, id := range a.PlatformIds {
		if strings.HasPrefix(id, "arn:aws:") {
			arn = id
		}
	}
	return &assetIdentifier{name: a.Name, arn: arn}
}

func mapStringInterfaceToStringString(m map[string]interface{}) map[string]string {
	newM := make(map[string]string)
	for k, v := range m {
		newM[k] = v.(string)
	}
	return newM
}

func remove(s []string, r string) []string {
	for i, v := range s {
		if v == r {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

// securityGroupIdHandler is a helper struct to handle security group ids and convert them to resources
// This makes it easy to extend the internal representation of a resource and fetch security groups asynchronous
type securityGroupIdHandler struct {
	securityGroupArns []string
}

// setSecurityGroupArns sets the security group arns
func (sgh *securityGroupIdHandler) setSecurityGroupArns(ids []string) {
	sgh.securityGroupArns = ids
}

// newSecurityGroupResources creates new security group resources based on the security group arns
func (sgh *securityGroupIdHandler) newSecurityGroupResources(runtime *plugin.Runtime) ([]interface{}, error) {
	sgs := []interface{}{}
	for i := range sgh.securityGroupArns {
		sgArn := sgh.securityGroupArns[i]
		mqlSg, err := NewResource(runtime, "aws.ec2.securitygroup",
			map[string]*llx.RawData{
				"arn": llx.StringData(sgArn),
			})
		if err != nil {
			return nil, err
		}
		sgs = append(sgs, mqlSg)
	}
	return sgs, nil
}
