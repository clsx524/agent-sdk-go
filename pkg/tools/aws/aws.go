package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
	"github.com/Ingenimax/agent-sdk-go/pkg/multitenancy"
)

// Tool implements an AWS provider tool
type Tool struct {
	config             aws.Config
	viewOnly           bool
	restrictedServices map[string]bool
}

// Option represents an option for configuring the tool
type Option func(*Tool)

// WithConfig sets the AWS config for the tool
func WithConfig(cfg aws.Config) Option {
	return func(t *Tool) {
		t.config = cfg
	}
}

// WithViewOnly sets the tool to view-only mode
func WithViewOnly(viewOnly bool) Option {
	return func(t *Tool) {
		t.viewOnly = viewOnly
	}
}

// WithRestrictedServices sets the list of allowed services
func WithRestrictedServices(services []string) Option {
	return func(t *Tool) {
		t.restrictedServices = make(map[string]bool)
		for _, service := range services {
			t.restrictedServices[strings.ToLower(service)] = true
		}
	}
}

// New creates a new AWS provider tool
func New(options ...Option) (*Tool, error) {
	// Load default AWS config
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	tool := &Tool{
		config:             cfg,
		restrictedServices: make(map[string]bool),
	}

	for _, option := range options {
		option(tool)
	}

	return tool, nil
}

// Name returns the name of the tool
func (t *Tool) Name() string {
	return "aws_provider"
}

// Description returns a description of what the tool does
func (t *Tool) Description() string {
	return `Interact with AWS services. Available services and their actions:

S3 (Simple Storage Service):
- list_buckets: List all S3 buckets
- create_bucket: Create a new S3 bucket
- delete_bucket: Delete an existing S3 bucket

EC2 (Elastic Compute Cloud):
- list_instances: List all EC2 instances
- start_instance: Start a stopped EC2 instance
- stop_instance: Stop a running EC2 instance

RDS (Relational Database Service):
- list_databases: List all RDS database instances
- describe_instance: Get detailed information about a specific RDS instance
- list_snapshots: List all RDS database snapshots

Lambda:
- list_functions: List all Lambda functions
- get_function: Get detailed information about a specific Lambda function

EKS (Elastic Kubernetes Service):
- list_clusters: List all EKS clusters
- describe_cluster: Get detailed information about a specific EKS cluster

VPC (Virtual Private Cloud):
- list_vpcs: List all VPCs
- describe_vpc: Get detailed information about a specific VPC
- list_subnets: List all subnets in a VPC

Route53:
- list_hosted_zones: List all Route53 hosted zones
- list_records: List DNS records for a hosted zone

Note: When view-only mode is enabled, all write operations (create, delete, start, stop) will be disabled.`
}

// Parameters returns the parameters that the tool accepts
func (t *Tool) Parameters() map[string]interfaces.ParameterSpec {
	// Define all possible services
	allServices := []interface{}{
		"s3", "ec2", "rds", "lambda", "eks", "vpc", "route53",
	}

	// Define all possible actions
	allActions := []interface{}{
		// S3 actions
		"list_buckets", "create_bucket", "delete_bucket",
		// EC2 actions
		"list_instances", "start_instance", "stop_instance",
		// RDS actions
		"list_databases", "describe_instance", "list_snapshots",
		// Lambda actions
		"list_functions", "get_function",
		// EKS actions
		"list_clusters", "describe_cluster",
		// VPC actions
		"list_vpcs", "describe_vpc", "list_subnets",
		// Route53 actions
		"list_hosted_zones", "list_records",
	}

	// If restricted services are set, filter the list
	var availableServices []interface{}
	if len(t.restrictedServices) > 0 {
		availableServices = []interface{}{}
		for _, service := range allServices {
			if t.restrictedServices[strings.ToLower(service.(string))] {
				availableServices = append(availableServices, service)
			}
		}
	} else {
		// If no restrictions, all services are available
		availableServices = allServices
	}

	return map[string]interfaces.ParameterSpec{
		"service": {
			Type:        "string",
			Description: "The AWS service to use",
			Required:    true,
			Enum:        availableServices,
		},
		"action": {
			Type:        "string",
			Description: "The action to perform",
			Required:    true,
			Enum:        allActions,
		},
		"params": {
			Type:        "object",
			Description: "Parameters for the action",
			Required:    false,
		},
	}
}

// Run executes the tool with the given input
func (t *Tool) Run(ctx context.Context, input string) (string, error) {
	// Parse input as JSON
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}

	// Get service parameter
	service, ok := params["service"].(string)
	if !ok || service == "" {
		return "", fmt.Errorf("service parameter is required")
	}

	// Check if service is allowed (only if restrictions are set)
	if len(t.restrictedServices) > 0 && !t.restrictedServices[strings.ToLower(service)] {
		return "", fmt.Errorf("service %s is not allowed", service)
	}

	// Get action parameter
	action, ok := params["action"].(string)
	if !ok || action == "" {
		return "", fmt.Errorf("action parameter is required")
	}

	// Get organization ID for permission checking
	orgID, _ := multitenancy.GetOrgID(ctx)

	// Check permissions
	if err := t.checkPermissions(ctx, orgID, service, action); err != nil {
		return "", err
	}

	// Execute action based on service
	switch strings.ToLower(service) {
	case "s3":
		return t.handleS3(ctx, action, params["params"])
	case "ec2":
		return t.handleEC2(ctx, action, params["params"])
	case "rds":
		return t.handleRDS(ctx, action, params["params"])
	case "lambda":
		return t.handleLambda(ctx, action, params["params"])
	case "eks":
		return t.handleEKS(ctx, action, params["params"])
	case "vpc":
		return t.handleVPC(ctx, action, params["params"])
	case "route53":
		return t.handleRoute53(ctx, action, params["params"])
	default:
		return "", fmt.Errorf("unsupported service: %s", service)
	}
}

// checkPermissions checks if the organization has permission to perform the action
func (t *Tool) checkPermissions(ctx context.Context, orgID, service, action string) error {
	// In a real implementation, this would check against a permission system
	// For now, we'll just allow all actions
	return nil
}

// handleS3 handles S3 actions
func (t *Tool) handleS3(ctx context.Context, action string, params interface{}) (string, error) {
	// Create S3 client
	client := s3.NewFromConfig(t.config)

	// Parse parameters
	var p map[string]interface{}
	if params != nil {
		var ok bool
		p, ok = params.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("params must be an object")
		}
	}

	// Execute action
	switch strings.ToLower(action) {
	case "list_buckets":
		return t.listBuckets(ctx, client)
	case "create_bucket":
		return t.createBucket(ctx, client, p)
	case "delete_bucket":
		return t.deleteBucket(ctx, client, p)
	default:
		return "", fmt.Errorf("unsupported S3 action: %s", action)
	}
}

// listBuckets lists all S3 buckets
func (t *Tool) listBuckets(ctx context.Context, client *s3.Client) (string, error) {
	// List buckets
	result, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return "", fmt.Errorf("failed to list buckets: %w", err)
	}

	// Format result
	var sb strings.Builder
	sb.WriteString("S3 Buckets:\n")
	for _, bucket := range result.Buckets {
		sb.WriteString(fmt.Sprintf("- %s (created: %s)\n", *bucket.Name, bucket.CreationDate))
	}

	return sb.String(), nil
}

// createBucket creates an S3 bucket
func (t *Tool) createBucket(ctx context.Context, client *s3.Client, params map[string]interface{}) (string, error) {
	if t.viewOnly {
		return "", fmt.Errorf("you are not allowed to create buckets")
	}

	// Get bucket name
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return "", fmt.Errorf("bucket name is required")
	}

	// Create bucket
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(name),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create bucket: %w", err)
	}

	return fmt.Sprintf("Bucket '%s' created successfully", name), nil
}

// deleteBucket deletes an S3 bucket
func (t *Tool) deleteBucket(ctx context.Context, client *s3.Client, params map[string]interface{}) (string, error) {
	if t.viewOnly {
		return "", fmt.Errorf("you are not allowed to delete buckets")
	}

	// Get bucket name
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return "", fmt.Errorf("bucket name is required")
	}

	// Delete bucket
	_, err := client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(name),
	})
	if err != nil {
		return "", fmt.Errorf("failed to delete bucket: %w", err)
	}

	return fmt.Sprintf("Bucket '%s' deleted successfully", name), nil
}

// handleEC2 handles EC2 actions
func (t *Tool) handleEC2(ctx context.Context, action string, params interface{}) (string, error) {
	// Create EC2 client
	client := ec2.NewFromConfig(t.config)

	// Parse parameters
	var p map[string]interface{}
	if params != nil {
		var ok bool
		p, ok = params.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("params must be an object")
		}
	}

	// Execute action
	switch strings.ToLower(action) {
	case "list_instances":
		return t.listInstances(ctx, client)
	case "start_instance":
		return t.startInstance(ctx, client, p)
	case "stop_instance":
		return t.stopInstance(ctx, client, p)
	default:
		return "", fmt.Errorf("unsupported EC2 action: %s", action)
	}
}

// listInstances lists all EC2 instances
func (t *Tool) listInstances(ctx context.Context, client *ec2.Client) (string, error) {
	// List instances
	result, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
	if err != nil {
		return "", fmt.Errorf("failed to list instances: %w", err)
	}

	// Format result
	var sb strings.Builder
	sb.WriteString("EC2 Instances:\n")
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			sb.WriteString(fmt.Sprintf("- ID: %s\n", *instance.InstanceId))
			sb.WriteString(fmt.Sprintf("  Type: %s\n", instance.InstanceType))
			sb.WriteString(fmt.Sprintf("  State: %s\n", instance.State.Name))
			sb.WriteString(fmt.Sprintf("  Public IP: %s\n", aws.ToString(instance.PublicIpAddress)))
			sb.WriteString(fmt.Sprintf("  Private IP: %s\n", aws.ToString(instance.PrivateIpAddress)))
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}

// startInstance starts an EC2 instance
func (t *Tool) startInstance(ctx context.Context, client *ec2.Client, params map[string]interface{}) (string, error) {
	if t.viewOnly {
		return "", fmt.Errorf("you are not allowed to start instances")
	}

	// Get instance ID
	id, ok := params["instance_id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("instance_id is required")
	}

	// Start instance
	_, err := client.StartInstances(ctx, &ec2.StartInstancesInput{
		InstanceIds: []string{id},
	})
	if err != nil {
		return "", fmt.Errorf("failed to start instance: %w", err)
	}

	return fmt.Sprintf("Instance '%s' started successfully", id), nil
}

// stopInstance stops an EC2 instance
func (t *Tool) stopInstance(ctx context.Context, client *ec2.Client, params map[string]interface{}) (string, error) {
	if t.viewOnly {
		return "", fmt.Errorf("you are not allowed to stop instances")
	}

	// Get instance ID
	id, ok := params["instance_id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("instance_id is required")
	}

	// Stop instance
	_, err := client.StopInstances(ctx, &ec2.StopInstancesInput{
		InstanceIds: []string{id},
	})
	if err != nil {
		return "", fmt.Errorf("failed to stop instance: %w", err)
	}

	return fmt.Sprintf("Instance '%s' stopped successfully", id), nil
}

// handleRDS handles RDS actions
func (t *Tool) handleRDS(ctx context.Context, action string, params interface{}) (string, error) {
	client := rds.NewFromConfig(t.config)

	// Parse parameters
	var p map[string]interface{}
	if params != nil {
		var ok bool
		p, ok = params.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("params must be an object")
		}
	}

	switch strings.ToLower(action) {
	case "list_databases":
		return t.listDatabases(ctx, client)
	case "describe_instance":
		return t.describeDBInstance(ctx, client, p)
	case "list_snapshots":
		return t.listDBSnapshots(ctx, client)
	default:
		return "", fmt.Errorf("unsupported RDS action: %s", action)
	}
}

// listDatabases lists all RDS instances
func (t *Tool) listDatabases(ctx context.Context, client *rds.Client) (string, error) {
	result, err := client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{})
	if err != nil {
		return "", fmt.Errorf("failed to list databases: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("RDS Instances:\n")
	for _, instance := range result.DBInstances {
		sb.WriteString(fmt.Sprintf("- ID: %s\n", *instance.DBInstanceIdentifier))
		sb.WriteString(fmt.Sprintf("  Engine: %s\n", *instance.Engine))
		sb.WriteString(fmt.Sprintf("  Status: %s\n", *instance.DBInstanceStatus))
		sb.WriteString(fmt.Sprintf("  Endpoint: %s\n", *instance.Endpoint.Address))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// describeDBInstance describes a specific RDS instance
func (t *Tool) describeDBInstance(ctx context.Context, client *rds.Client, params map[string]interface{}) (string, error) {
	id, ok := params["instance_id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("instance_id is required")
	}

	result, err := client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(id),
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe database: %w", err)
	}

	if len(result.DBInstances) == 0 {
		return "", fmt.Errorf("database instance not found")
	}

	instance := result.DBInstances[0]
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("RDS Instance Details for %s:\n", *instance.DBInstanceIdentifier))
	sb.WriteString(fmt.Sprintf("Engine: %s\n", *instance.Engine))
	sb.WriteString(fmt.Sprintf("Status: %s\n", *instance.DBInstanceStatus))
	sb.WriteString(fmt.Sprintf("Endpoint: %s\n", *instance.Endpoint.Address))
	sb.WriteString(fmt.Sprintf("Storage: %d GB\n", *instance.AllocatedStorage))
	sb.WriteString(fmt.Sprintf("Instance Class: %s\n", *instance.DBInstanceClass))
	sb.WriteString(fmt.Sprintf("Multi-AZ: %v\n", *instance.MultiAZ))

	return sb.String(), nil
}

// listDBSnapshots lists all RDS snapshots
func (t *Tool) listDBSnapshots(ctx context.Context, client *rds.Client) (string, error) {
	result, err := client.DescribeDBSnapshots(ctx, &rds.DescribeDBSnapshotsInput{})
	if err != nil {
		return "", fmt.Errorf("failed to list snapshots: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("RDS Snapshots:\n")
	for _, snapshot := range result.DBSnapshots {
		sb.WriteString(fmt.Sprintf("- ID: %s\n", *snapshot.DBSnapshotIdentifier))
		sb.WriteString(fmt.Sprintf("  Instance: %s\n", *snapshot.DBInstanceIdentifier))
		sb.WriteString(fmt.Sprintf("  Status: %s\n", *snapshot.Status))
		sb.WriteString(fmt.Sprintf("  Created: %s\n", snapshot.SnapshotCreateTime))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// handleLambda handles Lambda actions
func (t *Tool) handleLambda(ctx context.Context, action string, params interface{}) (string, error) {
	client := lambda.NewFromConfig(t.config)

	// Parse parameters
	var p map[string]interface{}
	if params != nil {
		var ok bool
		p, ok = params.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("params must be an object")
		}
	}

	switch strings.ToLower(action) {
	case "list_functions":
		return t.listFunctions(ctx, client)
	case "get_function":
		return t.getFunction(ctx, client, p)
	default:
		return "", fmt.Errorf("unsupported Lambda action: %s", action)
	}
}

// listFunctions lists all Lambda functions
func (t *Tool) listFunctions(ctx context.Context, client *lambda.Client) (string, error) {
	result, err := client.ListFunctions(ctx, &lambda.ListFunctionsInput{})
	if err != nil {
		return "", fmt.Errorf("failed to list functions: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("Lambda Functions:\n")
	for _, function := range result.Functions {
		sb.WriteString(fmt.Sprintf("- Name: %s\n", *function.FunctionName))
		sb.WriteString(fmt.Sprintf("  Runtime: %s\n", function.Runtime))
		sb.WriteString(fmt.Sprintf("  Handler: %s\n", *function.Handler))
		sb.WriteString(fmt.Sprintf("  Memory: %d MB\n", *function.MemorySize))
		sb.WriteString(fmt.Sprintf("  Timeout: %d seconds\n", *function.Timeout))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// getFunction gets details of a specific Lambda function
func (t *Tool) getFunction(ctx context.Context, client *lambda.Client, params map[string]interface{}) (string, error) {
	name, ok := params["function_name"].(string)
	if !ok || name == "" {
		return "", fmt.Errorf("function_name is required")
	}

	result, err := client.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(name),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get function: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Lambda Function Details for %s:\n", *result.Configuration.FunctionName))
	sb.WriteString(fmt.Sprintf("Runtime: %s\n", result.Configuration.Runtime))
	sb.WriteString(fmt.Sprintf("Handler: %s\n", *result.Configuration.Handler))
	sb.WriteString(fmt.Sprintf("Memory: %d MB\n", *result.Configuration.MemorySize))
	sb.WriteString(fmt.Sprintf("Timeout: %d seconds\n", *result.Configuration.Timeout))
	sb.WriteString(fmt.Sprintf("Last Modified: %s\n", *result.Configuration.LastModified))
	sb.WriteString(fmt.Sprintf("Code Size: %d bytes\n", result.Configuration.CodeSize))

	return sb.String(), nil
}

// handleEKS handles EKS actions
func (t *Tool) handleEKS(ctx context.Context, action string, params interface{}) (string, error) {
	client := eks.NewFromConfig(t.config)

	// Parse parameters
	var p map[string]interface{}
	if params != nil {
		var ok bool
		p, ok = params.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("params must be an object")
		}
	}

	switch strings.ToLower(action) {
	case "list_clusters":
		return t.listClusters(ctx, client)
	case "describe_cluster":
		return t.describeCluster(ctx, client, p)
	default:
		return "", fmt.Errorf("unsupported EKS action: %s", action)
	}
}

// listClusters lists all EKS clusters
func (t *Tool) listClusters(ctx context.Context, client *eks.Client) (string, error) {
	result, err := client.ListClusters(ctx, &eks.ListClustersInput{})
	if err != nil {
		return "", fmt.Errorf("failed to list clusters: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("EKS Clusters:\n")
	for _, cluster := range result.Clusters {
		sb.WriteString(fmt.Sprintf("- Name: %s\n", cluster))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// describeCluster describes a specific EKS cluster
func (t *Tool) describeCluster(ctx context.Context, client *eks.Client, params map[string]interface{}) (string, error) {
	name, ok := params["cluster_name"].(string)
	if !ok || name == "" {
		return "", fmt.Errorf("cluster_name is required")
	}

	result, err := client.DescribeCluster(ctx, &eks.DescribeClusterInput{
		Name: aws.String(name),
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe cluster: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("EKS Cluster Details for %s:\n", *result.Cluster.Name))
	sb.WriteString(fmt.Sprintf("Status: %s\n", result.Cluster.Status))
	sb.WriteString(fmt.Sprintf("Version: %s\n", *result.Cluster.Version))
	sb.WriteString(fmt.Sprintf("Endpoint: %s\n", *result.Cluster.Endpoint))
	sb.WriteString(fmt.Sprintf("Platform Version: %s\n", *result.Cluster.PlatformVersion))
	sb.WriteString(fmt.Sprintf("Role ARN: %s\n", *result.Cluster.RoleArn))
	sb.WriteString(fmt.Sprintf("VPC Config: %s\n", *result.Cluster.ResourcesVpcConfig.VpcId))

	return sb.String(), nil
}

// handleVPC handles VPC actions
func (t *Tool) handleVPC(ctx context.Context, action string, params interface{}) (string, error) {
	client := ec2.NewFromConfig(t.config)

	// Parse parameters
	var p map[string]interface{}
	if params != nil {
		var ok bool
		p, ok = params.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("params must be an object")
		}
	}

	switch strings.ToLower(action) {
	case "list_vpcs":
		return t.listVPCs(ctx, client)
	case "describe_vpc":
		return t.describeVPC(ctx, client, p)
	case "list_subnets":
		return t.listSubnets(ctx, client)
	default:
		return "", fmt.Errorf("unsupported VPC action: %s", action)
	}
}

// listVPCs lists all VPCs
func (t *Tool) listVPCs(ctx context.Context, client *ec2.Client) (string, error) {
	result, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
	if err != nil {
		return "", fmt.Errorf("failed to list VPCs: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("VPCs:\n")
	for _, vpc := range result.Vpcs {
		sb.WriteString(fmt.Sprintf("- ID: %s\n", *vpc.VpcId))
		sb.WriteString(fmt.Sprintf("  CIDR: %s\n", *vpc.CidrBlock))
		sb.WriteString(fmt.Sprintf("  State: %s\n", vpc.State))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// describeVPC describes a specific VPC
func (t *Tool) describeVPC(ctx context.Context, client *ec2.Client, params map[string]interface{}) (string, error) {
	id, ok := params["vpc_id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("vpc_id is required")
	}

	result, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		VpcIds: []string{id},
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe VPC: %w", err)
	}

	if len(result.Vpcs) == 0 {
		return "", fmt.Errorf("VPC not found")
	}

	vpc := result.Vpcs[0]
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("VPC Details for %s:\n", *vpc.VpcId))
	sb.WriteString(fmt.Sprintf("CIDR: %s\n", *vpc.CidrBlock))
	sb.WriteString(fmt.Sprintf("State: %s\n", vpc.State))
	sb.WriteString(fmt.Sprintf("Is Default: %v\n", *vpc.IsDefault))
	sb.WriteString(fmt.Sprintf("Tenancy: %s\n", vpc.InstanceTenancy))

	return sb.String(), nil
}

// listSubnets lists all subnets
func (t *Tool) listSubnets(ctx context.Context, client *ec2.Client) (string, error) {
	result, err := client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{})
	if err != nil {
		return "", fmt.Errorf("failed to list subnets: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("Subnets:\n")
	for _, subnet := range result.Subnets {
		sb.WriteString(fmt.Sprintf("- ID: %s\n", *subnet.SubnetId))
		sb.WriteString(fmt.Sprintf("  VPC ID: %s\n", *subnet.VpcId))
		sb.WriteString(fmt.Sprintf("  CIDR: %s\n", *subnet.CidrBlock))
		sb.WriteString(fmt.Sprintf("  AZ: %s\n", *subnet.AvailabilityZone))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// handleRoute53 handles Route53 actions
func (t *Tool) handleRoute53(ctx context.Context, action string, params interface{}) (string, error) {
	client := route53.NewFromConfig(t.config)

	// Parse parameters
	var p map[string]interface{}
	if params != nil {
		var ok bool
		p, ok = params.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("params must be an object")
		}
	}

	switch strings.ToLower(action) {
	case "list_hosted_zones":
		return t.listHostedZones(ctx, client)
	case "list_records":
		return t.listRecords(ctx, client, p)
	default:
		return "", fmt.Errorf("unsupported Route53 action: %s", action)
	}
}

// listHostedZones lists all Route53 hosted zones
func (t *Tool) listHostedZones(ctx context.Context, client *route53.Client) (string, error) {
	result, err := client.ListHostedZones(ctx, &route53.ListHostedZonesInput{})
	if err != nil {
		return "", fmt.Errorf("failed to list hosted zones: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("Route53 Hosted Zones:\n")
	for _, zone := range result.HostedZones {
		sb.WriteString(fmt.Sprintf("- ID: %s\n", *zone.Id))
		sb.WriteString(fmt.Sprintf("  Name: %s\n", *zone.Name))
		sb.WriteString(fmt.Sprintf("  Private: %v\n", zone.Config.PrivateZone))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// listRecords lists records for a hosted zone
func (t *Tool) listRecords(ctx context.Context, client *route53.Client, params map[string]interface{}) (string, error) {
	zoneID, ok := params["zone_id"].(string)
	if !ok || zoneID == "" {
		return "", fmt.Errorf("zone_id is required")
	}

	result, err := client.ListResourceRecordSets(ctx, &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
	})
	if err != nil {
		return "", fmt.Errorf("failed to list records: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("Route53 Records:\n")
	for _, record := range result.ResourceRecordSets {
		sb.WriteString(fmt.Sprintf("- Name: %s\n", *record.Name))
		sb.WriteString(fmt.Sprintf("  Type: %s\n", record.Type))
		sb.WriteString(fmt.Sprintf("  TTL: %d\n", *record.TTL))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// Execute implements the interfaces.Tool interface
func (t *Tool) Execute(ctx context.Context, input string) (string, error) {
	return t.Run(ctx, input)
}
