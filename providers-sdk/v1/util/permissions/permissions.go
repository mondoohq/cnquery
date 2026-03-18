// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// PermissionManifest is the JSON output for a provider's permissions.
type PermissionManifest struct {
	Provider    string             `json:"provider"`
	Version     string             `json:"version"`
	GeneratedAt string             `json:"generated_at"`
	Permissions []string           `json:"permissions"`
	Details     []PermissionDetail `json:"details"`
}

// PermissionDetail describes a single extracted API call and its mapped permission.
type PermissionDetail struct {
	Permission string `json:"permission"`
	Service    string `json:"service"`
	Action     string `json:"action"`
	SourceFile string `json:"source_file"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: permissions <provider-path> [--output <path>]\n")
		fmt.Fprintf(os.Stderr, "  provider-path: path to provider directory (e.g., providers/aws)\n")
		os.Exit(1)
	}

	providerPath := os.Args[1]
	outputPath := ""
	for i, arg := range os.Args {
		if arg == "--output" && i+1 < len(os.Args) {
			outputPath = os.Args[i+1]
		}
	}

	providerName := filepath.Base(providerPath)
	resourcesDir := filepath.Join(providerPath, "resources")

	// Read provider version from config/config.go
	version := readProviderVersion(filepath.Join(providerPath, "config", "config.go"))

	// Detect provider type and extract permissions
	var details []PermissionDetail
	switch providerName {
	case "aws":
		details = extractAWSPermissions(resourcesDir)
	case "gcp":
		details = extractGCPPermissions(resourcesDir)
	case "azure":
		details = extractAzurePermissions(resourcesDir)
	default:
		fmt.Fprintf(os.Stderr, "skipping %s: not a supported cloud provider (aws, gcp, azure)\n", providerName)
		os.Exit(0)
	}

	// Deduplicate and sort permissions
	permSet := map[string]bool{}
	for _, d := range details {
		permSet[d.Permission] = true
	}
	permissions := make([]string, 0, len(permSet))
	for p := range permSet {
		permissions = append(permissions, p)
	}
	sort.Strings(permissions)

	// Sort details for stable output
	sort.Slice(details, func(i, j int) bool {
		if details[i].Permission != details[j].Permission {
			return details[i].Permission < details[j].Permission
		}
		return details[i].SourceFile < details[j].SourceFile
	})

	// Deduplicate details (same permission + source file)
	if len(details) > 0 {
		deduped := []PermissionDetail{details[0]}
		for i := 1; i < len(details); i++ {
			prev := deduped[len(deduped)-1]
			if details[i].Permission != prev.Permission || details[i].SourceFile != prev.SourceFile {
				deduped = append(deduped, details[i])
			}
		}
		details = deduped
	}

	manifest := PermissionManifest{
		Provider:    providerName,
		Version:     version,
		GeneratedAt: deterministicTimestamp(),
		Permissions: permissions,
		Details:     details,
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling JSON: %v\n", err)
		os.Exit(1)
	}
	data = append(data, '\n')

	if outputPath == "" {
		outputPath = filepath.Join(providerPath, "resources", providerName+".permissions.json")
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating output directory: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("  %s: %d permissions → %s\n", providerName, len(permissions), outputPath)
}

var versionRegex = regexp.MustCompile(`Version:\s*"([^"]+)"`)

func readProviderVersion(configPath string) string {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "unknown"
	}
	m := versionRegex.FindSubmatch(data)
	if m == nil {
		return "unknown"
	}
	return string(m[1])
}

// deterministicTimestamp returns a reproducible timestamp for the manifest.
// It checks SOURCE_DATE_EPOCH first (standard reproducible-builds env var),
// then falls back to the latest git commit timestamp.
func deterministicTimestamp() string {
	// Check SOURCE_DATE_EPOCH (Unix timestamp) and format as RFC 3339
	if epoch := os.Getenv("SOURCE_DATE_EPOCH"); epoch != "" {
		secs, err := strconv.ParseInt(epoch, 10, 64)
		if err == nil {
			return time.Unix(secs, 0).UTC().Format(time.RFC3339)
		}
	}

	// Fall back to git commit timestamp
	out, err := exec.Command("git", "log", "-1", "--format=%cI").Output()
	if err == nil {
		ts := strings.TrimSpace(string(out))
		if ts != "" {
			return ts
		}
	}

	return "unknown"
}

// listGoFiles returns all non-test, non-generated .go files in a directory tree.
func listGoFiles(dir string) []string {
	var files []string
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".go") {
			return nil
		}
		if strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, ".lr.go") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files
}

// =============================================================================
// AWS Permission Extraction
// =============================================================================

// awsServiceNameOverrides maps AWS SDK package names to IAM service prefixes
// where they differ from the Go package name.
var awsServiceNameOverrides = map[string]string{
	"cloudwatchlogs":           "logs",
	"configservice":            "config",
	"cognitoidentityprovider":  "cognito-idp",
	"cognitoidentity":          "cognito-identity",
	"databasemigrationservice": "dms",
	"directoryservice":         "ds",
	"docdb":                    "rds",
	"elasticsearchservice":     "es",
	"elasticloadbalancing":     "elasticloadbalancing",
	"elasticloadbalancingv2":   "elasticloadbalancing",
	"firehose":                 "firehose",
	"inspector2":               "inspector2",
	"kafka":                    "kafka",
	"lightsail":                "lightsail",
	"macie2":                   "macie2",
	"memorydb":                 "memorydb",
	"mq":                       "mq",
	"neptune":                  "rds",
	"networkfirewall":          "network-firewall",
	"opensearch":               "es",
	"organizations":            "organizations",
	"pipes":                    "pipes",
	"route53domains":           "route53domains",
	"s3control":                "s3",
	"secretsmanager":           "secretsmanager",
	"securityhub":              "securityhub",
	"shield":                   "shield",
	"timestreamwrite":          "timestream",
	"timestreaminfluxdb":       "timestream-influxdb",
	"workspacesweb":            "workspaces-web",
	"applicationautoscaling":   "application-autoscaling",
	"elasticbeanstalk":         "elasticbeanstalk",
	"elasticache":              "elasticache",
	"accessanalyzer":           "access-analyzer",
}

func extractAWSPermissions(resourcesDir string) []PermissionDetail {
	var details []PermissionDetail
	files := listGoFiles(resourcesDir)

	for _, filePath := range files {
		fileName := filepath.Base(filePath)
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, filePath, nil, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to parse %s: %v\n", filePath, err)
			continue
		}

		// Build import map: alias -> package name
		// e.g., "sns" -> "sns", "s3control" -> "s3control"
		awsImports := extractAWSImports(f)
		if len(awsImports) == 0 {
			continue
		}

		// Track variable -> service mappings within each function
		// e.g., svc := conn.Sns(region) -> svc maps to "sns"
		ast.Inspect(f, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok {
				return true
			}

			// Build variable -> service map for this function
			varServices := map[string]string{}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				assignStmt, ok := n.(*ast.AssignStmt)
				if !ok {
					return true
				}
				// Look for: svc := conn.ServiceMethod(region)
				// or: svc, err := conn.ServiceMethod(region)
				for i, rhs := range assignStmt.Rhs {
					call, ok := rhs.(*ast.CallExpr)
					if !ok {
						continue
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						continue
					}
					methodName := sel.Sel.Name
					// Check if this is a known connection method (e.g., conn.Sns, conn.Ec2)
					svcName := awsConnectionMethodToService(methodName)
					if svcName == "" {
						continue
					}
					if i < len(assignStmt.Lhs) {
						if ident, ok := assignStmt.Lhs[i].(*ast.Ident); ok {
							varServices[ident.Name] = svcName
						}
					}
				}
				return true
			})

			// Find API calls: svc.MethodName(ctx, &input)
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				methodName := sel.Sel.Name

				// Pattern 1: svc.MethodName() where svc is a tracked variable
				if ident, ok := sel.X.(*ast.Ident); ok {
					if svcName, ok := varServices[ident.Name]; ok {
						if isAWSAPIMethod(methodName) {
							iamService := awsServiceToIAM(svcName)
							details = append(details, PermissionDetail{
								Permission: iamService + ":" + methodName,
								Service:    iamService,
								Action:     methodName,
								SourceFile: fileName,
							})
						}
					}
				}

				// Pattern 2: pkg.NewMethodPaginator() where pkg is an AWS import
				if ident, ok := sel.X.(*ast.Ident); ok {
					if _, isAWSPkg := awsImports[ident.Name]; isAWSPkg {
						if strings.HasPrefix(methodName, "New") && strings.HasSuffix(methodName, "Paginator") {
							action := strings.TrimPrefix(methodName, "New")
							action = strings.TrimSuffix(action, "Paginator")
							iamService := awsServiceToIAM(awsImports[ident.Name])
							details = append(details, PermissionDetail{
								Permission: iamService + ":" + action,
								Service:    iamService,
								Action:     action,
								SourceFile: fileName,
							})
						}
					}
				}

				return true
			})

			return false // don't recurse into nested functions again
		})
	}

	return details
}

// extractAWSImports returns a map of import alias -> package name for AWS SDK imports.
func extractAWSImports(f *ast.File) map[string]string {
	result := map[string]string{}
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if !strings.Contains(path, "github.com/aws/aws-sdk-go-v2/service/") {
			continue
		}
		pkgName := filepath.Base(path)
		alias := pkgName
		if imp.Name != nil {
			alias = imp.Name.Name
		}
		result[alias] = pkgName
	}
	return result
}

// awsConnectionMethodToService maps AwsConnection method names to service names.
// e.g., "Sns" -> "sns", "Ec2" -> "ec2", "S3Control" -> "s3control"
func awsConnectionMethodToService(method string) string {
	// The connection methods are PascalCase versions of the service name
	// e.g., Ec2, Iam, Sns, S3, S3Control, CloudwatchLogs, etc.
	lower := strings.ToLower(method)
	// Known connection method names (lowercase -> service package name)
	knownMethods := map[string]string{
		"organizations":            "organizations",
		"ec2":                      "ec2",
		"wafv2":                    "wafv2",
		"ecs":                      "ecs",
		"iam":                      "iam",
		"ecr":                      "ecr",
		"ecrpublic":                "ecrpublic",
		"s3":                       "s3",
		"s3control":                "s3control",
		"cloudtrail":               "cloudtrail",
		"cloudwatch":               "cloudwatch",
		"cloudwatchlogs":           "cloudwatchlogs",
		"configservice":            "configservice",
		"rds":                      "rds",
		"lambda":                   "lambda",
		"dynamodb":                 "dynamodb",
		"kms":                      "kms",
		"sns":                      "sns",
		"sqs":                      "sqs",
		"redshift":                 "redshift",
		"cloudfront":               "cloudfront",
		"cloudformation":           "cloudformation",
		"ssm":                      "ssm",
		"sts":                      "sts",
		"acm":                      "acm",
		"elb":                      "elasticloadbalancing",
		"elbv2":                    "elasticloadbalancingv2",
		"route53":                  "route53",
		"route53domains":           "route53domains",
		"eks":                      "eks",
		"efs":                      "efs",
		"apigateway":               "apigateway",
		"autoscaling":              "autoscaling",
		"backup":                   "backup",
		"codebuild":                "codebuild",
		"emr":                      "emr",
		"guardduty":                "guardduty",
		"kinesis":                  "kinesis",
		"secretsmanager":           "secretsmanager",
		"securityhub":              "securityhub",
		"shield":                   "shield",
		"batch":                    "batch",
		"drs":                      "drs",
		"athena":                   "athena",
		"glue":                     "glue",
		"dms":                      "databasemigrationservice",
		"databasemigrationservice": "databasemigrationservice",
		"fsx":                      "fsx",
		"neptune":                  "neptune",
		"opensearch":               "opensearch",
		"docdb":                    "docdb",
		"elasticache":              "elasticache",
		"elasticbeanstalk":         "elasticbeanstalk",
		"elasticsearchservice":     "elasticsearchservice",
		"es":                       "elasticsearchservice",
		"eventbridge":              "eventbridge",
		"firehose":                 "firehose",
		"inspector2":               "inspector2",
		"kafka":                    "kafka",
		"lightsail":                "lightsail",
		"macie2":                   "macie2",
		"memorydb":                 "memorydb",
		"mq":                       "mq",
		"networkfirewall":          "networkfirewall",
		"appstream":                "appstream",
		"applicationautoscaling":   "applicationautoscaling",
		"account":                  "account",
		"sagemaker":                "sagemaker",
		"cognitoidentity":          "cognitoidentity",
		"cognitoidentityprovider":  "cognitoidentityprovider",
		"directoryservice":         "directoryservice",
		"pipes":                    "pipes",
		"scheduler":                "scheduler",
		"accessanalyzer":           "accessanalyzer",
		"timestreamwrite":          "timestreamwrite",
		"timestreaminfluxdb":       "timestreaminfluxdb",
		"workdocs":                 "workdocs",
		"workspaces":               "workspaces",
		"workspacesweb":            "workspacesweb",
		"codedeploy":               "codedeploy",
	}
	if svc, ok := knownMethods[lower]; ok {
		return svc
	}
	return ""
}

// awsServiceToIAM maps an AWS SDK package name to the IAM service prefix.
func awsServiceToIAM(sdkPkg string) string {
	if override, ok := awsServiceNameOverrides[sdkPkg]; ok {
		return override
	}
	return sdkPkg
}

// isAWSAPIMethod returns true if the method name looks like an AWS API call.
func isAWSAPIMethod(name string) bool {
	prefixes := []string{
		"Describe", "List", "Get", "Put", "Create", "Delete", "Update",
		"Batch", "Generate", "Assume", "Decode", "Lookup", "Search",
		"Tag", "Untag", "Enable", "Disable", "Start", "Stop",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// =============================================================================
// GCP Permission Extraction
// =============================================================================

func extractGCPPermissions(resourcesDir string) []PermissionDetail {
	var details []PermissionDetail
	files := listGoFiles(resourcesDir)

	for _, filePath := range files {
		fileName := filepath.Base(filePath)
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, filePath, nil, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to parse %s: %v\n", filePath, err)
			continue
		}

		// Build import map
		gcpImports := extractGCPImports(f)
		if len(gcpImports) == 0 {
			continue
		}

		// Track client/service variables and find API calls
		details = append(details, extractGCPgRPCCalls(f, gcpImports, fileName)...)
	}

	return details
}

// gcpImportInfo holds information about a GCP import.
type gcpImportInfo struct {
	alias   string // import alias used in code
	service string // GCP service name (e.g., "compute", "iam", "kms")
	style   string // "rest" or "grpc"
	path    string // full import path
}

func extractGCPImports(f *ast.File) map[string]*gcpImportInfo {
	result := map[string]*gcpImportInfo{}
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		info := classifyGCPImport(path)
		if info == nil {
			continue
		}
		if imp.Name != nil {
			info.alias = imp.Name.Name
		}
		result[info.alias] = info
	}
	return result
}

// classifyGCPImport determines if an import path is a GCP SDK and returns info.
func classifyGCPImport(path string) *gcpImportInfo {
	// REST discovery-based APIs: google.golang.org/api/<service>/v1
	// e.g., google.golang.org/api/compute/v1 -> parts: [google.golang.org, api, compute, v1]
	if strings.HasPrefix(path, "google.golang.org/api/") {
		parts := strings.Split(path, "/")
		if len(parts) >= 3 {
			svc := parts[2] // "compute", "dns", "sqladmin", etc.
			return &gcpImportInfo{
				alias:   svc,
				service: gcpServiceName(svc),
				style:   "rest",
				path:    path,
			}
		}
	}

	// gRPC client APIs: cloud.google.com/go/<service>/apiv1
	// e.g., cloud.google.com/go/kms/apiv1 -> service "kms"
	//        cloud.google.com/go/spanner/admin/database/apiv1 -> service "spanner"
	//        cloud.google.com/go/logging/logadmin -> service "logging"
	//        cloud.google.com/go/pubsub -> service "pubsub"
	// cloud.google.com/go/<service>[/<sub>...]/apiv1
	// e.g., cloud.google.com/go/kms/apiv1 -> parts: [cloud.google.com, go, kms, apiv1]
	//        cloud.google.com/go/iam/admin/apiv1 -> parts: [cloud.google.com, go, iam, admin, apiv1]
	//        cloud.google.com/go/pubsub -> parts: [cloud.google.com, go, pubsub]
	if strings.HasPrefix(path, "cloud.google.com/go/") {
		parts := strings.Split(path, "/")
		if len(parts) >= 3 {
			// The primary service name is always parts[2] (first component after "go")
			svc := parts[2] // "kms", "iam", "compute", "pubsub", etc.
			// Determine the alias: use the last path component unless it's a version
			alias := filepath.Base(path)
			if strings.HasPrefix(alias, "apiv") || alias == "v2" {
				alias = svc
			}
			// Skip protobuf packages (end in "pb")
			if strings.HasSuffix(alias, "pb") {
				return nil
			}
			return &gcpImportInfo{
				alias:   alias,
				service: gcpServiceName(svc),
				style:   "grpc",
				path:    path,
			}
		}
	}

	return nil
}

// gcpServiceName normalizes GCP service names.
var gcpServiceNameMap = map[string]string{
	"compute":              "compute",
	"cloudresourcemanager": "cloudresourcemanager",
	"iam":                  "iam",
	"dns":                  "dns",
	"bigquery":             "bigquery",
	"logging":              "logging",
	"monitoring":           "monitoring",
	"container":            "container",
	"storage":              "storage",
	"sqladmin":             "sqladmin",
	"serviceusage":         "serviceusage",
	"apikeys":              "apikeys",
	"kms":                  "cloudkms",
	"functions":            "cloudfunctions",
	"run":                  "run",
	"artifactregistry":     "artifactregistry",
	"alloydb":              "alloydb",
	"aiplatform":           "aiplatform",
	"privateca":            "privateca",
	"binaryauthorization":  "binaryauthorization",
	"spanner":              "spanner",
	"redis":                "redis",
	"filestore":            "file",
	"scheduler":            "cloudscheduler",
	"deploy":               "clouddeploy",
	"firestore":            "datastore",
	"essentialcontacts":    "essentialcontacts",
	"accessapproval":       "accessapproval",
	"logadmin":             "logging",
	"pubsub":               "pubsub",
	"dataproc":             "dataproc",
	"notebooks":            "notebooks",
	"composer":             "composer",
	"bigtable":             "bigtable",
	"memcache":             "memcache",
	"recaptchaenterprise":  "recaptchaenterprise",
	"cloudbuild":           "cloudbuild",
	"certificatemanager":   "certificatemanager",
	"secretmanager":        "secretmanager",
	"batch":                "batch",
	"dataplex":             "dataplex",
	"orgpolicy":            "orgpolicy",
}

func gcpServiceName(pkg string) string {
	if name, ok := gcpServiceNameMap[pkg]; ok {
		return name
	}
	return pkg
}

// extractGCPgRPCCalls finds gRPC client creation and subsequent method calls.
func extractGCPgRPCCalls(f *ast.File, imports map[string]*gcpImportInfo, fileName string) []PermissionDetail {
	var details []PermissionDetail

	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}
		if fn.Body == nil {
			return true
		}

		// Track client variables: varName -> service info
		clientVars := map[string]*gcpImportInfo{}

		// Also track REST service variables
		restVars := map[string]*gcpImportInfo{}

		ast.Inspect(fn.Body, func(n ast.Node) bool {
			assignStmt, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}

			for i, rhs := range assignStmt.Rhs {
				call, ok := rhs.(*ast.CallExpr)
				if !ok {
					continue
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					continue
				}
				methodName := sel.Sel.Name

				pkgIdent, ok := sel.X.(*ast.Ident)
				if !ok {
					continue
				}
				pkgName := pkgIdent.Name

				imp, isGCPImport := imports[pkgName]
				if !isGCPImport {
					continue
				}

				// gRPC client creation: pkg.NewXxxClient(ctx, ...)
				if strings.HasPrefix(methodName, "New") && strings.HasSuffix(methodName, "Client") && imp.style == "grpc" {
					if i < len(assignStmt.Lhs) {
						if ident, ok := assignStmt.Lhs[i].(*ast.Ident); ok {
							clientVars[ident.Name] = imp
						}
					}
				}

				// REST service creation: pkg.NewService(ctx, ...)
				if methodName == "NewService" && imp.style == "rest" {
					if i < len(assignStmt.Lhs) {
						if ident, ok := assignStmt.Lhs[i].(*ast.Ident); ok {
							restVars[ident.Name] = imp
						}
					}
				}
			}
			return true
		})

		// Now find calls on client variables: client.ListXxx(ctx, req)
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			methodName := sel.Sel.Name

			// gRPC client calls
			if ident, ok := sel.X.(*ast.Ident); ok {
				if imp, ok := clientVars[ident.Name]; ok {
					if isGCPAPIMethod(methodName) {
						perm := gcpMethodToPermission(imp.service, methodName)
						if perm != "" {
							details = append(details, PermissionDetail{
								Permission: perm,
								Service:    imp.service,
								Action:     methodName,
								SourceFile: fileName,
							})
						}
					}
				}
			}

			// REST chained calls: restVar.Resource.Method(...)
			if innerSel, ok := sel.X.(*ast.SelectorExpr); ok {
				resource := innerSel.Sel.Name
				if ident, ok := innerSel.X.(*ast.Ident); ok {
					if imp, ok := restVars[ident.Name]; ok {
						perm := gcpRESTToPermission(imp.service, resource, methodName)
						if perm != "" {
							details = append(details, PermissionDetail{
								Permission: perm,
								Service:    imp.service,
								Action:     resource + "." + methodName,
								SourceFile: fileName,
							})
						}
					}
				}
				// Deeper chains: restVar.Projects.Locations.Resource.Method(...)
				if innerSel2, ok := innerSel.X.(*ast.SelectorExpr); ok {
					_ = innerSel2
					// Walk up the chain to find the root variable
					rootVar, chain := walkSelectorChain(sel)
					if rootVar != "" {
						if imp, ok := restVars[rootVar]; ok {
							// The last element is the method, the rest form the resource path
							if len(chain) >= 2 {
								method := chain[len(chain)-1]
								// Find the meaningful resource (skip "Projects", "Locations")
								resourceName := findMeaningfulResource(chain[:len(chain)-1])
								perm := gcpRESTToPermission(imp.service, resourceName, method)
								if perm != "" {
									details = append(details, PermissionDetail{
										Permission: perm,
										Service:    imp.service,
										Action:     resourceName + "." + method,
										SourceFile: fileName,
									})
								}
							}
						}
					}
				}
			}

			return true
		})

		return false
	})

	return details
}

// walkSelectorChain walks a nested selector expression and returns the root variable
// name and the chain of selected names.
// e.g., svc.Projects.Locations.Keys.List -> ("svc", ["Projects", "Locations", "Keys", "List"])
func walkSelectorChain(sel *ast.SelectorExpr) (string, []string) {
	chain := []string{sel.Sel.Name}
	current := sel.X
	for {
		switch x := current.(type) {
		case *ast.SelectorExpr:
			chain = append([]string{x.Sel.Name}, chain...)
			current = x.X
		case *ast.Ident:
			return x.Name, chain
		case *ast.CallExpr:
			// Handle cases like svc.Method().Chain
			if s, ok := x.Fun.(*ast.SelectorExpr); ok {
				chain = append([]string{s.Sel.Name}, chain...)
				current = s.X
			} else {
				return "", chain
			}
		default:
			return "", chain
		}
	}
}

// findMeaningfulResource finds the meaningful resource name from a chain,
// skipping common parent levels like "Projects", "Locations".
func findMeaningfulResource(chain []string) string {
	skip := map[string]bool{
		"Projects": true, "Locations": true, "Regions": true,
		"Zones": true, "Global": true,
	}
	for i := len(chain) - 1; i >= 0; i-- {
		if !skip[chain[i]] {
			return chain[i]
		}
	}
	if len(chain) > 0 {
		return chain[len(chain)-1]
	}
	return ""
}

func isGCPAPIMethod(name string) bool {
	prefixes := []string{
		"List", "Get", "Create", "Delete", "Update", "Set",
		"Aggregated", "Search", "Test",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// gcpMethodToPermission maps a gRPC method to a GCP IAM permission.
func gcpMethodToPermission(service, method string) string {
	// gRPC methods: ListKeyRings -> cloudkms.keyRings.list
	// ListServiceAccounts -> iam.serviceAccounts.list
	// GetKeyRotationStatus -> cloudkms.cryptoKeys.get

	verb := ""
	resource := ""

	if strings.HasPrefix(method, "AggregatedList") {
		verb = "list"
		resource = strings.TrimPrefix(method, "AggregatedList")
	} else if strings.HasPrefix(method, "List") {
		verb = "list"
		resource = strings.TrimPrefix(method, "List")
	} else if strings.HasPrefix(method, "Get") {
		verb = "get"
		resource = strings.TrimPrefix(method, "Get")
		if resource == "" {
			resource = service
		}
	} else if strings.HasPrefix(method, "Create") {
		verb = "create"
		resource = strings.TrimPrefix(method, "Create")
	} else if strings.HasPrefix(method, "Delete") {
		verb = "delete"
		resource = strings.TrimPrefix(method, "Delete")
	} else if strings.HasPrefix(method, "Update") {
		verb = "update"
		resource = strings.TrimPrefix(method, "Update")
	} else if strings.HasPrefix(method, "Set") {
		verb = "update"
		resource = strings.TrimPrefix(method, "Set")
	} else if strings.HasPrefix(method, "Test") {
		verb = "get"
		resource = strings.TrimPrefix(method, "Test")
	} else if strings.HasPrefix(method, "Search") {
		verb = "list"
		resource = strings.TrimPrefix(method, "Search")
	} else {
		return ""
	}

	if resource == "" {
		return ""
	}

	// Convert PascalCase to camelCase
	resource = strings.ToLower(resource[:1]) + resource[1:]

	return service + "." + resource + "." + verb
}

// gcpRESTToPermission maps a REST-style call to a GCP IAM permission.
func gcpRESTToPermission(service, resource, method string) string {
	if resource == "" {
		return ""
	}
	verb := ""
	switch method {
	case "List", "AggregatedList", "Pages":
		verb = "list"
	case "Get", "Do":
		verb = "get"
	case "Create", "Insert":
		verb = "create"
	case "Delete":
		verb = "delete"
	case "Update", "Patch":
		verb = "update"
	case "GetIamPolicy":
		return service + "." + strings.ToLower(resource[:1]) + resource[1:] + ".getIamPolicy"
	case "SetIamPolicy":
		return service + "." + strings.ToLower(resource[:1]) + resource[1:] + ".setIamPolicy"
	default:
		verb = strings.ToLower(method)
	}

	// Convert PascalCase resource to camelCase
	res := strings.ToLower(resource[:1]) + resource[1:]
	return service + "." + res + "." + verb
}

// =============================================================================
// Azure Permission Extraction
// =============================================================================

func extractAzurePermissions(resourcesDir string) []PermissionDetail {
	var details []PermissionDetail
	files := listGoFiles(resourcesDir)

	for _, filePath := range files {
		fileName := filepath.Base(filePath)
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, filePath, nil, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to parse %s: %v\n", filePath, err)
			continue
		}

		// Build import map: alias -> ARM info
		azureImports := extractAzureImports(f)
		if len(azureImports) == 0 {
			continue
		}

		ast.Inspect(f, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok {
				return true
			}
			if fn.Body == nil {
				return true
			}

			// Track client variables: varName -> (ARM provider, resource type)
			clientVars := map[string]*azureClientInfo{}

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				assignStmt, ok := n.(*ast.AssignStmt)
				if !ok {
					return true
				}

				for i, rhs := range assignStmt.Rhs {
					call, ok := rhs.(*ast.CallExpr)
					if !ok {
						continue
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						continue
					}
					methodName := sel.Sel.Name

					// Pattern: pkg.NewXxxClient(...)
					if !strings.HasPrefix(methodName, "New") || !strings.HasSuffix(methodName, "Client") {
						continue
					}
					// Skip NewClientFactory
					if methodName == "NewClientFactory" {
						continue
					}

					pkgIdent, ok := sel.X.(*ast.Ident)
					if !ok {
						continue
					}

					imp, isAzureImport := azureImports[pkgIdent.Name]
					if !isAzureImport {
						continue
					}

					// Extract resource type from constructor name
					// e.g., NewVirtualMachinesClient -> VirtualMachines
					// NewClient -> "" (generic client, map via package)
					resourceType := strings.TrimPrefix(methodName, "New")
					resourceType = strings.TrimSuffix(resourceType, "Client")

					// For generic NewClient (no resource type), use package info
					if resourceType == "" {
						resourceType = azureResourceFromPackage(imp.pkgName)
					}

					if i < len(assignStmt.Lhs) {
						if ident, ok := assignStmt.Lhs[i].(*ast.Ident); ok {
							clientVars[ident.Name] = &azureClientInfo{
								armProvider:  imp.armProvider,
								resourceType: resourceType,
							}
						}
					}
				}
				return true
			})

			// Find pager/method calls on client variables
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				methodName := sel.Sel.Name

				ident, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}

				info, ok := clientVars[ident.Name]
				if !ok {
					return true
				}

				// Only care about read operations
				if isAzureReadMethod(methodName) {
					perm := azurePermission(info.armProvider, info.resourceType)
					details = append(details, PermissionDetail{
						Permission: perm,
						Service:    info.armProvider,
						Action:     methodName,
						SourceFile: fileName,
					})
				}

				return true
			})

			return false
		})
	}

	return details
}

type azureImportInfo struct {
	alias       string // import alias
	armProvider string // e.g., "Microsoft.Compute"
	pkgName     string // e.g., "armcompute"
}

type azureClientInfo struct {
	armProvider  string // e.g., "Microsoft.Compute"
	resourceType string // e.g., "VirtualMachines"
}

func extractAzureImports(f *ast.File) map[string]*azureImportInfo {
	result := map[string]*azureImportInfo{}
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if !strings.Contains(path, "azure-sdk-for-go/sdk/resourcemanager/") {
			continue
		}
		// e.g., github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7
		parts := strings.Split(path, "/")
		// Find the "resourcemanager" index
		rmIdx := -1
		for i, p := range parts {
			if p == "resourcemanager" {
				rmIdx = i
				break
			}
		}
		if rmIdx < 0 || rmIdx+2 >= len(parts) {
			continue
		}
		serviceName := parts[rmIdx+1] // e.g., "compute"
		armPkg := parts[rmIdx+2]      // e.g., "armcompute"

		alias := armPkg
		// Strip version suffix from alias if package has /v7 etc.
		if imp.Name != nil {
			alias = imp.Name.Name
		}

		armProvider := azureServiceToARM(serviceName)

		result[alias] = &azureImportInfo{
			alias:       alias,
			armProvider: armProvider,
			pkgName:     armPkg,
		}
	}
	return result
}

// azureServiceToARM maps Azure SDK service names to ARM provider namespaces.
var azureServiceToARMMap = map[string]string{
	"compute":               "Microsoft.Compute",
	"network":               "Microsoft.Network",
	"storage":               "Microsoft.Storage",
	"keyvault":              "Microsoft.KeyVault",
	"sql":                   "Microsoft.Sql",
	"postgresql":            "Microsoft.DBforPostgreSQL",
	"mysql":                 "Microsoft.DBforMySQL",
	"mariadb":               "Microsoft.DBforMariaDB",
	"cosmos":                "Microsoft.DocumentDB",
	"cosmosdb":              "Microsoft.DocumentDB",
	"redis":                 "Microsoft.Cache",
	"containerservice":      "Microsoft.ContainerService",
	"containerregistry":     "Microsoft.ContainerRegistry",
	"web":                   "Microsoft.Web",
	"monitor":               "Microsoft.Insights",
	"applicationinsights":   "Microsoft.Insights",
	"advisor":               "Microsoft.Advisor",
	"authorization":         "Microsoft.Authorization",
	"security":              "Microsoft.Security",
	"subscription":          "Microsoft.Resources",
	"resources":             "Microsoft.Resources",
	"search":                "Microsoft.Search",
	"servicebus":            "Microsoft.ServiceBus",
	"eventhub":              "Microsoft.EventHub",
	"iothub":                "Microsoft.Devices",
	"managedidentity":       "Microsoft.ManagedIdentity",
	"desktopvirtualization": "Microsoft.DesktopVirtualization",
	"appservice":            "Microsoft.Web",
	"databoxedge":           "Microsoft.DataBoxEdge",
	"logic":                 "Microsoft.Logic",
	"msi":                   "Microsoft.ManagedIdentity",
	"frontdoor":             "Microsoft.Network",
}

func azureServiceToARM(service string) string {
	if service == "" {
		return "Microsoft.Unknown"
	}
	if arm, ok := azureServiceToARMMap[service]; ok {
		return arm
	}
	// Default: Microsoft.<Capitalized service name>
	return "Microsoft." + strings.ToUpper(service[:1]) + service[1:]
}

// azurePermission constructs the RBAC permission string.
func azurePermission(armProvider, resourceType string) string {
	// Convert PascalCase to camelCase for the resource type
	rt := pascalToCamelCase(resourceType)
	return armProvider + "/" + rt + "/read"
}

func pascalToCamelCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// azureResourceFromPackage derives a resource type from an Azure ARM package name
// when the constructor is generic (NewClient).
// e.g., "armresources" -> "resources", "armsubscriptions" -> "subscriptions"
func azureResourceFromPackage(pkgName string) string {
	name := strings.TrimPrefix(pkgName, "arm")
	if name == "" {
		return pkgName
	}
	return name
}

func isAzureReadMethod(name string) bool {
	readMethods := []string{
		"NewListPager", "NewListAllPager", "NewListBySubscriptionPager",
		"NewListByResourceGroupPager", "Get",
		"NewListByServerPager", "NewListByAccountPager",
		"NewListByNamespacePager",
	}
	for _, m := range readMethods {
		if name == m {
			return true
		}
	}
	// Catch-all for other list pagers
	return strings.HasPrefix(name, "NewList") && strings.HasSuffix(name, "Pager")
}
