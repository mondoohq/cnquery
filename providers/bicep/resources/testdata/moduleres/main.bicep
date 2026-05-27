// Parent template that composes a local module and a registry module.

@description('Deployment location')
param location string = 'westeurope'

// Local module reference — target() should resolve to modules/child.bicep.
module network './modules/child.bicep' = {
  name: 'network-deploy'
  params: {
    location: location
    vnetName: 'core-vnet'
  }
}

// Registry module reference — target() must be null.
module shared 'br:contoso.azurecr.io/bicep/modules/storage:v1' = {
  name: 'shared-storage'
  params: {
    location: location
  }
}
