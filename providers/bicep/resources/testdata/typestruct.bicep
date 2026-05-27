// Fixture for user-defined type decomposition. Exercises a union type, an
// object type whose property names another user-defined type, an @export()ed
// object type, an @discriminator() tagged-union type, an alias type, plus
// parameters and an output that name a user-defined type vs a built-in.

type sku = 'Standard_LRS' | 'Premium_LRS'

type cfg = {
  name: string
  tier: sku
}

@export()
type exportedCfg = {
  id: string
  optional: bool?
}

@discriminator('kind')
type shape = { kind: 'circle', radius: int } | { kind: 'square', side: int }

type skuAlias = sku

param appCfg cfg
param appName string

output usedSku sku = 'Standard_LRS'
