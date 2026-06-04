resource sa 'Microsoft.Storage/storageAccounts@2023-05-01' = {
  name: 'examplestorage'
  location: 'eastus'
  properties: {
    supportsHttpsTrafficOnly: true
  }
}
