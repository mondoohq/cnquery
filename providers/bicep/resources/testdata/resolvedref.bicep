// Fixture for symbol-resolution on bicep.expression. Exercises a variable
// referencing a parameter, a resource whose name/location reference a
// parameter and a variable, a property/output referencing another resource by
// symbolic name, a module reference, a param typed by a user-defined type, and
// a reference to a built-in (resourceGroup) that must resolve to the empty
// reference kind.

type storageSku = 'Standard_LRS' | 'Premium_LRS'

param namePrefix string
param skuName storageSku

var saName = '${namePrefix}-sa'

resource sa 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: saName
  location: namePrefix
}

resource lock 'Microsoft.Authorization/locks@2020-05-01' = {
  name: 'sa-lock'
  scope: sa
  properties: {
    level: 'CanNotDelete'
  }
}

module net './network.bicep' = {
  name: 'netDeploy'
  scope: resourceGroup('rg-network')
}

output saId string = sa.id
output rgLoc string = resourceGroup().location
output netName string = net.name
