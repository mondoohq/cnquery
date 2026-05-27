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

// Path-traversal reference — points at a real, readable .bicep that lives
// OUTSIDE the scanned root (../loops.bicep). target() must be null: the
// provider only resolves to files the scan discovered and never reads an
// arbitrary path, so an out-of-root reference can't disclose file contents.
module escape '../loops.bicep' = {
  name: 'escape-attempt'
}
