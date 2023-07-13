package awspolicy

// There are different policies that are parsed differently:
//
// - sqs queue policy
// - s3 bucket policy
// - vpc endpoint policy
// - iam policy
// - sns topic policy
//
// see aws policy generator https://awspolicygen.s3.amazonaws.com/policygen.html
//
// There are also issues for the go sdks to add types but not resolved since 2015:
//
// - https://github.com/aws/aws-sdk-go-v2/issues/225
// - https://github.com/aws/aws-sdk-go/issues/127
