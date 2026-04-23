package main

import (
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfront"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfrontorigins"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3deployment"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

const domainName = "codyolsen.com"

type InfraStackProps struct {
	awscdk.StackProps
}

func NewInfraStack(scope constructs.Construct, id string, props *InfraStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	bucket := awss3.NewBucket(stack, jsii.String("SiteBucket"), &awss3.BucketProps{
		BlockPublicAccess: awss3.BlockPublicAccess_BLOCK_ALL(),
		RemovalPolicy:     awscdk.RemovalPolicy_DESTROY,
		AutoDeleteObjects: jsii.Bool(true),
	})

	cert := awscertificatemanager.NewCertificate(stack, jsii.String("SiteCert"), &awscertificatemanager.CertificateProps{
		DomainName:          jsii.String(domainName),
		SubjectAlternativeNames: jsii.Strings("*." + domainName),
		Validation:          awscertificatemanager.CertificateValidation_FromDns(nil),
	})

	distribution := awscloudfront.NewDistribution(stack, jsii.String("SiteDistribution"), &awscloudfront.DistributionProps{
		DefaultBehavior: &awscloudfront.BehaviorOptions{
			Origin: awscloudfrontorigins.S3BucketOrigin_WithOriginAccessControl(bucket, &awscloudfrontorigins.S3BucketOriginWithOACProps{}),
			ViewerProtocolPolicy: awscloudfront.ViewerProtocolPolicy_REDIRECT_TO_HTTPS,
		},
		DomainNames:       jsii.Strings(domainName, "www."+domainName),
		Certificate:       cert,
		DefaultRootObject: jsii.String("index.html"),
		ErrorResponses: &[]*awscloudfront.ErrorResponse{
			{
				HttpStatus:         jsii.Number(404),
				ResponseHttpStatus: jsii.Number(404),
				ResponsePagePath:   jsii.String("/404.html"),
			},
		},
	})

	awss3deployment.NewBucketDeployment(stack, jsii.String("DeploySite"), &awss3deployment.BucketDeploymentProps{
		Sources:              &[]awss3deployment.ISource{awss3deployment.Source_Asset(jsii.String("../public"), nil)},
		DestinationBucket:    bucket,
		Distribution:         distribution,
		DistributionPaths:    jsii.Strings("/*"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("DistributionDomainName"), &awscdk.CfnOutputProps{
		Value:       distribution.DistributionDomainName(),
		Description: jsii.String("CloudFront distribution domain — point your CNAME here"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("DistributionId"), &awscdk.CfnOutputProps{
		Value: distribution.DistributionId(),
	})

	return stack
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	NewInfraStack(app, "CodyOlsenSite", &InfraStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	app.Synth(nil)
}

func env() *awscdk.Environment {
	return &awscdk.Environment{
		Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
		Region:  jsii.String("us-east-1"),
	}
}
