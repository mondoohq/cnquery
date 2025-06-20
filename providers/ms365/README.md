# ms365 Provider

## Prerequisites

- A registered Azure AD app configured with certificate-based authentication
  - The certificate needs to be in (RSA) format
  - The certificate digest algorithm should be sha256
  - The certificate should to be stored in a pfx format (see troubleshooting if you only have pem)
- PowerShell

__NOTE__

At the moment some of the resources within this provider requires the use of PowerShell.  Please make sure you have `pwsh` is in your path. For example `pwsh -c 'Get-Date'` should return a timestamp.

Also it's recommended that you use certificate based authentication with this provider, especially with the resources that need to use powershell, this is due to the way those cmdlets authenticate.  Certificate auth is more reliable when testing.

### Mac
```shell
brew install --cask powershell
```
### Linux
Please refer to the [documentation](https://learn.microsoft.com/en-us/powershell/scripting/install/installing-powershell-on-linux) based on your distrubution.

## Authenticating
You will need to provide your app's client and tenant id as well certificate.
```
export MS365_CLIENT_ID='your-client-id'
export MS365_TENANT_ID='your-tenant-id'
export MS365_CERTIFICATE_PATH='certificate.pfx'
```
```shell
cnquery shell ms365 --certificate-path ${MS365_CERTIFICATE_PATH} --tenant-id ${MS365_TENANT_ID} --client-id ${MS365_CLIENT_ID}
```

## Examples
TODO: Add safe examples that should work regardless of tenant

## Troubleshooting

### Convert pem to pfx
```shell
openssl pkcs12 -export -out certificate.pfx -inkey privatekey.key -in certificate.pem
```

### Insufficient Privileges

```
x unable to create runtime for asset error="rpc error: code = Unknown desc = authentication failed: Insufficient privileges to complete the operation." asset=
FTL could not find an asset that we can connect to
```
This is a catch all so start with double check your certificate


### Unknown digest
```
pkcs12: unknown digest algorithm: 2.16.840.1.101.3.4.2.1"
```
Your certificate is probably not sha256, double check 
```
openssl x509 -in ./certificate.pem -noout -text | grep "Signature Algorithm"
```
```
Signature Algorithm: sha512WithRSAEncryption
```