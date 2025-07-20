# AWS Inventory Tool (awsinv)

A production-quality Go CLI tool for inventorying active AWS resources across regions with minimal external dependencies.

## Features

- **Comprehensive Coverage**: Enumerates AWS services across all enabled regions
- **Unified Model**: Normalizes resources into a consistent Resource model
- **Multiple Output Formats**: Table, JSON, and CSV output
- **Fast & Parallel**: Concurrent collection with configurable parallelism
- **Robust Error Handling**: Graceful error handling with partial successes
- **Minimal Dependencies**: Only official AWS SDK v2, Cobra, and go-cmp
- **Static Binary**: Compiles as a self-contained binary
- **Flexible Filtering**: Filter by resource properties and tags
- **Role Support**: AWS profile and role assumption support

## Supported Services

### Phase v0.1.0
- **EC2 instances** - Virtual machines and their metadata
- **RDS database instances** - Relational database services
- **Lambda functions** - Serverless compute functions
- **S3 buckets** - Object storage buckets (global)
- **DynamoDB tables** - NoSQL database tables
- **Step Functions** - Serverless workflow orchestration
- **CloudWatch alarms** - Monitoring and alerting
- **ECS clusters and services** - Container orchestration

## Installation

### From Source
```bash
git clone <repository>
cd aws-cost-estimate
make build
```

### Build Options
```bash
# Build for current platform
make build

# Build for all platforms (Linux, macOS, Windows)
make build-all

# Install to GOPATH/bin
make install
```

## Usage

### Basic Usage
```bash
# Inventory all services in all regions
./awsinv

# Specific services and regions
./awsinv --services ec2,rds --regions us-east-1,us-west-2

# JSON output with filtering
./awsinv --output json --filter state=running --filter name=prod*

# HTML output with cost estimates
./awsinv --output html > inventory.html

# Verbose output with role assumption
./awsinv --verbose --role-arn arn:aws:iam::123456789012:role/InventoryRole
```

### Command Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `--services` | Comma-separated list of services (ec2,rds,lambda,s3,dynamodb,sfn,cloudwatch,ecs) | all |
| `--regions` | Comma-separated list of regions | all enabled |
| `--output` | Output format (table\|json\|csv\|html) | table |
| `--parallel` | Number of parallel collectors | 12 |
| `--timeout` | Overall context timeout | 5m |
| `--fail-fast` | Abort on first collector error | false |
| `--verbose` | Log progress to stderr | false |
| `--no-color` | Disable ANSI color in table | false |
| `--profile` | AWS shared credentials profile | default |
| `--role-arn` | ARN of role to assume | none |
| `--external-id` | External ID for role assumption | none |
| `--sort` | Sort field (service\|region\|id\|name\|type\|state) | service |
| `--filter` | Filter resources (key=value, repeatable) | none |

### Filtering

Filters support exact matches and substring matching:
```bash
# Exact match
./awsinv --filter state=running

# Substring match (ends with *)
./awsinv --filter name=prod*

# Multiple filters
./awsinv --filter service=ec2 --filter state=running

# Tag filtering
./awsinv --filter Environment=production
```

### Output Formats

#### Table Format (Default)
```
AWS Resource Inventory Summary
==============================
Total Resources: 42
Duration: 2.3s
Errors: 0

By Service:
  ec2: 15
  rds: 8
  lambda: 12
  s3: 7

Resources:
SERVICE      REGION          ID                   NAME             TYPE       STATE      CLASS     
-------      ------          --                   ----             ----       -----      -----     
ec2          us-east-1       i-1234567890abcdef0  web-server-01    t3.micro   running    t3.micro  
rds          us-east-1       db-1234567890        prod-db          mysql      available  db.t3.micro
```

#### JSON Format
```json
{
  "resources": [
    {
      "service": "ec2",
      "region": "us-east-1",
      "id": "i-1234567890abcdef0",
      "name": "web-server-01",
      "type": "t3.micro",
      "state": "running",
      "class": "t3.micro",
      "createdAt": "2024-01-15T10:30:00Z",
      "tags": {
        "Environment": "production",
        "Project": "web-app"
      },
      "extra": {
        "privateIp": "10.0.1.100",
        "architecture": "x86_64"
      }
    }
  ],
  "summary": {
    "totalResources": 42,
    "byService": {"ec2": 15, "rds": 8, "lambda": 12, "s3": 7},
    "byRegion": {"us-east-1": 25, "us-west-2": 17},
    "byState": {"running": 30, "stopped": 12},
    "errors": 0,
    "duration": "2.3s",
    "regions": ["us-east-1", "us-west-2"],
    "services": ["ec2", "rds", "lambda", "s3"]
  }
}
```

#### CSV Format
```csv
Service,Region,ID,Name,Type,State,Class,CreatedAt,Tags
ec2,us-east-1,i-1234567890abcdef0,web-server-01,t3.micro,running,t3.micro,2024-01-15T10:30:00Z,Environment=production,Project=web-app
rds,us-east-1,db-1234567890,prod-db,mysql,available,db.t3.micro,2024-01-10T08:15:00Z,Environment=production
```

#### HTML Format
The HTML output generates a beautiful, interactive report with:
- **Summary dashboard** with resource counts and statistics
- **Cost estimates** for each service (monthly estimates)
- **Interactive table** with color-coded service badges
- **Error reporting** if any collection issues occurred
- **Responsive design** that works on desktop and mobile

```bash
./awsinv --output html > inventory.html
# Open inventory.html in your browser
```

### Cost Estimation Details

The HTML output includes detailed cost estimates with explanations for each service:

#### **EC2 Instances**
- **Basis**: On-demand pricing from us-east-1 region
- **Calculation**: Instance type × 730 hours/month
- **Examples**: t3.micro ($8.47), t3.small ($16.94), m5.large ($86.40)
- **Assumptions**: 24/7 usage, excludes data transfer and storage

#### **RDS Databases**
- **Basis**: On-demand pricing for Single-AZ deployments
- **Calculation**: Instance class × 730 hours/month
- **Examples**: db.t3.micro ($15), db.m5.large ($171)
- **Assumptions**: 24/7 usage, excludes storage and backup costs

#### **Lambda Functions**
- **Basis**: Estimated moderate usage
- **Calculation**: $5/month per function
- **Assumptions**: 1000 requests/month, 128MB memory, 100ms execution

#### **S3 Buckets**
- **Basis**: Estimated minimal usage
- **Calculation**: $1/month per bucket
- **Assumptions**: 1GB storage, standard class, low request volume

#### **DynamoDB Tables**
- **Basis**: On-demand billing mode
- **Calculation**: $10/month per table
- **Assumptions**: Moderate read/write capacity, minimal storage

#### **Step Functions**
- **Basis**: Estimated moderate usage
- **Calculation**: $5/month per state machine
- **Assumptions**: Standard workflow complexity, moderate execution

#### **CloudWatch Alarms**
- **Basis**: Estimated moderate usage
- **Calculation**: $2/month per alarm
- **Assumptions**: Standard resolution metrics, moderate volume

#### **ECS Clusters & Services**
- **Basis**: Infrastructure-dependent costs
- **Calculation**: $5/month per cluster, $15/month per service
- **Assumptions**: Cluster management overhead, moderate task requirements

## Resource Model

All AWS resources are normalized into a unified model:

```go
type Resource struct {
    Service      string                 `json:"service"`
    Region       string                 `json:"region"`
    ID           string                 `json:"id"`
    Name         string                 `json:"name,omitempty"`
    Type         string                 `json:"type,omitempty"`          // instance type, engine, runtime...
    State        string                 `json:"state,omitempty"`
    Class        string                 `json:"class,omitempty"`         // db class, memory size, etc.
    CreatedAt    *time.Time             `json:"createdAt,omitempty"`
    Tags         map[string]string      `json:"tags,omitempty"`
    Extra        map[string]interface{} `json:"extra,omitempty"`
}
```

## AWS Configuration

The tool uses standard AWS credential resolution:

1. **Environment variables** (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. **Shared credentials file** (`~/.aws/credentials`)
3. **AWS profiles** (`--profile` flag)
4. **IAM roles** (EC2 instance profiles, ECS task roles)
5. **Role assumption** (`--role-arn` flag)

### Required Permissions

Minimum IAM permissions required:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeRegions",
        "ec2:DescribeInstances",
        "rds:DescribeDBInstances",
        "lambda:ListFunctions",
        "s3:ListBuckets",
        "dynamodb:ListTables",
        "dynamodb:DescribeTable",
        "sfn:ListStateMachines",
        "sfn:DescribeStateMachine",
        "cloudwatch:DescribeAlarms",
        "ecs:ListClusters",
        "ecs:DescribeClusters",
        "ecs:ListServices",
        "ecs:DescribeServices"
      ],
      "Resource": "*"
    }
  ]
}
```

## Development

### Prerequisites
- Go 1.21+
- AWS credentials configured

### Building
```bash
# Build binary
make build

# Run tests
make test

# Format code
make fmt

# Lint code
make lint
```

### Project Structure
```
.
├── cmd/awsinv/          # CLI application
├── pkg/
│   ├── aws/            # AWS client management
│   ├── collectors/     # Service-specific collectors
│   ├── models/         # Data models
│   ├── orchestrator/   # Collection orchestration
│   └── output/         # Output formatters
├── Makefile            # Build automation
└── README.md          # This file
```

### Adding New Services

1. Create a new collector in `pkg/collectors/`
2. Implement the `Collector` interface
3. Register the collector in `pkg/orchestrator/orchestrator.go`
4. Update documentation

Example collector structure:
```go
type NewServiceCollector struct {
    clientManager *aws.ClientManager
}

func (c *NewServiceCollector) Name() string {
    return "newservice"
}

func (c *NewServiceCollector) Collect(ctx context.Context, region string) ([]models.Resource, error) {
    // Implementation
}
```

## License

[Add your license here]

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## Version History

- **v0.1.0** - Initial release with EC2, RDS, Lambda, and S3 support 