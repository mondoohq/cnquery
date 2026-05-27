// Parent template exercising the three import forms.

// Named import: pulls one of shared.bicep's types and its function.
import { sku, buildName } from './shared.bicep'

// Wildcard import: pulls everything shared.bicep exports under an alias.
import * as shared from './shared.bicep'

// Provider import: bare provider string, no local target file.
import 'az@2.0.0'

// Path-traversal import: points at a real, readable .bicep OUTSIDE the
// scanned root. targetFile() must be null — the provider only resolves to
// files the scan discovered and never reads an arbitrary path.
import { sku as escapeSku } from '../loops.bicep'

@description('Deployment location')
param location string = 'westeurope'
