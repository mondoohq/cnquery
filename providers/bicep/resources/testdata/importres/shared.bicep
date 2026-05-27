// Shared library exporting two types and a function, imported by main.bicep.

@export()
type sku = 'Standard_LRS' | 'Premium_LRS'

@export()
type tier = 'hot' | 'cool' | 'archive'

@export()
func buildName(prefix string, suffix string) string => '${prefix}-${suffix}'
