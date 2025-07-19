package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"k8s.io/utils/ptr"
)

type AwsKMS struct {
	//"arn:aws:iam::095352988152:role/e2e-example-cluster"
	tenantRoleARN string
	// "arn:aws:kms:eu-north-1:142755254688:key/22de90f5-8345-42b7-9916-5119ef7afbcc"
	keyARN          string
	roleSessionName string
	token           TokenProvider

	// region allows to hardcode the aws region when running locally
	region string
}

type TokenProvider interface {
	GetToken(ctx context.Context, targetNamespace, k8sServiceAccountName, audience string) (string, error)
}

// Option is a function that configures an AwsKMS instance
type Option func(*AwsKMS)

// WithTenantRoleARN sets the tenant role ARN
// Example: "arn:aws:iam::095352988152:role/e2e-example-cluster"
func WithTenantRoleARN(tenantRoleARN string) Option {
	return func(a *AwsKMS) {
		a.tenantRoleARN = tenantRoleARN
	}
}

// WithKeyARN sets the key ARN
func WithKeyARN(keyARN string) Option {
	return func(a *AwsKMS) {
		a.keyARN = keyARN
	}
}

// WithRoleSessionName sets the role session name
func WithRoleSessionName(roleSessionName string) Option {
	return func(a *AwsKMS) {
		a.roleSessionName = roleSessionName
	}
}

// WithTokenProvider sets the token provider
func WithTokenProvider(token TokenProvider) Option {
	return func(a *AwsKMS) {
		a.token = token
	}
}

// WithRegion sets the AWS region
// useful for local development or testing where you can't autodetect the region
func WithRegion(region string) Option {
	return func(a *AwsKMS) {
		a.region = region
	}
}

func WithDefaultRoleSessionName(name string) Option {
	return func(a *AwsKMS) {
		a.roleSessionName = name
	}
}

// NewAwsKMS creates a new AwsKMS instance with the provided options
// Required options:
// - WithTenantRoleARN
// - WithKeyARN
// - WithTokenProvider
func NewAwsKMS(opts ...Option) AwsKMS {
	// Create a default instance
	awsKMS := AwsKMS{
		roleSessionName: "default-session",
	}

	// Apply all options
	for _, opt := range opts {
		opt(&awsKMS)
	}

	// Validate required fields
	if awsKMS.tenantRoleARN == "" {
		panic("tenantRoleARN is required, use WithTenantRoleARN option")
	}

	if awsKMS.keyARN == "" {
		panic("keyARN is required, use WithKeyARN option")
	}

	if awsKMS.token == nil {
		panic("token provider is required, use WithTokenProvider option")
	}

	return awsKMS
}

func (a AwsKMS) DescribeKey(ctx context.Context, targetNamespace, k8sServiceAccountName string) (*kms.DescribeKeyOutput, error) {
	webIdentityToken, err := a.token.GetToken(ctx, targetNamespace, k8sServiceAccountName, "sts.amazonaws.com")
	if err != nil {
		return nil, fmt.Errorf("failed to get web identity token: %w", err)
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %w", err)
	}

	stsClient := sts.NewFromConfig(cfg, func(options *sts.Options) {
		if a.region != "" {
			options.Region = a.region
		}
	})
	assumeRoleOutput, err := stsClient.AssumeRoleWithWebIdentity(context.Background(), &sts.AssumeRoleWithWebIdentityInput{
		RoleArn:          ptr.To(a.tenantRoleARN),
		RoleSessionName:  ptr.To(a.roleSessionName),
		WebIdentityToken: aws.String(webIdentityToken),
		DurationSeconds:  aws.Int32(900),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to assume role with web identity: %w", err)
	}
	creds := credentials.StaticCredentialsProvider{
		Value: aws.Credentials{
			AccessKeyID:     *assumeRoleOutput.Credentials.AccessKeyId,
			SecretAccessKey: *assumeRoleOutput.Credentials.SecretAccessKey,
			SessionToken:    *assumeRoleOutput.Credentials.SessionToken,
		},
	}

	kmsClient := kms.NewFromConfig(cfg, func(options *kms.Options) {
		options.Credentials = creds
		if a.region != "" {
			options.Region = a.region
		}
	})

	key, err := kmsClient.DescribeKey(context.Background(), &kms.DescribeKeyInput{KeyId: ptr.To(a.keyARN)})
	if err != nil {
		return nil, fmt.Errorf("failed to describe key: %w", err)
	}
	return key, nil
}
