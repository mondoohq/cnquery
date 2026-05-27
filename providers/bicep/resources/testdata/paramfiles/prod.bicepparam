// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

using './main.bicep'

param storageSku = 'Standard_LRS'
param location = resourceGroup().location
param adminPassword = readEnvironmentVariable('ADMIN_PW')
