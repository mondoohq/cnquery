// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Fixture exercising the resource/module expression-tree accessors:
// interpolated name, literal location, function-call location, a
// conditional resource, and a module with a scope expression and a
// condition.

@description('Resource name prefix')
param prefix string = 'demo'

@description('Whether to deploy the optional account')
param deployFlag bool = true

// name interpolates a parameter, location is a hardcoded literal
resource interpName 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: '${prefix}-sa'
  location: 'eastus'
  sku: {
    name: 'Standard_LRS'
  }
  kind: 'StorageV2'
}

// location is derived from a function call; conditional deployment
resource fnLoc 'Microsoft.Storage/storageAccounts@2023-01-01' = if (deployFlag) {
  name: 'fnlocsa'
  location: resourceGroup().location
  sku: {
    name: 'Standard_LRS'
  }
  kind: 'StorageV2'
}

// module with a scope expression and a deployment condition
module net './modules/network.bicep' = if (deployFlag) {
  name: 'networkDeploy'
  scope: resourceGroup('rg-network')
  params: {
    location: 'eastus'
  }
}
