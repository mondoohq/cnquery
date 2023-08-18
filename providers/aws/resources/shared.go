package resources

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go/transport/http"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/aws/connection"
	"go.mondoo.com/cnquery/providers/network/resources/certificates"
	"go.mondoo.com/cnquery/types"
	"k8s.io/client-go/util/cert"
)

const (
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
	s3ArnPattern                = "arn:aws:s3:::%s"
	dynamoTableArnPattern       = "arn:aws:dynamodb:%s:%s:table/%s"
	limitsArn                   = "arn:aws:dynamodb:%s:%s"
	dynamoGlobalTableArnPattern = "arn:aws:dynamodb:-:%s:globaltable/%s"
)

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
		if respErr.HTTPStatusCode() == 400 && strings.Contains(respErr.Error(), "AccessDeniedException") {
			return true
		}
	}
	return false
}

func Ec2TagsToMap(tags []ec2types.Tag) map[string]interface{} {
	tagsMap := make(map[string]interface{})

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[toString(tag.Key)] = toString(tag.Value)
		}
	}

	return tagsMap
}

func toBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func toString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func toTime(s *time.Time) time.Time {
	if s == nil {
		return time.Time{}
	}
	return *s
}

func strMapToInterface(m map[string]string) map[string]interface{} {
	res := map[string]interface{}{}
	for k, v := range m {
		res[k] = v
	}
	return res
}

func toInt64From32(i *int32) int64 {
	if i == nil {
		return int64(0)
	}
	return int64(*i)
}

func toFloat64(i *float64) float64 {
	if i == nil {
		return float64(0)
	}
	return *i
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
