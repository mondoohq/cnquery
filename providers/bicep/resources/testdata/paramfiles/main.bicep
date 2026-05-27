// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

@description('Storage account SKU')
param storageSku string = 'Standard_LRS'

@description('Deployment location')
param location string = resourceGroup().location

@secure()
param adminPassword string

resource sa 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: 'examplestorage'
  location: location
  sku: {
    name: storageSku
  }
}
