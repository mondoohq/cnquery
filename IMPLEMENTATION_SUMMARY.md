# MS365 Identity Security Defaults Policy Implementation

## Overview
This implementation extends the MS365 provider with a new resource for accessing the Identity Security Defaults Enforcement Policy through the Microsoft Graph API.

## Changes Made

### 1. Resource Definition (.lr file)
- Extended `microsoft.identityAndAccess` with `identityAndSignIn()` method
- Added new resource hierarchy:
  - `microsoft.identityAndAccess.identityAndSignIn`
  - `microsoft.identityAndAccess.identityAndSignIn.policies`
  - `microsoft.identityAndAccess.identityAndSignIn.policies.identitySecurityDefaultsEnforcementPolicy`

### 2. Resource Fields
The `identitySecurityDefaultsEnforcementPolicy` resource includes:
- `id` (string): The unique identifier for the policy
- `displayName` (string): The display name for the policy
- `description` (string): The description for the policy
- `isEnabled` (bool): Whether Azure Entra ID security defaults is enabled

### 3. Implementation Details
- Uses Microsoft Graph API endpoint: `/policies/identitySecurityDefaultsEnforcementPolicy`
- Implements proper error handling with `transformError()`
- Follows existing MS365 provider patterns for resource creation
- Generates proper resource IDs and handles null values safely

### 4. Files Modified
- `providers/ms365/resources/ms365.lr` - Resource definitions
- `providers/ms365/resources/identity_and_access.go` - Implementation logic
- `providers/ms365/resources/ms365.lr.go` - Generated Go structs (auto-generated)
- `providers/ms365/resources/ms365.lr.manifest.yaml` - Resource manifest (auto-generated)

## Usage Examples

### Basic Query
```mql
microsoft.identityAndAccess.identityAndSignIn.policies.identitySecurityDefaultsEnforcementPolicy {
  id
  displayName
  description
  isEnabled
}
```

### Security Check
```mql
microsoft.identityAndAccess.identityAndSignIn.policies.identitySecurityDefaultsEnforcementPolicy.isEnabled == true
```

## API Reference
- **Graph API**: https://learn.microsoft.com/en-us/graph/api/resources/identitysecuritydefaultsenforcementpolicy?view=graph-rest-1.0
- **PowerShell Alternative**: `Get-MgPolicyIdentitySecurityDefaultEnforcementPolicy`

## Testing
- Provider builds successfully without compilation errors
- Binary generated at `providers/ms365/dist/ms365`
- Resource manifest updated automatically
- Ready for integration testing with actual MS365 tenant

## Branch
Created on branch: `feature/ms365-identity-security-defaults-policy`
