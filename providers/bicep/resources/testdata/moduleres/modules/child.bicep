// Child module deployed by the parent's `network` module declaration.

@description('Location for the virtual network')
param location string

@description('Name of the virtual network')
param vnetName string

resource vnet 'Microsoft.Network/virtualNetworks@2023-05-01' = {
  name: vnetName
  location: location
  properties: {
    addressSpace: {
      addressPrefixes: [
        '10.0.0.0/16'
      ]
    }
  }
}

output vnetId string = vnet.id
