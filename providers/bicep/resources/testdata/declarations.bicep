// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

metadata description = 'My template'
metadata author = 'platform-team'

import { typeA, funcB } from './shared.bicep'
import * as shared from './shared.bicep'
import 'az@2.0.0'

@description('Storage SKU')
@export()
type sku = 'Standard_LRS' | 'Premium_LRS'

type cfg = {
  name: string
  tier: sku
}

@description('Build a resource name')
func buildName(prefix string, idx int) string => '${prefix}-${idx}'

param location string = 'eastus'
