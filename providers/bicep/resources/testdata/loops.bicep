// Test fixture for Bicep `for`-loop modeling.
param location string = 'eastus'

var storageNames = [
  'stga'
  'stgb'
]

// Looped variable using range(...)
var itemNames = [for i in range(0, 3): 'item-${i}']

// Looped resource using the indexed (name, i) form with an object body.
resource sas 'Microsoft.Storage/storageAccounts@2023-01-01' = [for (name, i) in storageNames: {
  name: name
  location: location
  sku: {
    name: 'Standard_LRS'
  }
}]

// Non-looped resource to prove isLoop is false.
resource kv 'Microsoft.KeyVault/vaults@2023-07-01' = {
  name: 'mykeyvault'
  location: location
}

// Looped module.
module stamps 'stamp.bicep' = [for sku in storageNames: {
  name: 'stamp-${sku}'
  params: {
    skuName: sku
  }
}]

// Looped output.
output ids array = [for sa in sas: sa.id]

// Non-looped output.
output region string = location
