// A storage account with a nested blobServices child and a deeper
// containers grandchild (two levels of nesting).
resource sa 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: 'mystorage'
  location: 'eastus'
  sku: {
    name: 'Standard_LRS'
  }
  resource blob 'blobServices' = {
    name: 'default'
    properties: {
      isVersioningEnabled: true
    }
    resource container 'containers@2023-01-01' = {
      name: 'data'
      properties: {
        publicAccess: 'None'
      }
    }
  }
}

// A management lock that targets a different scope via the `scope` keyword.
resource lock 'Microsoft.Authorization/locks@2020-05-01' = {
  name: 'no-delete'
  scope: sa
  properties: {
    level: 'CanNotDelete'
  }
}

// A normal flat resource with no nesting and no scope.
resource kv 'Microsoft.KeyVault/vaults@2023-07-01' = {
  name: 'mykeyvault'
  location: 'westus'
}
