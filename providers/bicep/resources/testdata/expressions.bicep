targetScope = 'resourceGroup'

@description('Resource name prefix')
param prefix string = 'app'

@secure()
param adminPassword string

var location = resourceGroup().location
var storageName = '${prefix}-sa'
var tier = prefix == 'prod' ? 'Premium' : 'Standard'
var tags = ['env', 'owner']
var literalName = 'fixed-name'

resource sa 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: storageName
  location: location
}

output saId string = sa.id
output region string = resourceGroup().location
output adminEcho string = '${adminPassword}'
output staticOut string = 'hello'
