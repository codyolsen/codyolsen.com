package main

import (
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfront"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfrontorigins"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
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
		DomainName:              jsii.String(domainName),
		SubjectAlternativeNames: jsii.Strings("*." + domainName),
		Validation:              awscertificatemanager.CertificateValidation_FromDns(nil),
	})

	distribution := awscloudfront.NewDistribution(stack, jsii.String("SiteDistribution"), &awscloudfront.DistributionProps{
		DefaultBehavior: &awscloudfront.BehaviorOptions{
			Origin:               awscloudfrontorigins.S3BucketOrigin_WithOriginAccessControl(bucket, &awscloudfrontorigins.S3BucketOriginWithOACProps{}),
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
		Sources:           &[]awss3deployment.ISource{awss3deployment.Source_Asset(jsii.String("../public"), nil)},
		DestinationBucket: bucket,
		Distribution:      distribution,
		DistributionPaths: jsii.Strings("/*"),
		MemoryLimit:       jsii.Number(1024),
	})

	deployRole := awsiam.NewRole(stack, jsii.String("DeployRole"), &awsiam.RoleProps{
		AssumedBy:   awsiam.NewAccountPrincipal(jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT"))),
		Description: jsii.String("Deploy and update the CodyOlsenRoot CDK stack"),
	})

	deployRole.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:     jsii.String("CloudFormation"),
		Actions: jsii.Strings("cloudformation:*"),
		Resources: jsii.Strings(
			"arn:aws:cloudformation:us-east-1:*:stack/CodyOlsenRoot/*",
			"arn:aws:cloudformation:us-east-1:*:stack/CDKToolkit/*",
		),
	}))

	deployRole.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:       jsii.String("CloudFormationList"),
		Actions:   jsii.Strings("cloudformation:DescribeStacks", "cloudformation:ListStacks", "cloudformation:GetTemplate"),
		Resources: jsii.Strings("*"),
	}))

	deployRole.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:     jsii.String("S3SiteBucket"),
		Actions: jsii.Strings("s3:*"),
		Resources: jsii.Strings(
			"arn:aws:s3:::codyolsenroot-*",
			"arn:aws:s3:::codyolsenroot-*/*",
		),
	}))

	deployRole.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:     jsii.String("S3CDKStagingBucket"),
		Actions: jsii.Strings("s3:GetObject", "s3:PutObject", "s3:ListBucket"),
		Resources: jsii.Strings(
			"arn:aws:s3:::cdk-*",
			"arn:aws:s3:::cdk-*/*",
		),
	}))

	deployRole.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:       jsii.String("CloudFront"),
		Actions:   jsii.Strings("cloudfront:*"),
		Resources: jsii.Strings("*"),
	}))

	deployRole.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:       jsii.String("ACM"),
		Actions:   jsii.Strings("acm:*"),
		Resources: jsii.Strings("*"),
	}))

	deployRole.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid: jsii.String("IAMForCDKRoles"),
		Actions: jsii.Strings(
			"iam:CreateRole", "iam:DeleteRole", "iam:GetRole",
			"iam:PutRolePolicy", "iam:DeleteRolePolicy",
			"iam:AttachRolePolicy", "iam:DetachRolePolicy", "iam:GetRolePolicy",
			"iam:PassRole", "iam:TagRole", "iam:UntagRole",
		),
		Resources: jsii.Strings("arn:aws:iam::*:role/CodyOlsenRoot-*"),
	}))

	deployRole.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:       jsii.String("LambdaBucketDeployment"),
		Actions:   jsii.Strings("lambda:*"),
		Resources: jsii.Strings("arn:aws:lambda:us-east-1:*:function:CodyOlsenRoot-*"),
	}))

	deployRole.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:       jsii.String("SSMBootstrapVersion"),
		Actions:   jsii.Strings("ssm:GetParameter"),
		Resources: jsii.Strings("arn:aws:ssm:us-east-1:*:parameter/cdk-bootstrap/*"),
	}))

	deployRole.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:       jsii.String("EC2DescribeAZs"),
		Actions:   jsii.Strings("ec2:DescribeAvailabilityZones"),
		Resources: jsii.Strings("*"),
	}))

	deployRole.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:       jsii.String("STSAssumeBootstrapRoles"),
		Actions:   jsii.Strings("sts:AssumeRole"),
		Resources: jsii.Strings("arn:aws:iam::*:role/cdk-*"),
	}))

	// GitHub Actions OIDC: the deploy workflow assumes this role on pushes to
	// main of codyolsen/codyolsen.com — no long-lived AWS keys in GitHub.
	githubOIDC := awsiam.NewOpenIdConnectProvider(stack, jsii.String("GitHubOIDC"), &awsiam.OpenIdConnectProviderProps{
		Url:       jsii.String("https://token.actions.githubusercontent.com"),
		ClientIds: jsii.Strings("sts.amazonaws.com"),
	})

	githubDeployRole := awsiam.NewRole(stack, jsii.String("GitHubDeployRole"), &awsiam.RoleProps{
		AssumedBy: awsiam.NewWebIdentityPrincipal(githubOIDC.OpenIdConnectProviderArn(), &map[string]interface{}{
			"StringEquals": map[string]interface{}{
				"token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
			},
			"StringLike": map[string]interface{}{
				"token.actions.githubusercontent.com:sub": "repo:codyolsen/codyolsen.com:ref:refs/heads/main",
			},
		}),
		Description: jsii.String("Assumed by GitHub Actions to sync the site to S3 and invalidate CloudFront"),
	})

	bucket.GrantReadWrite(githubDeployRole, nil)

	githubDeployRole.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:     jsii.String("CloudFrontInvalidation"),
		Actions: jsii.Strings("cloudfront:CreateInvalidation"),
		Resources: &[]*string{awscdk.Fn_Join(jsii.String(""), &[]*string{
			jsii.String("arn:aws:cloudfront::"), stack.Account(), jsii.String(":distribution/"), distribution.DistributionId(),
		})},
	}))

	awscdk.NewCfnOutput(stack, jsii.String("GitHubDeployRoleArn"), &awscdk.CfnOutputProps{
		Value:       githubDeployRole.RoleArn(),
		Description: jsii.String("Set as the AWS_DEPLOY_ROLE_ARN repo variable in GitHub"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("SiteBucketName"), &awscdk.CfnOutputProps{
		Value:       bucket.BucketName(),
		Description: jsii.String("Set as the SITE_BUCKET repo variable in GitHub"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("DeployRoleArn"), &awscdk.CfnOutputProps{
		Value:       deployRole.RoleArn(),
		Description: jsii.String("Assume this role to deploy the stack"),
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

	NewInfraStack(app, "CodyOlsenRoot", &InfraStackProps{
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
