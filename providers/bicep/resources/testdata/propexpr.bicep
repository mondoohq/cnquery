// Fixture for bicep.resource.propertyExpressions and
// bicep.module.paramExpressions. A storage account whose nested properties
// (a) deeply interpolate a @secure parameter, (b) reference another resource
// by symbolic name, (c) call a built-in function, and (d) hardcode a literal,
// plus a module whose params forward a parameter.

@secure()
param adminPassword string

param location string = 'eastus'

resource kv 'Microsoft.KeyVault/vaults@2023-07-01' existing = {
  name: 'my-keyvault'
}

resource sa 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: 'mystorage'
  location: resourceGroup().location
  properties: {
    accessTier: 'Hot'
    derivedLocation: resourceGroup().location
    encryption: {
      keyVaultProperties: {
        keyVaultUri: kv.properties.vaultUri
      }
    }
    adminCredentials: {
      password: '${adminPassword}'
    }
  }
}

module net './network.bicep' = {
  name: 'netDeploy'
  params: {
    secret: adminPassword
    region: location
  }
}
