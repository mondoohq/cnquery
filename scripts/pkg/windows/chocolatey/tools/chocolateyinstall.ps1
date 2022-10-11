$ErrorActionPreference = 'Stop'; # stop on all errors
$toolsDir   = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"

$cnquery_version = "6.19.0"
$url64      = "https://releases.mondoo.com/cnquery/${version}/cnquery_${version}_windows_amd64.zip"

$packageArgs = @{
  packageName   = $env:ChocolateyPackageName
  unzipLocation = $toolsDir
  url64bit      = $url64

  checksum64    = 'e58b0becdd0232a2a7e90b0e53ba40105d75eb2b33412f93309beda1c7293662'
  checksumType64= 'sha256' #default is checksumType
}

Install-ChocolateyZipPackage @packageArgs 


