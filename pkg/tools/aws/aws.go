package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
	"github.com/Ingenimax/agent-sdk-go/pkg/multitenancy"
)

// Tool implements an AWS provider tool
type Tool struct {
	config aws.Config
}

// Option represents an option for configuring the tool
type Option func(*Tool)

// WithConfig sets the AWS config for the tool
func WithConfig(cfg aws.Config) Option {
	return func(t *Tool) {
		t.config = cfg
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
		config: cfg,
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
	return "Interact with AWS services like S3, EC2, etc."
}

// Parameters returns the parameters that the tool accepts
func (t *Tool) Parameters() map[string]interfaces.ParameterSpec {
	return map[string]interfaces.ParameterSpec{
		"service": {
			Type:        "string",
			Description: "The AWS service to use (s3, ec2)",
			Required:    true,
			Enum:        []interface{}{"s3", "ec2"},
		},
		"action": {
			Type:        "string",
			Description: "The action to perform",
			Required:    true,
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
